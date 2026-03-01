package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	DBPath  string
	SyncDir string
}

func Load() Config {
	return Config{
		DBPath:  dbPath(),
		SyncDir: syncDir(),
	}
}

func syncDir() string {
	if p := os.Getenv("JOT_SYNC_DIR"); p != "" {
		return p
	}
	return filepath.Join(dataDir(), "sync")
}

func dbPath() string {
	if p := os.Getenv("JOT_DB"); p != "" {
		return p
	}
	return filepath.Join(dataDir(), "jot.db")
}

func dataDir() string {
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return filepath.Join(d, "jot")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "jot")
}
