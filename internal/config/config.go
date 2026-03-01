package config

import (
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	DBPath  string
	SyncDir string
}

func Load() (Config, error) {
	dir, err := dataDir()
	if err != nil {
		return Config{}, fmt.Errorf("resolving data directory: %w", err)
	}

	return Config{
		DBPath:  dbPath(dir),
		SyncDir: syncDir(dir),
	}, nil
}

func syncDir(dataDir string) string {
	if p := os.Getenv("JOT_SYNC_DIR"); p != "" {
		return p
	}
	return filepath.Join(dataDir, "sync")
}

func dbPath(dataDir string) string {
	if p := os.Getenv("JOT_DB"); p != "" {
		return p
	}
	return filepath.Join(dataDir, "jot.db")
}

func dataDir() (string, error) {
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return filepath.Join(d, "jot"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determining home directory: %w", err)
	}
	return filepath.Join(home, ".local", "share", "jot"), nil
}
