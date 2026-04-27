package cmd

import (
	"strings"
	"testing"
)

func TestFormatExports(t *testing.T) {
	vars := map[string]string{
		"APP_NAME":  "myapp",
		"LOG_LEVEL": "debug",
	}

	secrets := map[string]string{
		"API_KEY": "sk-xxx",
	}

	output := formatExports(vars, secrets)

	if !strings.Contains(output, "export APP_NAME='myapp'") {
		t.Errorf("missing APP_NAME export, got:\n%s", output)
	}
	if !strings.Contains(output, "export LOG_LEVEL='debug'") {
		t.Errorf("missing LOG_LEVEL export, got:\n%s", output)
	}
	if !strings.Contains(output, "export API_KEY='sk-xxx'") {
		t.Errorf("missing API_KEY export, got:\n%s", output)
	}
}

func TestFormatExportsEscaping(t *testing.T) {
	vars := map[string]string{}
	secrets := map[string]string{
		"PASS": "my'pass$word",
	}

	output := formatExports(vars, secrets)

	// Single quotes in value should be escaped with '\'' trick
	if !strings.Contains(output, "PASS=") {
		t.Errorf("missing PASS export, got:\n%s", output)
	}
	// Should not contain unescaped single quote within the value
	if strings.Contains(output, "my'pass") {
		t.Errorf("single quote not escaped properly, got:\n%s", output)
	}
}

func TestFormatExportsSecretOverridesVar(t *testing.T) {
	vars := map[string]string{
		"KEY": "from-vars",
	}
	secrets := map[string]string{
		"KEY": "from-secrets",
	}

	output := formatExports(vars, secrets)

	if !strings.Contains(output, "export KEY='from-secrets'") {
		t.Errorf("secret should override var, got:\n%s", output)
	}
}

func TestEnvTTYGuardRefusesWhenStdoutIsTerminal(t *testing.T) {
	originalIsTerminal := stdoutIsTerminal
	defer func() { stdoutIsTerminal = originalIsTerminal }()
	stdoutIsTerminal = func() bool { return true }

	envProfile = ""
	err := envCmd.RunE(envCmd, []string{})
	if err == nil {
		t.Fatal("expected error when stdout is a TTY, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "refusing to print") {
		t.Errorf("error should explain why we refuse; got: %s", msg)
	}
	if !strings.Contains(msg, `eval "$(lusterpass env)"`) {
		t.Errorf("error should suggest eval; got: %s", msg)
	}
	if !strings.Contains(msg, "lusterpass exec") {
		t.Errorf("error should suggest exec; got: %s", msg)
	}
}

func TestEnvTTYGuardErrorIncludesProfile(t *testing.T) {
	originalIsTerminal := stdoutIsTerminal
	defer func() { stdoutIsTerminal = originalIsTerminal }()
	stdoutIsTerminal = func() bool { return true }

	envProfile = "dev"
	defer func() { envProfile = "" }()
	err := envCmd.RunE(envCmd, []string{})
	if err == nil {
		t.Fatal("expected error when stdout is a TTY, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "--profile dev") {
		t.Errorf("error should preserve --profile flag in suggestion; got: %s", msg)
	}
}

func TestProfileSuffix(t *testing.T) {
	if got := profileSuffix(""); got != "" {
		t.Errorf("profileSuffix(\"\") = %q, want \"\"", got)
	}
	if got := profileSuffix("dev"); got != " --profile dev" {
		t.Errorf("profileSuffix(\"dev\") = %q, want \" --profile dev\"", got)
	}
}

func TestFormatExportsSorted(t *testing.T) {
	vars := map[string]string{
		"ZZZ": "last",
		"AAA": "first",
		"MMM": "middle",
	}

	output := formatExports(vars, nil)
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if !strings.HasPrefix(lines[0], "export AAA=") {
		t.Errorf("expected AAA first, got %s", lines[0])
	}
	if !strings.HasPrefix(lines[1], "export MMM=") {
		t.Errorf("expected MMM second, got %s", lines[1])
	}
	if !strings.HasPrefix(lines[2], "export ZZZ=") {
		t.Errorf("expected ZZZ third, got %s", lines[2])
	}
}
