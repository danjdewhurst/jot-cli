package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds all resolved configuration values. Precedence:
// CLI flags > env vars > config file > defaults.
type Config struct {
	DBPath       string
	SyncDir      string
	Editor       string
	DefaultLimit int
	DateFormat   string
	JSON         bool
}

// tomlFile mirrors the TOML structure on disk.
type tomlFile struct {
	General tomlGeneral `toml:"general"`
	Display tomlDisplay `toml:"display"`
	Sync    tomlSync    `toml:"sync"`
}

type tomlGeneral struct {
	Editor       string `toml:"editor"`
	DefaultLimit int    `toml:"default_limit"`
}

type tomlDisplay struct {
	DateFormat string `toml:"date_format"`
	JSON       bool   `toml:"json"`
}

type tomlSync struct {
	Dir string `toml:"dir"`
}

// Load reads the config file (if present), then applies env var overrides.
func Load() (Config, error) {
	dataDir, err := dataDir()
	if err != nil {
		return Config{}, fmt.Errorf("resolving data directory: %w", err)
	}

	// Start from config file (if it exists)
	var tf tomlFile
	cfgPath, err := FilePath()
	if err != nil {
		return Config{}, fmt.Errorf("resolving config path: %w", err)
	}

	if _, statErr := os.Stat(cfgPath); statErr == nil {
		if _, decErr := toml.DecodeFile(cfgPath, &tf); decErr != nil {
			return Config{}, fmt.Errorf("parsing config file: %w", decErr)
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return Config{}, fmt.Errorf("checking config file: %w", statErr)
	}

	cfg := Config{
		DBPath:       filepath.Join(dataDir, "jot.db"),
		SyncDir:      filepath.Join(dataDir, "sync"),
		Editor:       tf.General.Editor,
		DefaultLimit: tf.General.DefaultLimit,
		DateFormat:   tf.Display.DateFormat,
		JSON:         tf.Display.JSON,
	}

	// TOML sync dir
	if tf.Sync.Dir != "" {
		cfg.SyncDir = expandHome(tf.Sync.Dir)
	}

	// Env var overrides (highest precedence after CLI flags)
	if p := os.Getenv("JOT_DB"); p != "" {
		cfg.DBPath = p
	}
	if p := os.Getenv("JOT_SYNC_DIR"); p != "" {
		cfg.SyncDir = p
	}
	if v := os.Getenv("JOT_JSON"); v == "1" || v == "true" {
		cfg.JSON = true
	}

	return cfg, nil
}

// ConfigDir returns the jot configuration directory path.
func ConfigDir() (string, error) {
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "jot"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determining home directory: %w", err)
	}
	return filepath.Join(home, ".config", "jot"), nil
}

// FilePath returns the full path to the config file.
func FilePath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}

// DefaultConfig returns a commented default config file.
func DefaultConfig() string {
	return `# jot-cli configuration
# Env vars (JOT_DB, JOT_SYNC_DIR, JOT_JSON) and CLI flags override these values.

[general]
# editor = "nvim"           # Override $EDITOR for jot only
# default_limit = 20        # Default --limit for list/log (0 = unlimited)

[display]
# date_format = "relative"  # "relative" | "absolute" | "iso"
# json = false              # Default to JSON output

[sync]
# dir = "~/Dropbox/jot-sync"
`
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

// expandHome replaces a leading ~ with the user's home directory.
func expandHome(path string) string {
	if len(path) < 2 || path[0] != '~' || path[1] != '/' {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[2:])
}
