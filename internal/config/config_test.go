package config

import (
	"os"
	"path/filepath"
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

	resolved := cfg.ResolveProfile("dev")

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

func TestResolveProfileNotFound(t *testing.T) {
	cfg := &Config{
		Project:  "myapp",
		Profiles: map[string]Section{},
	}

	resolved := cfg.ResolveProfile("nonexistent")

	if len(resolved.Vars) != 0 || len(resolved.Secrets) != 0 {
		t.Errorf("expected empty resolved section for nonexistent profile")
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
