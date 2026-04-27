package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/lustertools/lusterpass/internal/auth"
	"github.com/lustertools/lusterpass/internal/bitwarden"
	"golang.org/x/term"
)

var loginAccount string

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Store Bitwarden access token and org ID for an account",
	RunE: func(cmd *cobra.Command, args []string) error {
		if loginAccount == "" {
			accounts, _ := auth.ListAccounts()
			msg := "--account is required."
			if len(accounts) > 0 {
				msg += "\nAvailable accounts: " + strings.Join(accounts, ", ")
			}
			msg += "\nUsage: lusterpass login --account <name>"
			return fmt.Errorf("%s", msg)
		}

		if err := auth.ValidateAccountName(loginAccount); err != nil {
			return err
		}

		reader := bufio.NewReader(os.Stdin)

		fmt.Print("Enter Bitwarden access token: ")
		tokenBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			return fmt.Errorf("reading token: %w", err)
		}

		token := string(tokenBytes)
		if token == "" {
			return fmt.Errorf("token cannot be empty")
		}

		// Verify token works
		fmt.Println("Verifying token...")
		client, err := bitwarden.NewSDKClient(token)
		if err != nil {
			return fmt.Errorf("token verification failed: %w", err)
		}
		client.Close()
		fmt.Println("Token verified.")

		// Store encrypted token in account dir
		accountDir := auth.AccountDir(loginAccount)
		os.MkdirAll(accountDir, 0700)
		configPath := filepath.Join(accountDir, "config")

		if err := auth.StoreToken(configPath, token); err != nil {
			return fmt.Errorf("storing token: %w", err)
		}
		fmt.Printf("Access token saved to %s (encrypted)\n", configPath)

		// Prompt for org ID
		fmt.Println()
		fmt.Println("Org ID can be found in Bitwarden Secrets Manager URL:")
		fmt.Println("  https://vault.bitwarden.com/#/sm/YOUR_ORG_ID/secrets")
		fmt.Println("Or from: bws project list → organizationId field")
		fmt.Println()
		fmt.Print("Enter Bitwarden organization ID (or press Enter to skip): ")
		orgID, _ := reader.ReadString('\n')
		orgID = strings.TrimSpace(orgID)

		if orgID != "" {
			orgPath := filepath.Join(accountDir, "org")
			if err := auth.StoreOrgID(orgPath, orgID); err != nil {
				return fmt.Errorf("storing org ID: %w", err)
			}
			fmt.Printf("Org ID saved to %s\n", orgPath)
		} else {
			fmt.Println("Skipped. You can pass --org to commands or re-run login later.")
		}

		// Set as active if no active account exists
		if _, err := auth.LoadActiveAccount(); err != nil {
			auth.SetActiveAccount(loginAccount)
			fmt.Printf("\nAccount %q set as active.\n", loginAccount)
		} else {
			active, _ := auth.LoadActiveAccount()
			fmt.Printf("\nAccount %q saved. Active account remains %q. Use 'lusterpass account use %s' to switch.\n", loginAccount, active, loginAccount)
		}

		return nil
	},
}

func init() {
	loginCmd.Flags().StringVar(&loginAccount, "account", "", "Account name (required)")
	rootCmd.AddCommand(loginCmd)
}
