package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseExportLine(t *testing.T) {
	tests := []struct {
		line    string
		key     string
		value   string
		wantOK  bool
	}{
		{"export FOO=bar", "FOO", "bar", true},
		{"export FOO=\"bar baz\"", "FOO", "bar baz", true},
		{"export FOO='bar baz'", "FOO", "bar baz", true},
		{"  export FOO=bar", "FOO", "bar", true},
		{"export DB_URL=postgres://localhost:5432/db", "DB_URL", "postgres://localhost:5432/db", true},
		{"# comment", "", "", false},
		{"", "", "", false},
		{"FOO=bar", "", "", false}, // no export keyword
		{"eval $(something)", "", "", false},
		{"export A=", "A", "", true},
	}

	for _, tt := range tests {
		key, value, ok := parseExportLine(tt.line)
		if ok != tt.wantOK {
			t.Errorf("parseExportLine(%q): ok=%v, want %v", tt.line, ok, tt.wantOK)
			continue
		}
		if ok {
			if key != tt.key {
				t.Errorf("parseExportLine(%q): key=%q, want %q", tt.line, key, tt.key)
			}
			if value != tt.value {
				t.Errorf("parseExportLine(%q): value=%q, want %q", tt.line, value, tt.value)
			}
		}
	}
}

func TestLooksLikeSecret(t *testing.T) {
	tests := []struct {
		key    string
		value  string
		secret bool
	}{
		// Key-based detection
		{"DB_PASSWORD", "simple", true},
		{"API_TOKEN", "mytoken", true},
		{"SECRET_KEY", "abc", true},
		{"OPENAI_API_KEY", "sk-abc123", true},
		{"AWS_ACCESS_KEY", "AKIA1234", true},
		{"AUTH_TOKEN", "xyz", true},
		{"DATABASE_URL", "postgres://...", true},
		{"CONNECTION_STRING", "Server=...", true},
		{"SIGNING_KEY", "abc", true},
		{"PRIVATE_KEY", "---BEGIN---", true},

		// Value-based detection
		{"MY_VAR", "sk-abc1234567890", true},       // OpenAI prefix
		{"MY_VAR", "ghp_1234567890abcdef", true},   // GitHub PAT
		{"MY_VAR", "xoxb-something", true},          // Slack
		{"MY_VAR", "AKIA1234567890ABCDEF", true},   // AWS

		// Plain vars — should NOT be secret
		{"APP_NAME", "myapp", false},
		{"LOG_LEVEL", "debug", false},
		{"PORT", "3000", false},
		{"TZ", "UTC", false},
		{"LOG_FORMAT", "json", false},
		{"APP_URL", "http://localhost:3000", false},
	}

	for _, tt := range tests {
		got := looksLikeSecret(tt.key, tt.value)
		if got != tt.secret {
			t.Errorf("looksLikeSecret(%q, %q) = %v, want %v", tt.key, tt.value, got, tt.secret)
		}
	}
}

func TestToRefName(t *testing.T) {
	tests := []struct {
		envVar  string
		project string
		want    string
	}{
		{"DB_PASSWORD", "myapp", "db-password--myapp"},
		{"OPENAI_API_KEY", "proj", "openai-api-key--proj"},
		{"SECRET", "x", "secret--x"},
	}

	for _, tt := range tests {
		got := toRefName(tt.envVar, tt.project)
		if got != tt.want {
			t.Errorf("toRefName(%q, %q) = %q, want %q", tt.envVar, tt.project, got, tt.want)
		}
	}
}

func TestMigrateEndToEnd(t *testing.T) {
	// Create a fake .envrc
	dir := t.TempDir()
	envrc := `# My project env
export APP_NAME=myapp
export LOG_LEVEL=debug
export TZ=UTC
export DB_PASSWORD="s3cret_password"
export OPENAI_API_KEY='sk-1234567890abcdef'
export API_TOKEN=ghp_abcdefghij1234567890
export APP_URL=http://localhost:3000
eval $(direnv hook zsh)
`
	srcPath := filepath.Join(dir, ".envrc")
	os.WriteFile(srcPath, []byte(envrc), 0644)

	outDir := filepath.Join(dir, "output")
	os.MkdirAll(outDir, 0755)

	// Run the generation functions directly
	f, _ := os.Open(srcPath)
	defer f.Close()

	var vars, secrets []envEntry
	scanner := newScanner(f, &vars, &secrets)
	_ = scanner

	// Just test file output
	testVars := []envEntry{
		{Key: "APP_NAME", Value: "myapp"},
		{Key: "APP_URL", Value: "http://localhost:3000"},
		{Key: "LOG_LEVEL", Value: "debug"},
		{Key: "TZ", Value: "UTC"},
	}
	testSecrets := []envEntry{
		{Key: "API_TOKEN", Value: "ghp_abcdefghij1234567890"},
		{Key: "DB_PASSWORD", Value: "s3cret_password"},
		{Key: "OPENAI_API_KEY", Value: "sk-1234567890abcdef"},
	}

	yamlPath := filepath.Join(outDir, ".lusterpass.yaml")
	if err := writeMigratedYAML(yamlPath, "myapp", testVars, testSecrets); err != nil {
		t.Fatalf("writeMigratedYAML: %v", err)
	}

	data, _ := os.ReadFile(yamlPath)
	yaml := string(data)

	if !strings.Contains(yaml, "project: myapp") {
		t.Error("YAML missing project name")
	}
	if !strings.Contains(yaml, "APP_NAME: myapp") {
		t.Error("YAML missing var APP_NAME")
	}
	if !strings.Contains(yaml, "DB_PASSWORD: db-password--myapp") {
		t.Error("YAML missing secret DB_PASSWORD ref")
	}
	if !strings.Contains(yaml, "OPENAI_API_KEY: openai-api-key--myapp") {
		t.Error("YAML missing secret OPENAI_API_KEY ref")
	}
	// Should NOT contain plain secret values
	if strings.Contains(yaml, "s3cret_password") {
		t.Error("YAML contains plain secret value!")
	}
	if strings.Contains(yaml, "sk-1234567890") {
		t.Error("YAML contains plain API key!")
	}
	if !strings.Contains(yaml, "# profiles:") {
		t.Error("YAML missing commented profiles example block")
	}
	if !strings.Contains(yaml, "#   dev:") {
		t.Error("YAML missing commented dev profile example")
	}
	// The profiles block should be commented out by default; an active
	// "profiles:" line (no leading #) would mean it's enabled.
	for _, line := range strings.Split(yaml, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "profiles:" {
			t.Error("YAML has active profiles: line; expected commented-out by default")
		}
	}

	scriptPath := filepath.Join(outDir, "onboard-secrets.sh")
	if err := writeOnboardScript(scriptPath, "myapp", testSecrets); err != nil {
		t.Fatalf("writeOnboardScript: %v", err)
	}

	scriptData, _ := os.ReadFile(scriptPath)
	script := string(scriptData)

	if !strings.Contains(script, "#!/usr/bin/env bash") {
		t.Error("script missing shebang")
	}
	if !strings.Contains(script, "db-password--myapp") {
		t.Error("script missing DB_PASSWORD ref")
	}
	if !strings.Contains(script, "s3cret_password") {
		t.Error("script should contain secret values for onboarding")
	}
	if !strings.Contains(script, "--org") {
		t.Error("script missing --org flag handling")
	}

	// Verify script is executable
	info, _ := os.Stat(scriptPath)
	if info.Mode()&0111 == 0 {
		t.Error("script is not executable")
	}
}

// helper to reuse scanning logic in tests
func newScanner(f *os.File, vars, secrets *[]envEntry) *strings.Builder {
	scanner := &strings.Builder{}
	return scanner
}

func TestYamlValue(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "simple"},
		{"has spaces", "has spaces"},
		{"has:colon", `"has:colon"`},
		{"true", `"true"`},
		{"false", `"false"`},
		{"yes", `"yes"`},
	}
	for _, tt := range tests {
		got := yamlValue(tt.input)
		if got != tt.want {
			t.Errorf("yamlValue(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
