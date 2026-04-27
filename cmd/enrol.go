package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/lustertools/lusterpass/internal/auth"
	"github.com/lustertools/lusterpass/internal/bitwarden"
	"golang.org/x/term"
)

var enrolCmd = &cobra.Command{
	Use:   "enrol",
	Short: "Add a new secret to Bitwarden",
	Long: `Add a new secret to Bitwarden interactively or via flags.

Interactive mode (default):
  lusterpass enrol --org YOUR_ORG_ID

Non-interactive mode (for scripting):
  lusterpass enrol --org YOUR_ORG_ID --ref my-key--project --value "secret" --project-name credentials`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// enrol doesn't load a config file
		account, err := resolveAccount(nil)
		if err != nil {
			return err
		}

		orgID, err := resolveOrgID(cmd, account)
		if err != nil {
			return err
		}

		// Resolve token
		token, err := auth.ResolveTokenForAccount(account)
		if err != nil {
			return fmt.Errorf("no access token: %w", err)
		}

		// Connect
		client, err := bitwarden.NewSDKClient(token)
		if err != nil {
			return err
		}
		defer client.Close()

		// Check for non-interactive flags
		refName, _ := cmd.Flags().GetString("ref")
		value, _ := cmd.Flags().GetString("value")

		if refName != "" && value != "" {
			return enrolNonInteractive(client, orgID, refName, value, cmd)
		}

		// Interactive mode
		return enrolInteractive(client, orgID, cmd)
	},
}

func enrolNonInteractive(client *bitwarden.SDKClient, orgID, refName, value string, cmd *cobra.Command) error {
	note, _ := cmd.Flags().GetString("note")
	projectName, _ := cmd.Flags().GetString("project-name")

	// Resolve project
	projects, err := client.ListProjects(orgID)
	if err != nil {
		return fmt.Errorf("listing projects: %w", err)
	}

	var projectID string
	for _, p := range projects {
		if p.Name == projectName {
			projectID = p.ID
			break
		}
	}
	if projectID == "" {
		return fmt.Errorf("project %q not found. Available: %s", projectName, projectNames(projects))
	}

	_, err = client.CreateSecret(refName, value, note, orgID, []string{projectID})
	if err != nil {
		return fmt.Errorf("creating secret: %w", err)
	}

	fmt.Printf("Created secret %q in project %q\n", refName, projectName)
	return nil
}

func enrolInteractive(client *bitwarden.SDKClient, orgID string, cmd *cobra.Command) error {
	reader := bufio.NewReader(os.Stdin)

	projects, err := client.ListProjects(orgID)
	if err != nil {
		return fmt.Errorf("listing projects: %w", err)
	}

	if len(projects) == 0 {
		return fmt.Errorf("no projects found in organization")
	}

	fmt.Println("Select project:")
	for i, p := range projects {
		fmt.Printf("  %d) %s\n", i+1, p.Name)
	}
	fmt.Print("Choice: ")
	choiceStr, _ := reader.ReadString('\n')
	choiceStr = strings.TrimSpace(choiceStr)

	var choice int
	fmt.Sscanf(choiceStr, "%d", &choice)
	if choice < 1 || choice > len(projects) {
		return fmt.Errorf("invalid choice")
	}
	selectedProject := projects[choice-1]

	fmt.Print("Reference name (e.g., openai-key--myapp--dev): ")
	refName, _ := reader.ReadString('\n')
	refName = strings.TrimSpace(refName)
	if refName == "" {
		return fmt.Errorf("reference name cannot be empty")
	}

	fmt.Print("Note (optional): ")
	note, _ := reader.ReadString('\n')
	note = strings.TrimSpace(note)

	fmt.Print("Secret value: ")
	valueBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return fmt.Errorf("reading value: %w", err)
	}

	_, err = client.CreateSecret(refName, string(valueBytes), note, orgID, []string{selectedProject.ID})
	if err != nil {
		return fmt.Errorf("creating secret: %w", err)
	}

	fmt.Printf("Created secret %q in project %q\n", refName, selectedProject.Name)
	return nil
}

func projectNames(projects []bitwarden.Project) string {
	names := make([]string, len(projects))
	for i, p := range projects {
		names[i] = p.Name
	}
	return strings.Join(names, ", ")
}

func init() {
	enrolCmd.Flags().String("org", "", "Bitwarden organization ID")
	enrolCmd.Flags().String("ref", "", "Secret reference name (non-interactive mode)")
	enrolCmd.Flags().String("value", "", "Secret value (non-interactive mode)")
	enrolCmd.Flags().String("note", "", "Optional note (non-interactive mode)")
	enrolCmd.Flags().String("project-name", "credentials", "Bitwarden project name (non-interactive mode)")
	rootCmd.AddCommand(enrolCmd)
}
