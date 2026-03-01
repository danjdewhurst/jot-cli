package cmd

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/danjdewhurst/jot-cli/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show or manage configuration",
	Long:  "Print resolved configuration, show config file path, or create a default config file.",
	RunE: func(cmd *cobra.Command, args []string) error {
		showPath, _ := cmd.Flags().GetBool("path")
		if showPath {
			path, err := config.FilePath()
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), path)
			return nil
		}

		// Print resolved config as TOML
		resolved := struct {
			DBPath       string `toml:"db_path"`
			SyncDir      string `toml:"sync_dir"`
			Editor       string `toml:"editor"`
			DefaultLimit int    `toml:"default_limit"`
			DateFormat   string `toml:"date_format"`
			JSON         bool   `toml:"json"`
		}{
			DBPath:       cfg.DBPath,
			SyncDir:      cfg.SyncDir,
			Editor:       cfg.Editor,
			DefaultLimit: cfg.DefaultLimit,
			DateFormat:   cfg.DateFormat,
			JSON:         cfg.JSON,
		}

		return toml.NewEncoder(cmd.OutOrStdout()).Encode(resolved)
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a default config file",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := config.FilePath()
		if err != nil {
			return err
		}

		if _, statErr := os.Stat(path); statErr == nil {
			return fmt.Errorf("config file already exists: %s", path)
		}

		dir, err := config.ConfigDir()
		if err != nil {
			return err
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating config directory: %w", err)
		}

		if err := os.WriteFile(path, []byte(config.DefaultConfig()), 0o644); err != nil {
			return fmt.Errorf("writing config file: %w", err)
		}

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created %s\n", path)
		return nil
	},
}

func init() {
	configCmd.Flags().Bool("path", false, "Print config file path")
	configCmd.AddCommand(configInitCmd)
	rootCmd.AddCommand(configCmd)
}
