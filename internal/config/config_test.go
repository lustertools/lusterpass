package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	yamlContent := `
project: myapp

common:
  vars:
    APP_NAME: myapp
    TZ: UTC
  secrets:
    SHARED_KEY: shared-key--myapp

profiles:
  dev:
    vars:
      LOG_LEVEL: debug
    secrets:
      OPENAI_API_KEY: openai-key--myapp--dev
      DB_PASSWORD: db-pass--myapp--dev
  prod:
    vars:
      LOG_LEVEL: warn
    secrets:
      OPENAI_API_KEY: openai-key--myapp--prod
`

	path := filepath.Join(dir, ".lusterpass.yaml")
	os.WriteFile(path, []byte(yamlContent), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Project != "myapp" {
		t.Errorf("expected project=myapp, got %s", cfg.Project)
	}

	if cfg.Common.Vars["APP_NAME"] != "myapp" {
		t.Errorf("expected common var APP_NAME=myapp")
	}

	if cfg.Common.Secrets["SHARED_KEY"] != "shared-key--myapp" {
		t.Errorf("expected common secret ref SHARED_KEY")
	}

	if cfg.Profiles["dev"].Secrets["OPENAI_API_KEY"] != "openai-key--myapp--dev" {
		t.Errorf("expected dev secret ref for OPENAI_API_KEY")
	}
}

func TestResolveProfile(t *testing.T) {
	cfg := &Config{
		Project: "myapp",
		Common: Section{
			Vars:    map[string]string{"APP_NAME": "myapp", "LOG_LEVEL": "info"},
			Secrets: map[string]string{"SHARED_KEY": "shared-key--myapp"},
		},
		Profiles: map[string]Section{
			"dev": {
				Vars:    map[string]string{"LOG_LEVEL": "debug"},
				Secrets: map[string]string{"API_KEY": "api-key--dev"},
			},
		},
	}

	resolved, err := cfg.ResolveProfile("dev")
	if err != nil {
		t.Fatalf("ResolveProfile(\"dev\"): %v", err)
	}

	if resolved.Vars["LOG_LEVEL"] != "debug" {
		t.Errorf("expected LOG_LEVEL=debug (profile override), got %s", resolved.Vars["LOG_LEVEL"])
	}

	if resolved.Vars["APP_NAME"] != "myapp" {
		t.Errorf("expected APP_NAME=myapp (from common), got %s", resolved.Vars["APP_NAME"])
	}

	if resolved.Secrets["SHARED_KEY"] != "shared-key--myapp" {
		t.Errorf("expected SHARED_KEY from common")
	}

	if resolved.Secrets["API_KEY"] != "api-key--dev" {
		t.Errorf("expected API_KEY from dev profile")
	}
}

func TestResolveProfileEmpty(t *testing.T) {
	cfg := &Config{
		Project: "myapp",
		Common: Section{
			Vars:    map[string]string{"APP_NAME": "myapp"},
			Secrets: map[string]string{"SHARED_KEY": "shared-key--myapp"},
		},
		Profiles: map[string]Section{
			"dev": {
				Secrets: map[string]string{"API_KEY": "api-key--dev"},
			},
		},
	}

	resolved, err := cfg.ResolveProfile("")
	if err != nil {
		t.Fatalf("ResolveProfile(\"\"): %v", err)
	}

	if resolved.Vars["APP_NAME"] != "myapp" {
		t.Errorf("expected APP_NAME from common")
	}
	if resolved.Secrets["SHARED_KEY"] != "shared-key--myapp" {
		t.Errorf("expected SHARED_KEY from common")
	}
	if _, ok := resolved.Secrets["API_KEY"]; ok {
		t.Errorf("expected dev profile secrets NOT included when no profile specified")
	}
}

func TestResolveProfileUnknownErrors(t *testing.T) {
	cfg := &Config{
		Project: "myapp",
		Profiles: map[string]Section{
			"dev":  {},
			"prod": {},
		},
	}

	_, err := cfg.ResolveProfile("staging")
	if err == nil {
		t.Fatal("expected error for unknown profile, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "staging") || !strings.Contains(msg, "dev") || !strings.Contains(msg, "prod") {
		t.Errorf("error should name the bad profile and list available; got: %s", msg)
	}
}

func TestResolveProfileUnknownNoProfilesDefined(t *testing.T) {
	cfg := &Config{
		Project:  "myapp",
		Profiles: map[string]Section{},
	}

	_, err := cfg.ResolveProfile("dev")
	if err == nil {
		t.Fatal("expected error when config has no profiles, got nil")
	}
	if !strings.Contains(err.Error(), "Run without --profile") {
		t.Errorf("error should hint to run without --profile; got: %s", err.Error())
	}
}

func TestLoadRejectsCommonAsProfileName(t *testing.T) {
	dir := t.TempDir()
	yamlContent := `
project: myapp
profiles:
  common:
    secrets:
      X: x-ref
`
	path := filepath.Join(dir, ".lusterpass.yaml")
	os.WriteFile(path, []byte(yamlContent), 0644)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for reserved profile name 'common', got nil")
	}
	if !strings.Contains(err.Error(), "reserved") {
		t.Errorf("error should explain that common is reserved; got: %s", err.Error())
	}
}

func TestCacheKey(t *testing.T) {
	cfg := &Config{}
	if got := cfg.CacheKey(""); got != "common" {
		t.Errorf("CacheKey(\"\") = %q, want %q", got, "common")
	}
	if got := cfg.CacheKey("dev"); got != "dev" {
		t.Errorf("CacheKey(\"dev\") = %q, want %q", got, "dev")
	}
}

func TestLoadConfigWithAccount(t *testing.T) {
	dir := t.TempDir()
	yamlContent := `
account: personal
project: myapp
common:
  vars:
    APP_NAME: myapp
profiles:
  dev:
    vars:
      LOG_LEVEL: debug
`
	path := filepath.Join(dir, ".lusterpass.yaml")
	os.WriteFile(path, []byte(yamlContent), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Account != "personal" {
		t.Errorf("expected account=personal, got %q", cfg.Account)
	}
}

func TestLoadConfigWithoutAccount(t *testing.T) {
	dir := t.TempDir()
	yamlContent := `
project: myapp
common:
  vars:
    APP_NAME: myapp
`
	path := filepath.Join(dir, ".lusterpass.yaml")
	os.WriteFile(path, []byte(yamlContent), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Account != "" {
		t.Errorf("expected empty account, got %q", cfg.Account)
	}
}
