package cache

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAndReadCache(t *testing.T) {
	dir := t.TempDir()

	secrets := map[string]string{
		"OPENAI_API_KEY": "sk-dev-xxx",
		"DB_PASSWORD":    "postgres-123",
	}

	accessToken := "0.test-token.xxxx"
	cachePath := filepath.Join(dir, "myapp", "dev.enc")

	err := Write(cachePath, accessToken, secrets)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	loaded, err := Read(cachePath, accessToken)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if loaded["OPENAI_API_KEY"] != "sk-dev-xxx" {
		t.Errorf("expected sk-dev-xxx, got %s", loaded["OPENAI_API_KEY"])
	}

	if loaded["DB_PASSWORD"] != "postgres-123" {
		t.Errorf("expected postgres-123, got %s", loaded["DB_PASSWORD"])
	}
}

func TestReadWrongToken(t *testing.T) {
	dir := t.TempDir()

	secrets := map[string]string{"KEY": "value"}
	cachePath := filepath.Join(dir, "test.enc")

	Write(cachePath, "correct-token", secrets)

	_, err := Read(cachePath, "wrong-token")
	if err == nil {
		t.Fatal("expected error when decrypting with wrong token")
	}
}

func TestReadNonexistent(t *testing.T) {
	_, err := Read("/nonexistent/path.enc", "token")
	if err == nil {
		t.Fatal("expected error for nonexistent cache")
	}
}

func TestCachePathIncludesAccount(t *testing.T) {
	home, _ := os.UserHomeDir()
	path := CachePath("personal", "myapp", "dev")
	expected := filepath.Join(home, ".lusterpass", "accounts", "personal", "cache", "myapp", "dev.enc")
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}
