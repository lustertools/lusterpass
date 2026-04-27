package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/lustertools/lusterpass/internal/auth"
	"github.com/lustertools/lusterpass/internal/bitwarden"
)

var listProject string

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List secrets in Bitwarden (names only, never values)",
	RunE: func(cmd *cobra.Command, args []string) error {
		// list doesn't load a config file, so pass nil to resolveAccount
		account, err := resolveAccount(nil)
		if err != nil {
			return err
		}

		orgID, err := resolveOrgID(cmd, account)
		if err != nil {
			return err
		}

		token, err := auth.ResolveTokenForAccount(account)
		if err != nil {
			return fmt.Errorf("no access token: %w", err)
		}

		client, err := bitwarden.NewSDKClient(token)
		if err != nil {
			return err
		}
		defer client.Close()

		secrets, err := client.ListSecrets(orgID)
		if err != nil {
			return fmt.Errorf("listing secrets: %w", err)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tID")
		for _, s := range secrets {
			fmt.Fprintf(w, "%s\t%s\n", s.Key, s.ID)
		}
		w.Flush()

		return nil
	},
}

func init() {
	listCmd.Flags().String("org", "", "Bitwarden organization ID")
	listCmd.Flags().StringVar(&listProject, "project", "", "Filter by project name")
	rootCmd.AddCommand(listCmd)
}
