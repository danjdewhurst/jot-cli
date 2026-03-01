package sync

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MigrateEncrypt re-encrypts existing plain .ndjson changeset files as .ndjson.age.
// Returns the number of files migrated.
func (s *Syncer) MigrateEncrypt() (int, error) {
	if s.passphrase == "" {
		return 0, fmt.Errorf("no passphrase configured")
	}

	changesetsDir := filepath.Join(s.syncDir, "changesets")
	entries, err := os.ReadDir(changesetsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("reading changesets directory: %w", err)
	}

	var migrated int
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".ndjson") || strings.HasSuffix(name, ".ndjson.age") {
			continue
		}

		srcPath := filepath.Join(changesetsDir, name)
		plaintext, err := os.ReadFile(srcPath)
		if err != nil {
			return migrated, fmt.Errorf("reading %s: %w", name, err)
		}

		ciphertext, err := Encrypt(s.passphrase, plaintext)
		if err != nil {
			return migrated, fmt.Errorf("encrypting %s: %w", name, err)
		}

		dstPath := srcPath + ".age"
		tmpPath := dstPath + ".tmp"

		if err := os.WriteFile(tmpPath, ciphertext, 0o600); err != nil {
			return migrated, fmt.Errorf("writing %s: %w", name+".age", err)
		}

		if err := os.Rename(tmpPath, dstPath); err != nil {
			_ = os.Remove(tmpPath)
			return migrated, fmt.Errorf("renaming %s: %w", name+".age", err)
		}

		if err := os.Remove(srcPath); err != nil {
			return migrated, fmt.Errorf("removing original %s: %w", name, err)
		}

		migrated++
	}

	return migrated, nil
}
