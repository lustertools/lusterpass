package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/crypto/hkdf"
)

var validAccountName = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// ValidateAccountName checks that an account name is safe for use as a directory name.
func ValidateAccountName(name string) error {
	if name == "" {
		return fmt.Errorf("account name cannot be empty")
	}
	if !validAccountName.MatchString(name) {
		return fmt.Errorf("account name %q is invalid: must contain only letters, digits, hyphens, and underscores", name)
	}
	return nil
}

// machineKey derives a 32-byte AES key from hostname + uid.
func machineKey() []byte {
	hostname, _ := os.Hostname()
	uid := strconv.Itoa(syscall.Getuid())
	seed := hostname + ":" + uid

	h := sha256.New
	reader := hkdf.New(h, []byte(seed), []byte("lusterpass-auth"), []byte("machine-key"))

	key := make([]byte, 32)
	io.ReadFull(reader, key)
	return key
}

func encrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func decrypt(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func StoreToken(path string, token string) error {
	key := machineKey()
	encrypted, err := encrypt(key, []byte(token))
	if err != nil {
		return fmt.Errorf("encrypting token: %w", err)
	}
	return os.WriteFile(path, encrypted, 0600)
}

func LoadToken(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading token file: %w", err)
	}

	key := machineKey()
	plaintext, err := decrypt(key, data)
	if err != nil {
		return "", fmt.Errorf("decrypting token: %w", err)
	}

	return string(plaintext), nil
}

// ResolveToken returns the access token from env var or config file.
// Priority: $BWS_ACCESS_TOKEN > config file.
func ResolveToken(configPath string) (string, error) {
	if token := os.Getenv("BWS_ACCESS_TOKEN"); token != "" {
		return token, nil
	}

	return LoadToken(configPath)
}

// StoreOrgID saves the org ID as plain text (not a secret).
func StoreOrgID(path, orgID string) error {
	return os.WriteFile(path, []byte(orgID), 0600)
}

// LoadOrgID reads the cached org ID.
func LoadOrgID(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading org ID: %w", err)
	}
	return string(data), nil
}

// AccountDir returns the path to an account's directory.
func AccountDir(name string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".lusterpass", "accounts", name)
}

// ActiveAccountPath returns the path to the active account file.
func ActiveAccountPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".lusterpass", "active")
}

// LoadActiveAccount reads the active account name.
func LoadActiveAccount() (string, error) {
	data, err := os.ReadFile(ActiveAccountPath())
	if err != nil {
		return "", fmt.Errorf("no active account: %w", err)
	}
	name := strings.TrimSpace(string(data))
	if name == "" {
		return "", fmt.Errorf("active account file is empty")
	}
	return name, nil
}

// SetActiveAccount writes the active account name atomically.
func SetActiveAccount(name string) error {
	path := ActiveAccountPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(name), 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ListAccounts returns the names of all configured accounts.
func ListAccounts() ([]string, error) {
	home, _ := os.UserHomeDir()
	accountsDir := filepath.Join(home, ".lusterpass", "accounts")
	entries, err := os.ReadDir(accountsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var accounts []string
	for _, e := range entries {
		if e.IsDir() {
			accounts = append(accounts, e.Name())
		}
	}
	return accounts, nil
}

// ResolveTokenForAccount returns the access token for an account.
// Priority: $BWS_ACCESS_TOKEN env var > account's config file.
func ResolveTokenForAccount(account string) (string, error) {
	if token := os.Getenv("BWS_ACCESS_TOKEN"); token != "" {
		return token, nil
	}
	configPath := filepath.Join(AccountDir(account), "config")
	return LoadToken(configPath)
}

// ResolveOrgIDForAccount returns the org ID for an account.
// Priority: flagOverride (--org) > account's org file.
func ResolveOrgIDForAccount(account, flagOverride string) (string, error) {
	if flagOverride != "" {
		return flagOverride, nil
	}
	orgPath := filepath.Join(AccountDir(account), "org")
	return LoadOrgID(orgPath)
}
