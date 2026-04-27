package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lustertools/lusterpass/internal/auth"
	"github.com/lustertools/lusterpass/internal/config"
)

func TestResolveAccountFromConfig(t *testing.T) {
	cfg := &config.Config{Account: "personal", Project: "test"}
	account, err := resolveAccount(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if account != "personal" {
		t.Errorf("expected personal, got %s", account)
	}
}

func TestResolveAccountFromActiveFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	os.MkdirAll(filepath.Join(dir, ".lusterpass", "accounts", "company"), 0700)
	auth.SetActiveAccount("company")

	cfg := &config.Config{Project: "test"} // no Account set
	account, err := resolveAccount(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if account != "company" {
		t.Errorf("expected company, got %s", account)
	}
}

func TestResolveAccountConfigOverridesActive(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	os.MkdirAll(filepath.Join(dir, ".lusterpass", "accounts", "company"), 0700)
	auth.SetActiveAccount("company")

	cfg := &config.Config{Account: "personal", Project: "test"}
	account, err := resolveAccount(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if account != "personal" {
		t.Errorf("expected personal (config override), got %s", account)
	}
}

func TestResolveAccountNilConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	os.MkdirAll(filepath.Join(dir, ".lusterpass", "accounts", "default"), 0700)
	auth.SetActiveAccount("default")

	account, err := resolveAccount(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if account != "default" {
		t.Errorf("expected default, got %s", account)
	}
}

func TestResolveAccountNoAccountError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cfg := &config.Config{Project: "test"}
	_, err := resolveAccount(cfg)
	if err == nil {
		t.Fatal("expected error when no account configured")
	}
}
