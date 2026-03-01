package config

import (
	"path/filepath"
	"testing"
)

func TestLoadJOTDBOverride(t *testing.T) {
	t.Setenv("JOT_DB", "/custom/path/jot.db")
	t.Setenv("JOT_SYNC_DIR", "") // clear to avoid interference

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}
	if cfg.DBPath != "/custom/path/jot.db" {
		t.Errorf("DBPath = %q, want %q", cfg.DBPath, "/custom/path/jot.db")
	}
}

func TestLoadJOTSyncDirOverride(t *testing.T) {
	t.Setenv("JOT_SYNC_DIR", "/custom/sync")
	t.Setenv("JOT_DB", "") // clear to avoid interference

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}
	if cfg.SyncDir != "/custom/sync" {
		t.Errorf("SyncDir = %q, want %q", cfg.SyncDir, "/custom/sync")
	}
}

func TestLoadXDGDataHomeOverride(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/xdg/data")
	t.Setenv("JOT_DB", "")
	t.Setenv("JOT_SYNC_DIR", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	wantDB := filepath.Join("/xdg/data", "jot", "jot.db")
	if cfg.DBPath != wantDB {
		t.Errorf("DBPath = %q, want %q", cfg.DBPath, wantDB)
	}

	wantSync := filepath.Join("/xdg/data", "jot", "sync")
	if cfg.SyncDir != wantSync {
		t.Errorf("SyncDir = %q, want %q", cfg.SyncDir, wantSync)
	}
}

func TestLoadDefaultFallback(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("JOT_DB", "")
	t.Setenv("JOT_SYNC_DIR", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	// Should fall back to ~/.local/share/jot
	if cfg.DBPath == "" {
		t.Error("DBPath should not be empty with default fallback")
	}
	if !filepath.IsAbs(cfg.DBPath) {
		t.Errorf("DBPath should be absolute, got %q", cfg.DBPath)
	}
	if filepath.Base(cfg.DBPath) != "jot.db" {
		t.Errorf("DBPath should end with jot.db, got %q", cfg.DBPath)
	}
	if filepath.Base(cfg.SyncDir) != "sync" {
		t.Errorf("SyncDir should end with sync, got %q", cfg.SyncDir)
	}
}
