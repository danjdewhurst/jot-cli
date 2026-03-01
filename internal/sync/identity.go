package sync

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// IdentityPath returns the default path for the sync identity file.
func IdentityPath() string {
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return filepath.Join(d, "jot", "sync.key")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".local", "share", "jot", "sync.key")
	}
	return filepath.Join(home, ".local", "share", "jot", "sync.key")
}

// GenerateIdentity generates a random passphrase and writes it to path
// with 0600 permissions. Parent directories are created as needed.
func GenerateIdentity(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating identity directory: %w", err)
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return fmt.Errorf("generating random passphrase: %w", err)
	}

	passphrase := hex.EncodeToString(b)
	if err := os.WriteFile(path, []byte(passphrase+"\n"), 0o600); err != nil {
		return fmt.Errorf("writing identity file: %w", err)
	}

	return nil
}

// LoadIdentity reads the passphrase from an identity file.
func LoadIdentity(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading identity file: %w", err)
	}

	passphrase := strings.TrimSpace(string(data))
	if passphrase == "" {
		return "", fmt.Errorf("identity file is empty: %s", path)
	}

	return passphrase, nil
}
