package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/lustertools/lusterpass/internal/auth"
	"github.com/lustertools/lusterpass/internal/cache"
	"github.com/lustertools/lusterpass/internal/config"
)

var envProfile string

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Output export lines from cached secrets and config vars",
	RunE: func(cmd *cobra.Command, args []string) error {
		if envProfile == "" {
			return fmt.Errorf("--profile is required")
		}

		cfg, err := config.Load(configFile)
		if err != nil {
			return err
		}

		account, err := resolveAccount(cfg)
		if err != nil {
			return err
		}

		resolved := cfg.ResolveProfile(envProfile)

		// Load cached secrets
		token, err := auth.ResolveTokenForAccount(account)
		if err != nil {
			return fmt.Errorf("no access token for cache decryption: %w", err)
		}

		cachePath := cache.CachePath(account, cfg.Project, envProfile)
		secrets, err := cache.Read(cachePath, token)
		if err != nil {
			return fmt.Errorf("reading cache (run 'lusterpass pull --profile %s' first): %w", envProfile, err)
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

func init() {
	envCmd.Flags().StringVar(&envProfile, "profile", "", "Environment profile (e.g., dev, staging, prod)")
	rootCmd.AddCommand(envCmd)
}
