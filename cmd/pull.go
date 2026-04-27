package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/lustertools/lusterpass/internal/auth"
	"github.com/lustertools/lusterpass/internal/bitwarden"
	"github.com/lustertools/lusterpass/internal/cache"
	"github.com/lustertools/lusterpass/internal/config"
)

var pullProfile string

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Fetch secrets from Bitwarden and cache locally",
	Long:  "Fetch secrets defined in .lusterpass.yaml from Bitwarden and cache locally.\n\nOmit --profile to load only the common section. Pass --profile to additionally\noverlay an environment-specific profile (e.g., dev, staging, prod).",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(configFile)
		if err != nil {
			return err
		}

		account, err := resolveAccount(cfg)
		if err != nil {
			return err
		}

		resolved, err := cfg.ResolveProfile(pullProfile)
		if err != nil {
			return err
		}

		if len(resolved.Secrets) == 0 {
			if pullProfile == "" {
				fmt.Println("No secrets to pull (no secrets defined in common section).")
			} else {
				fmt.Printf("No secrets to pull for profile %q.\n", pullProfile)
			}
			return nil
		}

		token, err := auth.ResolveTokenForAccount(account)
		if err != nil {
			return fmt.Errorf("no access token found. Run 'lusterpass login --account <name>' or set $BWS_ACCESS_TOKEN: %w", err)
		}

		client, err := bitwarden.NewSDKClient(token)
		if err != nil {
			return fmt.Errorf("connecting to Bitwarden: %w", err)
		}
		defer client.Close()

		orgID, err := resolveOrgID(cmd, account)
		if err != nil {
			return err
		}
		allSecrets, err := client.ListSecrets(orgID)
		if err != nil {
			return fmt.Errorf("listing secrets: %w", err)
		}

		keyToID := make(map[string]string)
		for _, s := range allSecrets {
			keyToID[s.Key] = s.ID
		}

		fetched := make(map[string]string)
		for envVar, refName := range resolved.Secrets {
			id, ok := keyToID[refName]
			if !ok {
				return fmt.Errorf("secret %q (ref: %q) not found in Bitwarden", envVar, refName)
			}

			secret, err := client.GetSecret(id)
			if err != nil {
				return fmt.Errorf("fetching secret %q: %w", refName, err)
			}

			fetched[envVar] = secret.Value
		}

		cachePath := cache.CachePath(account, cfg.Project, cfg.CacheKey(pullProfile))
		if err := cache.Write(cachePath, token, fetched); err != nil {
			return fmt.Errorf("writing cache: %w", err)
		}

		fmt.Printf("Pulled %d secrets → cached (%s)\n", len(fetched), cachePath)
		return nil
	},
}

func init() {
	pullCmd.Flags().StringVar(&pullProfile, "profile", "", "Environment profile (e.g., dev, staging, prod); omit to load only the common section")
	pullCmd.Flags().String("org", "", "Bitwarden organization ID")
	rootCmd.AddCommand(pullCmd)
}
