package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/lustertools/lusterpass/internal/auth"
	"github.com/lustertools/lusterpass/internal/cache"
	"github.com/lustertools/lusterpass/internal/config"
)

var execProfile string

var execCmd = &cobra.Command{
	Use:   "exec [--profile X] -- <command> [args...]",
	Short: "Run a command with cached secrets in its environment, never exposed to the parent shell",
	Long: `Run a command with the resolved secrets and config vars set in its environment,
without ever printing values to stdout, your shell history, or an AI agent's transcript.

On Unix, lusterpass replaces its own process image with the target command via
execve(2) — there is no parent lusterpass process during the run, so signals
and exit codes pass through naturally and there is zero memory overhead.

On Windows, lusterpass forks the target as a child process, forwards SIGINT
and SIGTERM, and exits with the child's exit code.

Examples:
  lusterpass exec -- ./run-migrations.sh
  lusterpass exec --profile dev -- npm test
  lusterpass exec -- python train.py --gpu`,
	Args:               cobra.MinimumNArgs(1),
	DisableFlagParsing: false,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(configFile)
		if err != nil {
			return err
		}

		account, err := resolveAccount(cfg)
		if err != nil {
			return err
		}

		resolved, err := cfg.ResolveProfile(execProfile)
		if err != nil {
			return err
		}

		token, err := auth.ResolveTokenForAccount(account)
		if err != nil {
			return fmt.Errorf("no access token for cache decryption: %w", err)
		}

		cachePath := cache.CachePath(account, cfg.Project, cfg.CacheKey(execProfile))
		secrets, err := cache.Read(cachePath, token)
		if err != nil {
			pullHint := "lusterpass pull"
			if execProfile != "" {
				pullHint = fmt.Sprintf("lusterpass pull --profile %s", execProfile)
			}
			return fmt.Errorf("reading cache (run '%s' first): %w", pullHint, err)
		}

		envv := buildExecEnv(resolved.Vars, secrets)

		return runExec(args, envv)
	},
}

// buildExecEnv merges shell env + config vars + decrypted secrets in that
// precedence order (later layers override earlier on key collision) and
// returns the result as a "KEY=VALUE" slice ready for execve / Cmd.Env.
func buildExecEnv(vars, secrets map[string]string) []string {
	merged := make(map[string]string)

	for _, kv := range os.Environ() {
		i := strings.IndexByte(kv, '=')
		if i < 0 {
			continue
		}
		merged[kv[:i]] = kv[i+1:]
	}
	for k, v := range vars {
		merged[k] = v
	}
	for k, v := range secrets {
		merged[k] = v
	}

	out := make([]string, 0, len(merged))
	for k, v := range merged {
		out = append(out, k+"="+v)
	}
	return out
}

func init() {
	execCmd.Flags().StringVar(&execProfile, "profile", "", "Environment profile (e.g., dev, staging, prod); omit for common-only")
	execCmd.Flags().SetInterspersed(false)
	rootCmd.AddCommand(execCmd)
}
