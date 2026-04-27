package cache

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/crypto/hkdf"
)

func deriveKey(accessToken string) []byte {
	h := sha256.New
	reader := hkdf.New(h, []byte(accessToken), []byte("lusterpass-cache"), []byte("cache-key"))
	key := make([]byte, 32)
	io.ReadFull(reader, key)
	return key
}

func Write(path string, accessToken string, secrets map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("creating cache dir: %w", err)
	}

	data, err := json.Marshal(secrets)
	if err != nil {
		return fmt.Errorf("marshaling secrets: %w", err)
	}

	key := deriveKey(accessToken)
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}

	encrypted := gcm.Seal(nonce, nonce, data, nil)
	return os.WriteFile(path, encrypted, 0600)
}

func Read(path string, accessToken string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading cache: %w", err)
	}

	key := deriveKey(accessToken)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("cache file too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypting cache: %w", err)
	}

	var secrets map[string]string
	if err := json.Unmarshal(plaintext, &secrets); err != nil {
		return nil, fmt.Errorf("parsing cache: %w", err)
	}

	return secrets, nil
}

// CachePath returns the standard cache file path for an account/project/profile.
func CachePath(account, project, profile string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".lusterpass", "accounts", account, "cache", project, profile+".enc")
}
