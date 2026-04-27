package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/lustertools/lusterpass/internal/auth"
	"github.com/lustertools/lusterpass/internal/bitwarden"
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Manage Bitwarden accounts",
}

var accountListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured accounts",
	RunE: func(cmd *cobra.Command, args []string) error {
		accounts, err := auth.ListAccounts()
		if err != nil {
			return err
		}

		if len(accounts) == 0 {
			fmt.Println("No accounts configured. Run 'lusterpass login --account <name>' to add one.")
			return nil
		}

		active, _ := auth.LoadActiveAccount()

		for _, a := range accounts {
			if a == active {
				fmt.Printf("* %s\n", a)
			} else {
				fmt.Printf("  %s\n", a)
			}
		}
		return nil
	},
}

var accountUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Set the active account",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := auth.ValidateAccountName(name); err != nil {
			return err
		}

		// Check account exists
		dir := auth.AccountDir(name)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			accounts, _ := auth.ListAccounts()
			if len(accounts) > 0 {
				return fmt.Errorf("account %q not found. Available: %s", name, strings.Join(accounts, ", "))
			}
			return fmt.Errorf("account %q not found. Run 'lusterpass login --account %s' first", name, name)
		}

		if err := auth.SetActiveAccount(name); err != nil {
			return err
		}

		fmt.Printf("Active account: %s\n", name)
		return nil
	},
}

var accountRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove an account and its cached data",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := auth.ValidateAccountName(name); err != nil {
			return err
		}

		dir := auth.AccountDir(name)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return fmt.Errorf("account %q not found", name)
		}

		// Confirm
		fmt.Printf("Remove account %q and all its cached data? [y/N] ", name)
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}

		// Check if this is the active account before removing
		active, _ := auth.LoadActiveAccount()
		isActive := active == name

		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("removing account: %w", err)
		}

		if isActive {
			os.Remove(auth.ActiveAccountPath())
			fmt.Printf("Removed active account %q. Set a new one with 'lusterpass account use <name>'.\n", name)
		} else {
			fmt.Printf("Removed account %q.\n", name)
		}

		return nil
	},
}

var defaultProjects = []string{"credentials", "certificates", "testing"}

var accountSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Create default Bitwarden projects (credentials, certificates, testing)",
	RunE: func(cmd *cobra.Command, args []string) error {
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

		// List existing projects
		existing, err := client.ListProjects(orgID)
		if err != nil {
			return fmt.Errorf("listing projects: %w", err)
		}

		existingNames := make(map[string]bool)
		for _, p := range existing {
			existingNames[p.Name] = true
		}

		created := 0
		for _, name := range defaultProjects {
			if existingNames[name] {
				fmt.Printf("  ✓ %s (already exists)\n", name)
				continue
			}
			_, err := client.CreateProject(orgID, name)
			if err != nil {
				return fmt.Errorf("creating project %q: %w", name, err)
			}
			fmt.Printf("  + %s (created)\n", name)
			created++
		}

		if created == 0 {
			fmt.Println("\nAll projects already exist.")
		} else {
			fmt.Printf("\nCreated %d project(s).\n", created)
		}
		return nil
	},
}

func init() {
	accountSetupCmd.Flags().String("org", "", "Bitwarden organization ID")
	accountCmd.AddCommand(accountListCmd)
	accountCmd.AddCommand(accountUseCmd)
	accountCmd.AddCommand(accountRemoveCmd)
	accountCmd.AddCommand(accountSetupCmd)
	rootCmd.AddCommand(accountCmd)
}
