package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lustertools/lusterpass/internal/auth"
	"github.com/lustertools/lusterpass/internal/bitwarden"
)

const testOrgID = "a1e4a796-78c7-41c7-8d7c-b40a00cf6392"

// ensureTestAccount ensures a "testing" account exists under ~/.lusterpass/accounts/
// by migrating from the old flat config if needed. Returns the token.
func ensureTestAccount(t *testing.T) string {
	t.Helper()
	home, _ := os.UserHomeDir()
	accountName := "testing"
	accountDir := filepath.Join(home, ".lusterpass", "accounts", accountName)
	accountConfig := filepath.Join(accountDir, "config")
	accountOrg := filepath.Join(accountDir, "org")
	activePath := filepath.Join(home, ".lusterpass", "active")

	// If account already exists, use it
	if token, err := auth.LoadToken(accountConfig); err == nil {
		// Ensure it's set as active
		os.WriteFile(activePath, []byte(accountName), 0600)
		return token
	}

	// Try env var
	if token := os.Getenv("BWS_ACCESS_TOKEN"); token != "" {
		os.MkdirAll(accountDir, 0700)
		auth.StoreToken(accountConfig, token)
		auth.StoreOrgID(accountOrg, testOrgID)
		os.WriteFile(activePath, []byte(accountName), 0600)
		return token
	}

	// Try old flat config
	oldConfig := filepath.Join(home, ".lusterpass", "config")
	if token, err := auth.LoadToken(oldConfig); err == nil {
		os.MkdirAll(accountDir, 0700)
		auth.StoreToken(accountConfig, token)
		// Copy org ID if exists
		if orgData, err := os.ReadFile(filepath.Join(home, ".lusterpass", "org")); err == nil {
			os.WriteFile(accountOrg, orgData, 0600)
		} else {
			auth.StoreOrgID(accountOrg, testOrgID)
		}
		os.WriteFile(activePath, []byte(accountName), 0600)
		return token
	}

	return ""
}

// ensureAccountOrSkip ensures a test account is set up, or skips the test.
func ensureAccountOrSkip(t *testing.T) {
	t.Helper()
	token := ensureTestAccount(t)
	if token == "" {
		t.Skipf("No access token available, skipping integration test")
	}
}

func getTestClient(t *testing.T) bitwarden.Client {
	t.Helper()
	token := ensureTestAccount(t)
	if token == "" {
		t.Skipf("No access token available, skipping integration test")
	}

	client, err := bitwarden.NewSDKClient(token)
	if err != nil {
		t.Fatalf("Failed to create BW client: %v", err)
	}
	return client
}

func findProjectID(t *testing.T, client bitwarden.Client, name string) string {
	t.Helper()
	projects, err := client.ListProjects(testOrgID)
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}
	for _, p := range projects {
		if p.Name == name {
			return p.ID
		}
	}
	t.Fatalf("Project %q not found", name)
	return ""
}

func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "lusterpass")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	return bin
}

// Scenario 1: Multi-profile (dev vs prod with same env var names)
func TestScenarioMultiProfile(t *testing.T) {
	client := getTestClient(t)
	defer client.Close()

	testingID := findProjectID(t, client, "testing")

	// Create secrets for dev and prod
	devKey, _ := client.CreateSecret("api-key--multi--dev", "dev-key-123", "", testOrgID, []string{testingID})
	prodKey, _ := client.CreateSecret("api-key--multi--prod", "prod-key-456", "", testOrgID, []string{testingID})
	devDB, _ := client.CreateSecret("db-pass--multi--dev", "devdb-pass", "", testOrgID, []string{testingID})
	prodDB, _ := client.CreateSecret("db-pass--multi--prod", "proddb-secret", "", testOrgID, []string{testingID})
	defer client.DeleteSecrets([]string{devKey.ID, prodKey.ID, devDB.ID, prodDB.ID})

	// Write config
	dir := t.TempDir()
	configYAML := `
project: multi-test
common:
  vars:
    APP_NAME: multiapp
profiles:
  dev:
    vars:
      ENV: development
    secrets:
      API_KEY: api-key--multi--dev
      DB_PASSWORD: db-pass--multi--dev
  prod:
    vars:
      ENV: production
    secrets:
      API_KEY: api-key--multi--prod
      DB_PASSWORD: db-pass--multi--prod
`
	os.WriteFile(filepath.Join(dir, ".lusterpass.yaml"), []byte(configYAML), 0644)

	bin := buildBinary(t)

	// Pull dev
	cmd := exec.Command(bin, "pull", "--profile", "dev", "--org", testOrgID)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pull dev failed: %s: %v", out, err)
	}

	// Env dev
	cmd = exec.Command(bin, "env", "--profile", "dev")
	cmd.Dir = dir
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("env dev failed: %s: %v", out, err)
	}
	devOutput := string(out)

	if !strings.Contains(devOutput, "export API_KEY='dev-key-123'") {
		t.Errorf("dev: expected dev API_KEY, got:\n%s", devOutput)
	}
	if !strings.Contains(devOutput, "export DB_PASSWORD='devdb-pass'") {
		t.Errorf("dev: expected dev DB_PASSWORD, got:\n%s", devOutput)
	}
	if !strings.Contains(devOutput, "export ENV='development'") {
		t.Errorf("dev: expected ENV=development, got:\n%s", devOutput)
	}
	if !strings.Contains(devOutput, "export APP_NAME='multiapp'") {
		t.Errorf("dev: expected APP_NAME from common, got:\n%s", devOutput)
	}

	// Pull prod
	cmd = exec.Command(bin, "pull", "--profile", "prod", "--org", testOrgID)
	cmd.Dir = dir
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pull prod failed: %s: %v", out, err)
	}

	// Env prod
	cmd = exec.Command(bin, "env", "--profile", "prod")
	cmd.Dir = dir
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("env prod failed: %s: %v", out, err)
	}
	prodOutput := string(out)

	if !strings.Contains(prodOutput, "export API_KEY='prod-key-456'") {
		t.Errorf("prod: expected prod API_KEY, got:\n%s", prodOutput)
	}
	if !strings.Contains(prodOutput, "export DB_PASSWORD='proddb-secret'") {
		t.Errorf("prod: expected prod DB_PASSWORD, got:\n%s", prodOutput)
	}
	if !strings.Contains(prodOutput, "export ENV='production'") {
		t.Errorf("prod: expected ENV=production, got:\n%s", prodOutput)
	}

	t.Log("Scenario: Multi-profile dev/prod passed")
}

// Scenario 2: Common secrets override
func TestScenarioCommonOverride(t *testing.T) {
	client := getTestClient(t)
	defer client.Close()

	testingID := findProjectID(t, client, "testing")

	sharedSecret, _ := client.CreateSecret("shared-key--override", "shared-value", "", testOrgID, []string{testingID})
	devOverride, _ := client.CreateSecret("override-key--dev", "overridden-value", "", testOrgID, []string{testingID})
	defer client.DeleteSecrets([]string{sharedSecret.ID, devOverride.ID})

	dir := t.TempDir()
	configYAML := `
project: override-test
common:
  vars:
    LOG_LEVEL: info
    APP_NAME: overrideapp
  secrets:
    SHARED_KEY: shared-key--override
    MY_SECRET: shared-key--override
profiles:
  dev:
    vars:
      LOG_LEVEL: debug
    secrets:
      MY_SECRET: override-key--dev
`
	os.WriteFile(filepath.Join(dir, ".lusterpass.yaml"), []byte(configYAML), 0644)

	bin := buildBinary(t)

	cmd := exec.Command(bin, "pull", "--profile", "dev", "--org", testOrgID)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pull failed: %s: %v", out, err)
	}

	cmd = exec.Command(bin, "env", "--profile", "dev")
	cmd.Dir = dir
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("env failed: %s: %v", out, err)
	}
	output := string(out)

	// LOG_LEVEL should be overridden by dev profile
	if !strings.Contains(output, "export LOG_LEVEL='debug'") {
		t.Errorf("expected LOG_LEVEL=debug (override), got:\n%s", output)
	}
	// APP_NAME should come from common
	if !strings.Contains(output, "export APP_NAME='overrideapp'") {
		t.Errorf("expected APP_NAME from common, got:\n%s", output)
	}
	// SHARED_KEY from common secrets
	if !strings.Contains(output, "export SHARED_KEY='shared-value'") {
		t.Errorf("expected SHARED_KEY from common secrets, got:\n%s", output)
	}
	// MY_SECRET should be overridden by dev profile secret
	if !strings.Contains(output, "export MY_SECRET='overridden-value'") {
		t.Errorf("expected MY_SECRET=overridden-value (profile override), got:\n%s", output)
	}

	t.Log("Scenario: Common override passed")
}

// Scenario 3: Special characters in secret values
func TestScenarioSpecialChars(t *testing.T) {
	client := getTestClient(t)
	defer client.Close()

	testingID := findProjectID(t, client, "testing")

	// Secrets with tricky characters
	s1, _ := client.CreateSecret("special-single-quote", "it's a test", "", testOrgID, []string{testingID})
	s2, _ := client.CreateSecret("special-dollar-sign", "price=$100", "", testOrgID, []string{testingID})
	s3, _ := client.CreateSecret("special-backtick", "cmd=`echo hi`", "", testOrgID, []string{testingID})
	s4, _ := client.CreateSecret("special-newline", "line1\nline2", "", testOrgID, []string{testingID})
	s5, _ := client.CreateSecret("special-double-quote", `say "hello"`, "", testOrgID, []string{testingID})
	s6, _ := client.CreateSecret("special-backslash", `path\to\file`, "", testOrgID, []string{testingID})
	defer client.DeleteSecrets([]string{s1.ID, s2.ID, s3.ID, s4.ID, s5.ID, s6.ID})

	dir := t.TempDir()
	configYAML := `
project: special-test
profiles:
  test:
    secrets:
      SINGLE_QUOTE: special-single-quote
      DOLLAR_SIGN: special-dollar-sign
      BACKTICK: special-backtick
      NEWLINE: special-newline
      DOUBLE_QUOTE: special-double-quote
      BACKSLASH: special-backslash
`
	os.WriteFile(filepath.Join(dir, ".lusterpass.yaml"), []byte(configYAML), 0644)

	bin := buildBinary(t)

	cmd := exec.Command(bin, "pull", "--profile", "test", "--org", testOrgID)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pull failed: %s: %v", out, err)
	}

	cmd = exec.Command(bin, "env", "--profile", "test")
	cmd.Dir = dir
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("env failed: %s: %v", out, err)
	}
	output := string(out)

	// Verify all special chars are present and properly escaped
	// The shell escape uses single quotes with '\'' for embedded single quotes
	if !strings.Contains(output, "SINGLE_QUOTE=") {
		t.Errorf("missing SINGLE_QUOTE in output:\n%s", output)
	}
	if !strings.Contains(output, "DOLLAR_SIGN=") {
		t.Errorf("missing DOLLAR_SIGN in output:\n%s", output)
	}
	if !strings.Contains(output, "BACKTICK=") {
		t.Errorf("missing BACKTICK in output:\n%s", output)
	}
	if !strings.Contains(output, "DOUBLE_QUOTE=") {
		t.Errorf("missing DOUBLE_QUOTE in output:\n%s", output)
	}
	if !strings.Contains(output, "BACKSLASH=") {
		t.Errorf("missing BACKSLASH in output:\n%s", output)
	}

	// Now test that eval actually works in a real shell
	evalCmd := exec.Command("bash", "-c", output+"\necho $SINGLE_QUOTE")
	evalOut, err := evalCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("shell eval failed: %s: %v", evalOut, err)
	}
	if strings.TrimSpace(string(evalOut)) != "it's a test" {
		t.Errorf("shell eval SINGLE_QUOTE: expected \"it's a test\", got %q", strings.TrimSpace(string(evalOut)))
	}

	evalCmd = exec.Command("bash", "-c", output+"\necho \"$DOLLAR_SIGN\"")
	evalOut, err = evalCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("shell eval failed: %s: %v", evalOut, err)
	}
	if strings.TrimSpace(string(evalOut)) != "price=$100" {
		t.Errorf("shell eval DOLLAR_SIGN: expected \"price=$100\", got %q", strings.TrimSpace(string(evalOut)))
	}

	evalCmd = exec.Command("bash", "-c", output+"\necho \"$DOUBLE_QUOTE\"")
	evalOut, err = evalCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("shell eval failed: %s: %v", evalOut, err)
	}
	if strings.TrimSpace(string(evalOut)) != `say "hello"` {
		t.Errorf("shell eval DOUBLE_QUOTE: expected 'say \"hello\"', got %q", strings.TrimSpace(string(evalOut)))
	}

	t.Log("Scenario: Special characters passed")
}

// Scenario 4: Empty profile (only common vars, no secrets)
func TestScenarioVarsOnly(t *testing.T) {
	ensureAccountOrSkip(t)
	dir := t.TempDir()
	configYAML := `
project: vars-only-test
common:
  vars:
    APP_NAME: varsapp
    LOG_LEVEL: info
    TZ: UTC
profiles:
  dev:
    vars:
      LOG_LEVEL: debug
      PORT: "3000"
`
	os.WriteFile(filepath.Join(dir, ".lusterpass.yaml"), []byte(configYAML), 0644)

	bin := buildBinary(t)

	// Pull with no secrets should succeed gracefully
	cmd := exec.Command(bin, "pull", "--profile", "dev", "--org", testOrgID)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pull with no secrets failed: %s: %v", out, err)
	}
	if !strings.Contains(string(out), "No secrets to pull") {
		t.Errorf("expected 'No secrets to pull' message, got: %s", out)
	}

	t.Log("Scenario: Vars-only passed")
}

// Scenario 5: Missing secret reference (should error clearly)
func TestScenarioMissingSecret(t *testing.T) {
	ensureAccountOrSkip(t)
	dir := t.TempDir()
	configYAML := `
project: missing-test
profiles:
  dev:
    secrets:
      NONEXISTENT: this-secret-does-not-exist-in-bitwarden
`
	os.WriteFile(filepath.Join(dir, ".lusterpass.yaml"), []byte(configYAML), 0644)

	bin := buildBinary(t)

	cmd := exec.Command(bin, "pull", "--profile", "dev", "--org", testOrgID)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error for missing secret, but pull succeeded")
	}
	if !strings.Contains(string(out), "not found in Bitwarden") {
		t.Errorf("expected clear error message about missing secret, got: %s", out)
	}

	t.Log("Scenario: Missing secret error passed")
}

// Scenario 6: env without pull (should error clearly)
func TestScenarioEnvWithoutPull(t *testing.T) {
	ensureAccountOrSkip(t)
	dir := t.TempDir()
	configYAML := `
project: nopull-test
profiles:
  dev:
    secrets:
      KEY: some-ref
`
	os.WriteFile(filepath.Join(dir, ".lusterpass.yaml"), []byte(configYAML), 0644)

	bin := buildBinary(t)

	cmd := exec.Command(bin, "env", "--profile", "dev")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error for env without pull")
	}
	if !strings.Contains(string(out), "lusterpass pull") {
		t.Errorf("expected error message suggesting 'lusterpass pull', got: %s", out)
	}

	t.Log("Scenario: Env-without-pull error passed")
}

// Scenario 7: Large number of secrets
func TestScenarioLargeSecretSet(t *testing.T) {
	client := getTestClient(t)
	defer client.Close()

	testingID := findProjectID(t, client, "testing")

	// Create 20 secrets
	var ids []string
	secretRefs := make(map[string]string)
	for i := 0; i < 20; i++ {
		key := strings.Replace(strings.Replace(
			"large-test-KEY_NUM--large",
			"KEY", strings.ToUpper(string(rune('A'+i%26))), 1),
			"NUM", strings.Replace(
				"00X", "X", string(rune('0'+i%10)), 1), 1)
		// Simpler key naming
		key = "large-test-" + string(rune('a'+i)) + "--large"
		envVar := "SECRET_" + string(rune('A'+i))
		value := "value-" + string(rune('a'+i)) + "-secret"

		s, err := client.CreateSecret(key, value, "", testOrgID, []string{testingID})
		if err != nil {
			t.Fatalf("Creating secret %d failed: %v", i, err)
		}
		ids = append(ids, s.ID)
		secretRefs[envVar] = key
	}
	defer client.DeleteSecrets(ids)

	// Build YAML
	dir := t.TempDir()
	var yamlSecrets strings.Builder
	for envVar, ref := range secretRefs {
		yamlSecrets.WriteString("      " + envVar + ": " + ref + "\n")
	}
	configYAML := "project: large-test\nprofiles:\n  test:\n    secrets:\n" + yamlSecrets.String()
	os.WriteFile(filepath.Join(dir, ".lusterpass.yaml"), []byte(configYAML), 0644)

	bin := buildBinary(t)

	cmd := exec.Command(bin, "pull", "--profile", "test", "--org", testOrgID)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pull failed: %s: %v", out, err)
	}
	if !strings.Contains(string(out), "Pulled 20 secrets") {
		t.Errorf("expected 'Pulled 20 secrets', got: %s", out)
	}

	cmd = exec.Command(bin, "env", "--profile", "test")
	cmd.Dir = dir
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("env failed: %s: %v", out, err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 20 {
		t.Errorf("expected 20 export lines, got %d:\n%s", len(lines), out)
	}

	t.Log("Scenario: Large secret set (20 secrets) passed")
}

// Scenario 8: Credentials project (not just testing)
func TestScenarioCredentialsProject(t *testing.T) {
	client := getTestClient(t)
	defer client.Close()

	credsID := findProjectID(t, client, "credentials")

	s, err := client.CreateSecret("real-api-key--creds-test", "sk-real-key-value", "A real API key", testOrgID, []string{credsID})
	if err != nil {
		t.Fatalf("Creating secret in credentials failed: %v", err)
	}
	defer client.DeleteSecrets([]string{s.ID})

	dir := t.TempDir()
	configYAML := `
project: creds-test
profiles:
  dev:
    vars:
      APP_NAME: credsapp
    secrets:
      REAL_API_KEY: real-api-key--creds-test
`
	os.WriteFile(filepath.Join(dir, ".lusterpass.yaml"), []byte(configYAML), 0644)

	bin := buildBinary(t)

	cmd := exec.Command(bin, "pull", "--profile", "dev", "--org", testOrgID)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pull failed: %s: %v", out, err)
	}

	cmd = exec.Command(bin, "env", "--profile", "dev")
	cmd.Dir = dir
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("env failed: %s: %v", out, err)
	}
	output := string(out)

	if !strings.Contains(output, "export REAL_API_KEY='sk-real-key-value'") {
		t.Errorf("expected REAL_API_KEY from credentials project, got:\n%s", output)
	}

	t.Log("Scenario: Credentials project passed")
}

// Scenario 9: Re-pull updates cached values
func TestScenarioRepullUpdatesCache(t *testing.T) {
	client := getTestClient(t)
	defer client.Close()

	testingID := findProjectID(t, client, "testing")

	s, err := client.CreateSecret("repull-key--test", "original-value", "", testOrgID, []string{testingID})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer client.DeleteSecrets([]string{s.ID})

	dir := t.TempDir()
	configYAML := `
project: repull-test
profiles:
  dev:
    secrets:
      MY_KEY: repull-key--test
`
	os.WriteFile(filepath.Join(dir, ".lusterpass.yaml"), []byte(configYAML), 0644)

	bin := buildBinary(t)

	// First pull
	cmd := exec.Command(bin, "pull", "--profile", "dev", "--org", testOrgID)
	cmd.Dir = dir
	cmd.CombinedOutput()

	cmd = exec.Command(bin, "env", "--profile", "dev")
	cmd.Dir = dir
	out, _ := cmd.CombinedOutput()
	if !strings.Contains(string(out), "export MY_KEY='original-value'") {
		t.Fatalf("first pull: expected original-value, got:\n%s", out)
	}

	// Update secret value in BW
	_, err = client.(*bitwarden.SDKClient).UpdateSecret(s.ID, "repull-key--test", "updated-value", "", testOrgID, []string{testingID})
	if err != nil {
		// If UpdateSecret doesn't exist, skip this test
		t.Skipf("UpdateSecret not available, skipping re-pull test: %v", err)
	}

	// Re-pull
	cmd = exec.Command(bin, "pull", "--profile", "dev", "--org", testOrgID)
	cmd.Dir = dir
	cmd.CombinedOutput()

	cmd = exec.Command(bin, "env", "--profile", "dev")
	cmd.Dir = dir
	out, _ = cmd.CombinedOutput()
	if !strings.Contains(string(out), "export MY_KEY='updated-value'") {
		t.Errorf("re-pull: expected updated-value, got:\n%s", out)
	}

	t.Log("Scenario: Re-pull updates cache passed")
}

// TestConfigFlagEnv verifies lusterpass env works with --config pointing to a
// config file in a different directory than cwd.
func TestConfigFlagEnv(t *testing.T) {
	client := getTestClient(t)
	defer client.Close()

	testingID := findProjectID(t, client, "testing")

	// Create a test secret
	secret, _ := client.CreateSecret("cfg-flag-test-key", "cfg-flag-value", "", testOrgID, []string{testingID})
	defer client.DeleteSecrets([]string{secret.ID})

	// Write config to a subdirectory
	projectDir := t.TempDir()
	configYAML := `
project: cfg-flag-test
common:
  vars:
    APP_NAME: cfgtest
profiles:
  dev:
    vars:
      ENV: development
    secrets:
      TEST_KEY: cfg-flag-test-key
`
	os.WriteFile(filepath.Join(projectDir, ".lusterpass.yaml"), []byte(configYAML), 0644)

	bin := buildBinary(t)

	// Pull using --config from a different directory
	cmd := exec.Command(bin, "pull", "--profile", "dev", "--org", testOrgID, "--config", filepath.Join(projectDir, ".lusterpass.yaml"))
	cmd.Dir = t.TempDir() // run from a DIFFERENT directory
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pull with --config failed: %s: %v", out, err)
	}

	// Env using -c shorthand from a different directory
	cmd = exec.Command(bin, "env", "--profile", "dev", "-c", filepath.Join(projectDir, ".lusterpass.yaml"))
	cmd.Dir = t.TempDir() // run from a DIFFERENT directory
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("env with --config failed: %s: %v", out, err)
	}
	output := string(out)

	if !strings.Contains(output, "export APP_NAME='cfgtest'") {
		t.Errorf("expected APP_NAME from config, got:\n%s", output)
	}
	if !strings.Contains(output, "export ENV='development'") {
		t.Errorf("expected ENV from config, got:\n%s", output)
	}
	if !strings.Contains(output, "export TEST_KEY='cfg-flag-value'") {
		t.Errorf("expected TEST_KEY from config, got:\n%s", output)
	}

	t.Log("Scenario: --config flag passed")
}

// TestConfigFlagNonexistent verifies a clear error when --config points to
// a file that does not exist.
func TestConfigFlagNonexistent(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "env", "--profile", "dev", "--config", "/nonexistent/path/.lusterpass.yaml")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error for nonexistent config, got success")
	}
	output := string(out)
	if !strings.Contains(output, "reading config") {
		t.Errorf("expected 'reading config' error, got:\n%s", output)
	}
}

// --- Multi-account integration tests ---

func TestAccountListEmpty(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "account", "list")
	out, _ := cmd.CombinedOutput()
	output := string(out)

	// Should show helpful message (not crash)
	if !strings.Contains(output, "No accounts") && !strings.Contains(output, "account") {
		t.Logf("account list output: %s", output)
	}
}

func TestLoginRequiresAccount(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "login")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error without --account")
	}
	output := string(out)
	if !strings.Contains(output, "--account is required") {
		t.Errorf("expected '--account is required' error, got:\n%s", output)
	}
}

func TestEnvRequiresAccountOrCache(t *testing.T) {
	bin := buildBinary(t)

	dir := t.TempDir()
	configYAML := `
project: no-cache-test
profiles:
  dev:
    vars:
      KEY: value
`
	os.WriteFile(filepath.Join(dir, ".lusterpass.yaml"), []byte(configYAML), 0644)

	cmd := exec.Command(bin, "env", "--profile", "dev")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error without account or cache")
	}
	output := string(out)
	// Either no account configured, or account exists but no cache for this project
	if !strings.Contains(output, "no active account") &&
		!strings.Contains(output, "no accounts") &&
		!strings.Contains(output, "reading cache") {
		t.Errorf("expected account or cache error, got:\n%s", output)
	}
}

func TestLoginRejectsInvalidAccountName(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "login", "--account", "../etc")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error for invalid account name")
	}
	output := string(out)
	if !strings.Contains(output, "invalid") {
		t.Errorf("expected 'invalid' error, got:\n%s", output)
	}
}
