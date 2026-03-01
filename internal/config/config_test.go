package config

import (
	"os"
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

// --- New tests for TOML config file support ---

func TestConfigDir(t *testing.T) {
	t.Run("XDG_CONFIG_HOME set", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "/xdg/config")
		dir, err := ConfigDir()
		if err != nil {
			t.Fatalf("ConfigDir() error: %v", err)
		}
		want := filepath.Join("/xdg/config", "jot")
		if dir != want {
			t.Errorf("ConfigDir() = %q, want %q", dir, want)
		}
	})

	t.Run("default fallback", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		dir, err := ConfigDir()
		if err != nil {
			t.Fatalf("ConfigDir() error: %v", err)
		}
		if !filepath.IsAbs(dir) {
			t.Errorf("ConfigDir() should return absolute path, got %q", dir)
		}
		if filepath.Base(dir) != "jot" {
			t.Errorf("ConfigDir() should end with 'jot', got %q", dir)
		}
	})
}

func TestConfigFilePath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/xdg/config")
	path, err := FilePath()
	if err != nil {
		t.Fatalf("FilePath() error: %v", err)
	}
	want := filepath.Join("/xdg/config", "jot", "config.toml")
	if path != want {
		t.Errorf("FilePath() = %q, want %q", path, want)
	}
}

func TestLoadFromTOML(t *testing.T) {
	tmp := t.TempDir()
	cfgDir := filepath.Join(tmp, "jot")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgFile := filepath.Join(cfgDir, "config.toml")
	toml := `[general]
editor = "nvim"
default_limit = 30

[display]
date_format = "absolute"
json = true

[sync]
dir = "/tmp/jot-sync"
`
	if err := os.WriteFile(cfgFile, []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("JOT_DB", "")
	t.Setenv("JOT_SYNC_DIR", "")
	t.Setenv("JOT_JSON", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Editor != "nvim" {
		t.Errorf("Editor = %q, want %q", cfg.Editor, "nvim")
	}
	if cfg.DefaultLimit != 30 {
		t.Errorf("DefaultLimit = %d, want %d", cfg.DefaultLimit, 30)
	}
	if cfg.DateFormat != "absolute" {
		t.Errorf("DateFormat = %q, want %q", cfg.DateFormat, "absolute")
	}
	if cfg.JSON != true {
		t.Error("JSON = false, want true")
	}
	if cfg.SyncDir != "/tmp/jot-sync" {
		t.Errorf("SyncDir = %q, want %q", cfg.SyncDir, "/tmp/jot-sync")
	}
}

func TestEnvVarsOverrideTOML(t *testing.T) {
	tmp := t.TempDir()
	cfgDir := filepath.Join(tmp, "jot")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgFile := filepath.Join(cfgDir, "config.toml")
	toml := `[sync]
dir = "/toml/sync"
`
	if err := os.WriteFile(cfgFile, []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("JOT_DB", "/env/jot.db")
	t.Setenv("JOT_SYNC_DIR", "/env/sync")
	t.Setenv("JOT_JSON", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.DBPath != "/env/jot.db" {
		t.Errorf("DBPath = %q, want %q (env should override)", cfg.DBPath, "/env/jot.db")
	}
	if cfg.SyncDir != "/env/sync" {
		t.Errorf("SyncDir = %q, want %q (env should override)", cfg.SyncDir, "/env/sync")
	}
	if cfg.JSON != true {
		t.Error("JSON = false, want true (env JOT_JSON should override)")
	}
}

func TestLoadNoConfigFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("JOT_DB", "")
	t.Setenv("JOT_SYNC_DIR", "")
	t.Setenv("JOT_JSON", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() should not error when config file is missing: %v", err)
	}

	// Should have sensible defaults
	if cfg.Editor != "" {
		t.Errorf("Editor = %q, want empty (no override)", cfg.Editor)
	}
	if cfg.DefaultLimit != 0 {
		t.Errorf("DefaultLimit = %d, want 0", cfg.DefaultLimit)
	}
	if cfg.DateFormat != "" {
		t.Errorf("DateFormat = %q, want empty", cfg.DateFormat)
	}
	if cfg.JSON != false {
		t.Error("JSON = true, want false")
	}
}

func TestLoadInvalidTOML(t *testing.T) {
	tmp := t.TempDir()
	cfgDir := filepath.Join(tmp, "jot")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgFile := filepath.Join(cfgDir, "config.toml")
	if err := os.WriteFile(cfgFile, []byte("this is not valid toml [[["), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("JOT_DB", "")
	t.Setenv("JOT_SYNC_DIR", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should return error for invalid TOML")
	}
}

func TestDefaultConfig(t *testing.T) {
	content := DefaultConfig()
	if content == "" {
		t.Fatal("DefaultConfig() should return non-empty string")
	}
	// Should contain key sections
	for _, want := range []string{"[general]", "[display]", "[sync]", "editor", "default_limit", "date_format", "json"} {
		if !containsString(content, want) {
			t.Errorf("DefaultConfig() missing %q", want)
		}
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
