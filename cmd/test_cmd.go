package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/lustertools/lusterpass/internal/auth"
	"github.com/lustertools/lusterpass/internal/bitwarden"
	"github.com/lustertools/lusterpass/internal/cache"
	"github.com/lustertools/lusterpass/internal/config"
)

var testCmd = &cobra.Command{
	Use:    "test",
	Short:  "Run end-to-end test using Bitwarden testing project",
	Long:   "Maintainer self-check: seeds two secrets in the Bitwarden 'testing' project, pulls them via the testdata/mockapp fixture, verifies the resolved exports against expected.env, and cleans up. Requires a Bitwarden project named 'testing' to exist. Not intended for end-user workflows.",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// test uses the active account
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

		// Find "testing" project
		projects, err := client.ListProjects(orgID)
		if err != nil {
			return err
		}

		var testingProjectID string
		for _, p := range projects {
			if p.Name == "testing" {
				testingProjectID = p.ID
				break
			}
		}
		if testingProjectID == "" {
			return fmt.Errorf("Bitwarden project 'testing' not found. Create it first.")
		}

		fmt.Println("Step 1: Seeding test secrets...")
		testSecrets := map[string]string{
			"test-secret-a--lusterpass-test": "test-value-a",
			"test-secret-b--lusterpass-test": "test-value-b",
		}

		var createdIDs []string
		for key, value := range testSecrets {
			s, err := client.CreateSecret(key, value, "lusterpass e2e test", orgID, []string{testingProjectID})
			if err != nil {
				return fmt.Errorf("seeding %q: %w", key, err)
			}
			createdIDs = append(createdIDs, s.ID)
		}

		// Cleanup on exit
		defer func() {
			fmt.Println("Step 5: Cleaning up test secrets...")
			if err := client.DeleteSecrets(createdIDs); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: cleanup failed: %v\n", err)
			}
			fmt.Println("Cleanup done.")
		}()

		fmt.Println("Step 2: Pulling secrets...")
		mockDir := "testdata/mockapp"
		cfg, err := config.Load(filepath.Join(mockDir, ".lusterpass.yaml"))
		if err != nil {
			return err
		}

		resolved, err := cfg.ResolveProfile("test")
		if err != nil {
			return err
		}

		// List secrets to get IDs
		allSecrets, err := client.ListSecrets(orgID)
		if err != nil {
			return err
		}
		keyToID := make(map[string]string)
		for _, s := range allSecrets {
			keyToID[s.Key] = s.ID
		}

		fetched := make(map[string]string)
		for envVar, refName := range resolved.Secrets {
			id, ok := keyToID[refName]
			if !ok {
				return fmt.Errorf("secret %q not found", refName)
			}
			secret, err := client.GetSecret(id)
			if err != nil {
				return err
			}
			fetched[envVar] = secret.Value
		}

		cachePath := cache.CachePath(account, cfg.Project, "test")
		if err := cache.Write(cachePath, token, fetched); err != nil {
			return err
		}

		fmt.Println("Step 3: Generating env output...")
		output := formatExports(resolved.Vars, fetched)

		fmt.Println("Step 4: Verifying against expected output...")
		expectedBytes, err := os.ReadFile(filepath.Join(mockDir, "expected.env"))
		if err != nil {
			return fmt.Errorf("reading expected.env: %w", err)
		}

		// Parse expected: KEY=VALUE lines
		expectedLines := strings.Split(strings.TrimSpace(string(expectedBytes)), "\n")
		var failures []string
		for _, line := range expectedLines {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key, expectedVal := parts[0], parts[1]
			exportLine := fmt.Sprintf("export %s='%s'", key, expectedVal)
			if !strings.Contains(output, exportLine) {
				failures = append(failures, fmt.Sprintf("  expected: %s\n", exportLine))
			}
		}

		if len(failures) > 0 {
			return fmt.Errorf("FAIL: mismatched output:\n%s\nActual output:\n%s", strings.Join(failures, ""), output)
		}

		fmt.Println("\nAll tests passed!")
		return nil
	},
}

func init() {
	testCmd.Flags().String("org", "", "Bitwarden organization ID")
	rootCmd.AddCommand(testCmd)
}
