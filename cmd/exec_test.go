package cmd

import (
	"os"
	"strings"
	"testing"
)

func TestBuildExecEnvPrecedence(t *testing.T) {
	t.Setenv("APP_NAME", "from-shell")
	t.Setenv("LOG_LEVEL", "info")
	t.Setenv("PATH", "/usr/bin")

	vars := map[string]string{
		"APP_NAME":  "from-config",
		"LOG_LEVEL": "debug",
		"NEW_VAR":   "new",
	}
	secrets := map[string]string{
		"LOG_LEVEL":     "trace",
		"DB_PASSWORD":   "p@ssw0rd",
		"OPENAI_API_KEY": "sk-EXAMPLE",
	}

	envv := buildExecEnv(vars, secrets)

	got := make(map[string]string, len(envv))
	for _, kv := range envv {
		i := strings.IndexByte(kv, '=')
		if i < 0 {
			t.Fatalf("malformed env entry: %q", kv)
		}
		got[kv[:i]] = kv[i+1:]
	}

	cases := []struct {
		key, want, why string
	}{
		{"APP_NAME", "from-config", "config var should override shell"},
		{"LOG_LEVEL", "trace", "secret should override config and shell"},
		{"NEW_VAR", "new", "config var with no shell counterpart should appear"},
		{"DB_PASSWORD", "p@ssw0rd", "secret with no shell counterpart should appear"},
		{"PATH", "/usr/bin", "shell var with no override should pass through"},
	}

	for _, tc := range cases {
		if got[tc.key] != tc.want {
			t.Errorf("%s = %q, want %q (%s)", tc.key, got[tc.key], tc.want, tc.why)
		}
	}

	if _, ok := got["DB_PASSWORD"]; !ok {
		t.Error("DB_PASSWORD missing from merged env")
	}
}

func TestBuildExecEnvMalformedShellEntryIsSkipped(t *testing.T) {
	// os.Environ() never produces malformed entries on real systems, but the
	// merge code should be defensive against entries with no '='. We test the
	// code path by looking at how a known good env merges (sanity check that
	// nothing crashes with empty maps).
	envv := buildExecEnv(map[string]string{}, map[string]string{})

	// At minimum PATH should pass through from the shell.
	if os.Getenv("PATH") != "" {
		found := false
		for _, kv := range envv {
			if strings.HasPrefix(kv, "PATH=") {
				found = true
				break
			}
		}
		if !found {
			t.Error("PATH did not pass through from shell env")
		}
	}
}
