package cmd

import (
	"fmt"
	"os"

	"github.com/danjdewhurst/jot-cli/internal/config"
	"github.com/danjdewhurst/jot-cli/internal/store"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	flagJSON    bool
	flagDB      string
	flagVerbose bool

	cfg config.Config
	db  *store.Store
)

var rootCmd = &cobra.Command{
	Use:   "jot",
	Short: "A CLI-first notes app",
	Long:  "jot is a fast, context-aware notes tool for the terminal.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		cfg = config.Load()
		if flagDB != "" {
			cfg.DBPath = flagDB
		}

		// Commands that don't need DB
		if cmd.Name() == "version" {
			return nil
		}

		var err error
		db, err = store.Open(cfg.DBPath)
		if err != nil {
			return fmt.Errorf("opening database: %w", err)
		}
		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		if db != nil {
			return db.Close()
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if term.IsTerminal(int(os.Stdin.Fd())) {
			return tuiCmd.RunE(cmd, args)
		}
		return listCmd.RunE(cmd, args)
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "Output as JSON")
	rootCmd.PersistentFlags().StringVar(&flagDB, "db", "", "Database path")
	rootCmd.PersistentFlags().BoolVar(&flagVerbose, "verbose", false, "Verbose output")

	if v := os.Getenv("JOT_JSON"); v == "1" || v == "true" {
		flagJSON = true
	}
}
