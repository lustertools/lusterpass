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
	RunE: func(cmd *cobra.Command, args []string) error {
		if pullProfile == "" {
			return fmt.Errorf("--profile is required")
		}

		// Load config
		cfg, err := config.Load(configFile)
		if err != nil {
			return err
		}

		account, err := resolveAccount(cfg)
		if err != nil {
			return err
		}

		// Resolve token
		token, err := auth.ResolveTokenForAccount(account)
		if err != nil {
			return fmt.Errorf("no access token found. Run 'lusterpass login --account <name>' or set $BWS_ACCESS_TOKEN: %w", err)
		}

		// Connect to Bitwarden
		client, err := bitwarden.NewSDKClient(token)
		if err != nil {
			return fmt.Errorf("connecting to Bitwarden: %w", err)
		}
		defer client.Close()

		// Resolve profile
		resolved := cfg.ResolveProfile(pullProfile)

		if len(resolved.Secrets) == 0 {
			fmt.Println("No secrets to pull for this profile.")
			return nil
		}

		// Fetch secrets by reference name
		orgID, err := resolveOrgID(cmd, account)
		if err != nil {
			return err
		}
		allSecrets, err := client.ListSecrets(orgID)
		if err != nil {
			return fmt.Errorf("listing secrets: %w", err)
		}

		// Build key→ID map
		keyToID := make(map[string]string)
		for _, s := range allSecrets {
			keyToID[s.Key] = s.ID
		}

		// Fetch each referenced secret
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

		// Write to cache
		cachePath := cache.CachePath(account, cfg.Project, pullProfile)
		if err := cache.Write(cachePath, token, fetched); err != nil {
			return fmt.Errorf("writing cache: %w", err)
		}

		fmt.Printf("Pulled %d secrets → cached (%s)\n", len(fetched), cachePath)
		return nil
	},
}

func init() {
	pullCmd.Flags().StringVar(&pullProfile, "profile", "", "Environment profile (e.g., dev, staging, prod)")
	pullCmd.Flags().String("org", "", "Bitwarden organization ID")
	rootCmd.AddCommand(pullCmd)
}
