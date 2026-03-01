package sync

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateAndLoadIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sync.key")

	if err := GenerateIdentity(path); err != nil {
		t.Fatalf("GenerateIdentity: %v", err)
	}

	passphrase, err := LoadIdentity(path)
	if err != nil {
		t.Fatalf("LoadIdentity: %v", err)
	}

	if len(passphrase) != 64 { // 32 bytes as hex = 64 chars
		t.Errorf("passphrase length = %d, want 64 hex chars", len(passphrase))
	}
}

func TestGenerateIdentityFilePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sync.key")

	if err := GenerateIdentity(path); err != nil {
		t.Fatalf("GenerateIdentity: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0o600 {
		t.Errorf("file permissions = %o, want 0600", perm)
	}
}

func TestLoadIdentityMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.key")

	_, err := LoadIdentity(path)
	if err == nil {
		t.Error("LoadIdentity should return error for missing file")
	}
}

func TestGenerateIdentityCreatesParentDirs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "dir", "sync.key")

	if err := GenerateIdentity(path); err != nil {
		t.Fatalf("GenerateIdentity: %v", err)
	}

	passphrase, err := LoadIdentity(path)
	if err != nil {
		t.Fatalf("LoadIdentity: %v", err)
	}

	if passphrase == "" {
		t.Error("passphrase should not be empty")
	}
}

func TestGenerateIdentityUnique(t *testing.T) {
	path1 := filepath.Join(t.TempDir(), "key1")
	path2 := filepath.Join(t.TempDir(), "key2")

	if err := GenerateIdentity(path1); err != nil {
		t.Fatalf("GenerateIdentity 1: %v", err)
	}
	if err := GenerateIdentity(path2); err != nil {
		t.Fatalf("GenerateIdentity 2: %v", err)
	}

	p1, _ := LoadIdentity(path1)
	p2, _ := LoadIdentity(path2)

	if p1 == p2 {
		t.Error("two generated identities should have different passphrases")
	}
}

func TestIdentityPath(t *testing.T) {
	path := IdentityPath()
	if path == "" {
		t.Error("IdentityPath should return a non-empty path")
	}
}
