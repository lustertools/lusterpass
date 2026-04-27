package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/lustertools/lusterpass/internal/auth"
)

// resolveOrgID returns the org ID for the given account.
// Priority: --org flag > account's org file > error.
func resolveOrgID(cmd *cobra.Command, account string) (string, error) {
	orgFlag, _ := cmd.Flags().GetString("org")
	orgID, err := auth.ResolveOrgIDForAccount(account, orgFlag)
	if err != nil {
		return "", fmt.Errorf("no org ID for account %q: use --org or run 'lusterpass login --account %s'", account, account)
	}
	return orgID, nil
}
