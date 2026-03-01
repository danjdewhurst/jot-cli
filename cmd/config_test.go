package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigPrintsResolvedConfig(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("JOT_DB", "")
	t.Setenv("JOT_SYNC_DIR", "")
	t.Setenv("JOT_JSON", "")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"config"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("config command failed: %v", err)
	}

	out := buf.String()
	// Should contain key config fields
	for _, want := range []string{"db_path", "sync_dir", "date_format", "json"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestConfigPathFlag(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("JOT_DB", "")
	t.Setenv("JOT_SYNC_DIR", "")
	t.Setenv("JOT_JSON", "")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"config", "--path"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("config --path failed: %v", err)
	}

	out := strings.TrimSpace(buf.String())
	want := filepath.Join(tmp, "jot", "config.toml")
	if out != want {
		t.Errorf("config --path = %q, want %q", out, want)
	}
}

func TestConfigInitCreatesFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("JOT_DB", "")
	t.Setenv("JOT_SYNC_DIR", "")
	t.Setenv("JOT_JSON", "")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"config", "init"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("config init failed: %v", err)
	}

	cfgFile := filepath.Join(tmp, "jot", "config.toml")
	data, err := os.ReadFile(cfgFile)
	if err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "[general]") {
		t.Error("config file missing [general] section")
	}
	if !strings.Contains(content, "[display]") {
		t.Error("config file missing [display] section")
	}
	if !strings.Contains(content, "[sync]") {
		t.Error("config file missing [sync] section")
	}
}

func TestConfigInitDoesNotOverwrite(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("JOT_DB", "")
	t.Setenv("JOT_SYNC_DIR", "")
	t.Setenv("JOT_JSON", "")

	// Pre-create a config file
	cfgDir := filepath.Join(tmp, "jot")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgFile := filepath.Join(cfgDir, "config.toml")
	existing := "# my custom config\n"
	if err := os.WriteFile(cfgFile, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"config", "init"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("config init should error when file already exists")
	}

	// Existing file should be untouched
	data, _ := os.ReadFile(cfgFile)
	if string(data) != existing {
		t.Error("existing config file was overwritten")
	}
}
