package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMachineKey(t *testing.T) {
	key1 := machineKey()
	key2 := machineKey()

	if len(key1) != 32 {
		t.Errorf("expected 32-byte key, got %d", len(key1))
	}

	// Deterministic on same machine
	for i := range key1 {
		if key1[i] != key2[i] {
			t.Fatal("machineKey not deterministic")
		}
	}
}

func TestStoreAndLoadToken(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")

	token := "0.test-access-token.xxxx"

	err := StoreToken(configPath, token)
	if err != nil {
		t.Fatalf("StoreToken failed: %v", err)
	}

	// File should exist and not contain plaintext token
	data, _ := os.ReadFile(configPath)
	if string(data) == token {
		t.Fatal("token stored in plaintext!")
	}

	loaded, err := LoadToken(configPath)
	if err != nil {
		t.Fatalf("LoadToken failed: %v", err)
	}

	if loaded != token {
		t.Errorf("expected %q, got %q", token, loaded)
	}
}

func TestLoadTokenFileNotFound(t *testing.T) {
	_, err := LoadToken("/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestResolveTokenEnvVar(t *testing.T) {
	t.Setenv("BWS_ACCESS_TOKEN", "env-token-123")

	token, err := ResolveToken("/nonexistent/path")
	if err != nil {
		t.Fatalf("ResolveToken failed: %v", err)
	}

	if token != "env-token-123" {
		t.Errorf("expected env-token-123, got %s", token)
	}
}

func TestResolveTokenFallsBackToFile(t *testing.T) {
	t.Setenv("BWS_ACCESS_TOKEN", "")

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")

	StoreToken(configPath, "file-token-456")

	token, err := ResolveToken(configPath)
	if err != nil {
		t.Fatalf("ResolveToken failed: %v", err)
	}

	if token != "file-token-456" {
		t.Errorf("expected file-token-456, got %s", token)
	}
}

func TestStoreAndLoadOrgID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "org")

	orgID := "a1e4a796-78c7-41c7-8d7c-b40a00cf6392"

	if err := StoreOrgID(path, orgID); err != nil {
		t.Fatalf("StoreOrgID failed: %v", err)
	}

	loaded, err := LoadOrgID(path)
	if err != nil {
		t.Fatalf("LoadOrgID failed: %v", err)
	}

	if loaded != orgID {
		t.Errorf("expected %q, got %q", orgID, loaded)
	}
}

func TestLoadOrgIDNotFound(t *testing.T) {
	_, err := LoadOrgID("/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestValidateAccountName(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"personal", false},
		{"company", false},
		{"my-account", false},
		{"my_account", false},
		{"Account123", false},
		{"", true},
		{" ", true},
		{"../etc", true},
		{"foo/bar", true},
		{"foo bar", true},
		{".hidden", true},
		{"a.b", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAccountName(tt.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAccountName(%q) error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
		})
	}
}

func TestAccountDir(t *testing.T) {
	home, _ := os.UserHomeDir()
	dir := AccountDir("personal")
	expected := filepath.Join(home, ".lusterpass", "accounts", "personal")
	if dir != expected {
		t.Errorf("expected %q, got %q", expected, dir)
	}
}

func TestSetAndLoadActiveAccount(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	err := SetActiveAccount("personal")
	if err != nil {
		t.Fatalf("SetActiveAccount failed: %v", err)
	}

	loaded, err := LoadActiveAccount()
	if err != nil {
		t.Fatalf("LoadActiveAccount failed: %v", err)
	}
	if loaded != "personal" {
		t.Errorf("expected personal, got %q", loaded)
	}

	// Overwrite with a different account
	err = SetActiveAccount("company")
	if err != nil {
		t.Fatalf("SetActiveAccount failed: %v", err)
	}
	loaded, _ = LoadActiveAccount()
	if loaded != "company" {
		t.Errorf("expected company, got %q", loaded)
	}
}

func TestListAccounts(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	os.MkdirAll(filepath.Join(dir, ".lusterpass", "accounts", "personal"), 0700)
	os.MkdirAll(filepath.Join(dir, ".lusterpass", "accounts", "company"), 0700)

	accounts, err := ListAccounts()
	if err != nil {
		t.Fatalf("ListAccounts failed: %v", err)
	}

	if len(accounts) != 2 {
		t.Errorf("expected 2 accounts, got %d: %v", len(accounts), accounts)
	}
}

func TestListAccountsEmpty(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	accounts, err := ListAccounts()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(accounts) != 0 {
		t.Errorf("expected 0 accounts, got %d", len(accounts))
	}
}

func TestResolveTokenForAccount(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("BWS_ACCESS_TOKEN", "")

	accountDir := filepath.Join(dir, ".lusterpass", "accounts", "testacct")
	os.MkdirAll(accountDir, 0700)
	StoreToken(filepath.Join(accountDir, "config"), "acct-token-123")

	token, err := ResolveTokenForAccount("testacct")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "acct-token-123" {
		t.Errorf("expected acct-token-123, got %s", token)
	}
}

func TestResolveTokenForAccountEnvOverride(t *testing.T) {
	t.Setenv("BWS_ACCESS_TOKEN", "env-override-token")

	token, err := ResolveTokenForAccount("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "env-override-token" {
		t.Errorf("expected env-override-token, got %s", token)
	}
}

func TestResolveOrgIDForAccount(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	accountDir := filepath.Join(dir, ".lusterpass", "accounts", "testacct")
	os.MkdirAll(accountDir, 0700)
	StoreOrgID(filepath.Join(accountDir, "org"), "test-org-id")

	// Flag override takes priority
	orgID, err := ResolveOrgIDForAccount("testacct", "flag-org-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orgID != "flag-org-id" {
		t.Errorf("expected flag-org-id, got %s", orgID)
	}

	// Without flag, falls back to file
	orgID, err = ResolveOrgIDForAccount("testacct", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orgID != "test-org-id" {
		t.Errorf("expected test-org-id, got %s", orgID)
	}
}
