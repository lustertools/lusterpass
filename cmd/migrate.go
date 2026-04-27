package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// secretKeyPatterns matches env var names that likely hold secrets.
var secretKeyPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(^|_)(PASSWORD|PASSWD|PASS)(_|$)`),
	regexp.MustCompile(`(?i)(^|_)(SECRET)(_|$)`),
	regexp.MustCompile(`(?i)(^|_)(TOKEN)(_|$)`),
	regexp.MustCompile(`(?i)(^|_)(API_?KEY)(_|$)`),
	regexp.MustCompile(`(?i)(^|_)(PRIVATE_?KEY)(_|$)`),
	regexp.MustCompile(`(?i)(^|_)(ACCESS_?KEY)(_|$)`),
	regexp.MustCompile(`(?i)(^|_)(SECRET_?KEY)(_|$)`),
	regexp.MustCompile(`(?i)(^|_)(AUTH)(_|$)`),
	regexp.MustCompile(`(?i)(^|_)(CREDENTIAL)S?(_|$)`),
	regexp.MustCompile(`(?i)(^|_)(SIGNING_?KEY)(_|$)`),
	regexp.MustCompile(`(?i)(^|_)(ENCRYPTION_?KEY)(_|$)`),
	regexp.MustCompile(`(?i)(^|_)(CONN(ECTION)?_?STR(ING)?)(_|$)`),
	regexp.MustCompile(`(?i)(^|_)(DSN)(_|$)`),
	regexp.MustCompile(`(?i)(^|_)(DATABASE_?URL)(_|$)`),
}

// secretValuePatterns matches values that look like secrets.
var secretValuePatterns = []*regexp.Regexp{
	regexp.MustCompile(`^sk[-_]`),            // OpenAI, Stripe style
	regexp.MustCompile(`^pk[-_]`),            // public/private key prefixes
	regexp.MustCompile(`^ghp_`),              // GitHub PAT
	regexp.MustCompile(`^gho_`),              // GitHub OAuth
	regexp.MustCompile(`^github_pat_`),       // GitHub fine-grained PAT
	regexp.MustCompile(`^xox[bpras]-`),       // Slack tokens
	regexp.MustCompile(`^AKIA`),              // AWS access key
	regexp.MustCompile(`^eyJ[A-Za-z0-9]`),   // JWT
	regexp.MustCompile(`^[A-Za-z0-9+/]{40,}={0,2}$`), // base64 blobs
	regexp.MustCompile(`^[0-9a-f]{32,}$`),    // hex strings (API keys)
}

type envEntry struct {
	Key   string
	Value string
}

func looksLikeSecret(key, value string) bool {
	for _, p := range secretKeyPatterns {
		if p.MatchString(key) {
			return true
		}
	}
	for _, p := range secretValuePatterns {
		if p.MatchString(value) {
			return true
		}
	}
	// Long random-looking values (high entropy heuristic)
	// Exclude common non-secret patterns: URLs, file paths, email addresses
	if len(value) >= 20 && hasMixedCharClasses(value) && !looksLikePlainValue(value) {
		return true
	}
	return false
}

func hasMixedCharClasses(s string) bool {
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, c := range s {
		switch {
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= '0' && c <= '9':
			hasDigit = true
		default:
			hasSpecial = true
		}
	}
	classes := 0
	if hasUpper { classes++ }
	if hasLower { classes++ }
	if hasDigit { classes++ }
	if hasSpecial { classes++ }
	return classes >= 3
}

var plainValuePatterns = []*regexp.Regexp{
	regexp.MustCompile(`^https?://`),          // URLs
	regexp.MustCompile(`^/[a-zA-Z]`),          // Unix paths
	regexp.MustCompile(`^[a-zA-Z]:\\`),        // Windows paths
	regexp.MustCompile(`^[^@]+@[^@]+\.[^@]+`), // Email addresses
	regexp.MustCompile(`^[a-z]+://`),          // URI schemes
}

func looksLikePlainValue(value string) bool {
	for _, p := range plainValuePatterns {
		if p.MatchString(value) {
			return true
		}
	}
	return false
}

// parseExportLine extracts KEY and VALUE from an export line.
// Handles: export KEY=VALUE, export KEY="VALUE", export KEY='VALUE'
var exportRe = regexp.MustCompile(`^\s*export\s+([A-Za-z_][A-Za-z0-9_]*)=(.*)$`)

func parseExportLine(line string) (string, string, bool) {
	m := exportRe.FindStringSubmatch(line)
	if m == nil {
		return "", "", false
	}
	key := m[1]
	value := m[2]

	// Strip surrounding quotes
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
		}
	}

	return key, value, true
}

func toRefName(envVar, project string) string {
	// Convert ENV_VAR_NAME to ref-name--project
	ref := strings.ToLower(strings.ReplaceAll(envVar, "_", "-"))
	return ref + "--" + project
}

var migrateCmd = &cobra.Command{
	Use:   "migrate [file]",
	Short: "Generate .lusterpass.yaml and onboarding script from an existing .envrc or shell rc file",
	Long: `Scan an existing file containing export statements (e.g., .envrc, .zshrc, .bashrc),
auto-detect which values are secrets vs plain vars, and generate:

  1. .lusterpass.yaml — config with common vars/secrets (plus a commented-out profiles example)
  2. onboard-secrets.sh — editable script to enrol secrets into Bitwarden

Review and edit the generated files, then run onboard-secrets.sh to migrate.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		srcPath := args[0]
		project, _ := cmd.Flags().GetString("project")
		if project == "" {
			// Derive from current directory name
			cwd, _ := os.Getwd()
			project = filepath.Base(cwd)
		}

		outDir, _ := cmd.Flags().GetString("out")
		if outDir == "" {
			outDir = "."
		}

		f, err := os.Open(srcPath)
		if err != nil {
			return fmt.Errorf("opening %s: %w", srcPath, err)
		}
		defer f.Close()

		var vars []envEntry
		var secrets []envEntry
		var skipped []string

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())

			// Skip comments, blank lines, eval lines
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if strings.Contains(line, "eval ") || strings.Contains(line, "$(") {
				skipped = append(skipped, line)
				continue
			}

			key, value, ok := parseExportLine(line)
			if !ok {
				skipped = append(skipped, line)
				continue
			}

			entry := envEntry{Key: key, Value: value}
			if looksLikeSecret(key, value) {
				secrets = append(secrets, entry)
			} else {
				vars = append(vars, entry)
			}
		}

		sort.Slice(vars, func(i, j int) bool { return vars[i].Key < vars[j].Key })
		sort.Slice(secrets, func(i, j int) bool { return secrets[i].Key < secrets[j].Key })

		// Generate .lusterpass.yaml
		yamlPath := filepath.Join(outDir, ".lusterpass.yaml")
		if err := writeMigratedYAML(yamlPath, project, vars, secrets); err != nil {
			return err
		}

		// Generate onboard-secrets.sh
		scriptPath := filepath.Join(outDir, "onboard-secrets.sh")
		if err := writeOnboardScript(scriptPath, project, secrets); err != nil {
			return err
		}

		// Report
		fmt.Println("Migration analysis complete:")
		fmt.Printf("  Source:        %s\n", srcPath)
		fmt.Printf("  Plain vars:   %d\n", len(vars))
		fmt.Printf("  Secrets:      %d\n", len(secrets))
		if len(skipped) > 0 {
			fmt.Printf("  Skipped:      %d (non-export lines)\n", len(skipped))
		}
		fmt.Println()
		fmt.Printf("Generated files:\n")
		fmt.Printf("  %s  — lusterpass config (review and commit to git)\n", yamlPath)
		fmt.Printf("  %s    — secret onboarding script (edit, then run)\n", scriptPath)
		fmt.Println()

		fmt.Println("Next steps:")
		fmt.Println()
		fmt.Println("  1. Review .lusterpass.yaml")
		fmt.Println("     - Check that vars vs secrets are classified correctly")
		fmt.Println("     - Adjust reference names in secrets if needed")
		fmt.Println("     - Uncomment the profiles: section if you need dev/staging/prod separation")
		fmt.Println()
		fmt.Println("  2. Edit onboard-secrets.sh")
		fmt.Println("     - Review each secret's Bitwarden reference name (--ref)")
		fmt.Println("     - The naming convention is: <purpose>--<project>[--<env>]")
		fmt.Println("     - Remove any secrets that are already in Bitwarden")
		fmt.Println()
		fmt.Printf("  3. Run the onboarding script:\n")
		fmt.Printf("     chmod +x %s\n", scriptPath)
		fmt.Printf("     %s                                   # uses cached org ID\n", scriptPath)
		fmt.Printf("     %s --org YOUR_ORG_ID                 # or override org ID\n", scriptPath)
		fmt.Println()
		fmt.Println("  4. Pull and verify:")
		fmt.Println("     lusterpass pull")
		fmt.Println("     lusterpass env")
		fmt.Println()
		fmt.Println("  5. Replace your old .envrc with:")
		fmt.Println("     eval \"$(lusterpass env)\"")
		fmt.Println()
		fmt.Println("  6. Delete onboard-secrets.sh (contains plain secret values!)")

		return nil
	},
}

func writeMigratedYAML(path, project string, vars, secrets []envEntry) error {
	var b strings.Builder

	b.WriteString("# Generated by: lusterpass migrate\n")
	b.WriteString("# Review and adjust before committing.\n")
	b.WriteString("# This file contains NO secret values — safe to commit to git.\n\n")

	fmt.Fprintf(&b, "project: %s\n\n", project)

	b.WriteString("# Common: shared across all environments\n")
	b.WriteString("common:\n")

	if len(vars) > 0 {
		b.WriteString("  vars:\n")
		for _, v := range vars {
			fmt.Fprintf(&b, "    %s: %s\n", v.Key, yamlValue(v.Value))
		}
	}

	if len(secrets) > 0 {
		b.WriteString("  secrets:\n")
		b.WriteString("    # Format: ENV_VAR_NAME: bitwarden-reference-name\n")
		for _, s := range secrets {
			ref := toRefName(s.Key, project)
			fmt.Fprintf(&b, "    %s: %s\n", s.Key, ref)
		}
	}

	b.WriteString("\n# Per-environment profiles (optional)\n")
	b.WriteString("# Uncomment and customize if you need to differentiate dev / staging / prod.\n")
	b.WriteString("# Profile values override common values for the same key. Use:\n")
	b.WriteString("#   lusterpass pull --profile dev\n")
	b.WriteString("#   lusterpass exec --profile dev -- ./your-script.sh\n")
	b.WriteString("#\n")
	b.WriteString("# profiles:\n")
	b.WriteString("#   dev:\n")
	b.WriteString("#     vars:\n")
	b.WriteString("#       LOG_LEVEL: debug\n")

	if len(secrets) > 0 {
		b.WriteString("#     secrets:\n")
		for _, s := range secrets {
			ref := toRefName(s.Key, project)
			fmt.Fprintf(&b, "#       %s: %s--dev\n", s.Key, ref)
		}
	}

	return os.WriteFile(path, []byte(b.String()), 0644)
}

func writeOnboardScript(path, project string, secrets []envEntry) error {
	var b strings.Builder

	b.WriteString("#!/usr/bin/env bash\n")
	b.WriteString("set -euo pipefail\n\n")
	b.WriteString("# Secret onboarding script generated by: lusterpass migrate\n")
	b.WriteString("# REVIEW CAREFULLY before running — this file contains plain secret values!\n")
	b.WriteString("#\n")
	b.WriteString("# Usage:\n")
	b.WriteString("#   chmod +x onboard-secrets.sh\n")
	b.WriteString("#   ./onboard-secrets.sh                              # uses cached org ID from 'lusterpass login'\n")
	b.WriteString("#   ./onboard-secrets.sh --org YOUR_ORG_ID            # override org ID\n")
	b.WriteString("#   ./onboard-secrets.sh --project certificates       # use a different BW project\n")
	b.WriteString("#\n")
	b.WriteString("# For each secret below, verify:\n")
	b.WriteString("#   --ref   The Bitwarden reference name (naming: <purpose>--<project>[--<env>])\n")
	b.WriteString("#   --value The secret value\n")
	b.WriteString("#   --note  Optional description\n")
	b.WriteString("#\n")
	b.WriteString("# Delete this file after running — it contains sensitive values!\n\n")

	b.WriteString("ORG_FLAG=\"\"\n")
	b.WriteString("BW_PROJECT=\"credentials\"\n\n")

	b.WriteString("while [[ $# -gt 0 ]]; do\n")
	b.WriteString("  case $1 in\n")
	b.WriteString("    --org) ORG_FLAG=\"--org $2\"; shift 2 ;;\n")
	b.WriteString("    --project) BW_PROJECT=\"$2\"; shift 2 ;;\n")
	b.WriteString("    *) echo \"Unknown flag: $1\"; exit 1 ;;\n")
	b.WriteString("  esac\n")
	b.WriteString("done\n\n")

	b.WriteString("enrol_secret() {\n")
	b.WriteString("  local ref=\"$1\" value=\"$2\" note=\"${3:-}\"\n")
	b.WriteString("  echo \"  Enrolling: $ref\"\n")
	b.WriteString("  lusterpass enrol $ORG_FLAG --ref \"$ref\" --value \"$value\" --note \"$note\" --project-name \"$BW_PROJECT\"\n")
	b.WriteString("}\n\n")

	b.WriteString("echo \"\"\n")
	b.WriteString("echo \"Onboarding secrets to Bitwarden...\"\n")
	b.WriteString("echo \"\"\n\n")

	if len(secrets) == 0 {
		b.WriteString("echo \"No secrets to onboard.\"\n")
	} else {
		b.WriteString(fmt.Sprintf("TOTAL=%d\n", len(secrets)))
		b.WriteString("COUNT=0\n\n")

		for _, s := range secrets {
			ref := toRefName(s.Key, project)
			// Escape single quotes in value for shell safety
			escapedVal := strings.ReplaceAll(s.Value, "'", "'\\''")
			fmt.Fprintf(&b, "enrol_secret '%s' '%s' 'Migrated from .envrc: %s'\n", ref, escapedVal, s.Key)
			b.WriteString("COUNT=$((COUNT + 1))\n")
			b.WriteString("echo \"  [$COUNT/$TOTAL]\"\n\n")
		}

		b.WriteString("echo \"\"\n")
		b.WriteString("echo \"Done! $COUNT secrets enrolled.\"\n")
		b.WriteString("echo \"\"\n")
		b.WriteString("echo \"Next:\"\n")
		b.WriteString("echo \"  lusterpass pull --profile dev\"\n")
		b.WriteString("echo \"  lusterpass env --profile dev\"\n")
		b.WriteString("echo \"\"\n")
		b.WriteString("echo \"Then delete this script — it contains plain secret values!\"\n")
	}

	return os.WriteFile(path, []byte(b.String()), 0755)
}

// yamlValue returns a YAML-safe string value.
func yamlValue(s string) string {
	// If it contains special chars, quote it
	if strings.ContainsAny(s, ":{}[]&*?|>!%@`\"'\\#,") ||
		strings.HasPrefix(s, " ") || strings.HasSuffix(s, " ") ||
		s == "true" || s == "false" || s == "null" || s == "yes" || s == "no" {
		return fmt.Sprintf("%q", s)
	}
	return s
}

func init() {
	migrateCmd.Flags().String("project", "", "Project name (default: current directory name)")
	migrateCmd.Flags().String("out", "", "Output directory (default: current directory)")
	rootCmd.AddCommand(migrateCmd)
}
