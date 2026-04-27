package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/lustertools/lusterpass/internal/auth"
	"github.com/lustertools/lusterpass/internal/cache"
	"github.com/lustertools/lusterpass/internal/config"
)

// stdoutIsTerminal exists as a package var so tests can override it. In normal
// operation it returns true when lusterpass env's stdout is attached to an
// interactive terminal — i.e. the user typed the bare command at a prompt and
// the output would land in their scrollback (or in an AI agent's transcript).
var stdoutIsTerminal = func() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

var envProfile string

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Output export lines from cached secrets and config vars",
	Long:  "Output 'export VAR=value' lines for shell eval, drawing from the encrypted local cache.\n\nOmit --profile to emit only the common section. Pass --profile to additionally\noverlay an environment-specific profile (e.g., dev, staging, prod).",
	RunE: func(cmd *cobra.Command, args []string) error {
		if stdoutIsTerminal() {
			return fmt.Errorf(`refusing to print secret values directly to a terminal.
This would expose values to your terminal scrollback, your shell history,
or an AI agent's transcript. Use one of the safe forms instead:

  eval "$(lusterpass env%s)"            # load into the current shell
  lusterpass exec%s -- <command>        # run a single command with secrets`, profileSuffix(envProfile), profileSuffix(envProfile))
		}

		cfg, err := config.Load(configFile)
		if err != nil {
			return err
		}

		account, err := resolveAccount(cfg)
		if err != nil {
			return err
		}

		resolved, err := cfg.ResolveProfile(envProfile)
		if err != nil {
			return err
		}

		token, err := auth.ResolveTokenForAccount(account)
		if err != nil {
			return fmt.Errorf("no access token for cache decryption: %w", err)
		}

		cachePath := cache.CachePath(account, cfg.Project, cfg.CacheKey(envProfile))
		secrets, err := cache.Read(cachePath, token)
		if err != nil {
			pullHint := "lusterpass pull"
			if envProfile != "" {
				pullHint = fmt.Sprintf("lusterpass pull --profile %s", envProfile)
			}
			return fmt.Errorf("reading cache (run '%s' first): %w", pullHint, err)
		}

		fmt.Print(formatExports(resolved.Vars, secrets))
		return nil
	},
}

// formatExports produces sorted export lines for shell eval.
func formatExports(vars, secrets map[string]string) string {
	merged := make(map[string]string)
	for k, v := range vars {
		merged[k] = v
	}
	// Secrets override vars if same key
	for k, v := range secrets {
		merged[k] = v
	}

	keys := make([]string, 0, len(merged))
	for k := range merged {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		v := shellEscape(merged[k])
		fmt.Fprintf(&b, "export %s=%s\n", k, v)
	}
	return b.String()
}

// shellEscape wraps value in single quotes for safe shell eval.
// Single quotes inside the value are handled with the '\'' trick.
func shellEscape(s string) string {
	escaped := strings.ReplaceAll(s, "'", "'\\''")
	return "'" + escaped + "'"
}

func profileSuffix(profile string) string {
	if profile == "" {
		return ""
	}
	return " --profile " + profile
}

func init() {
	envCmd.Flags().StringVar(&envProfile, "profile", "", "Environment profile (e.g., dev, staging, prod); omit to emit only the common section")
	rootCmd.AddCommand(envCmd)
}
