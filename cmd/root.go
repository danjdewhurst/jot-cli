package cmd

import (
	"fmt"
	"os"

	"github.com/danjdewhurst/jot-cli/internal/config"
	"github.com/danjdewhurst/jot-cli/internal/render"
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
	Use:   "jot-cli",
	Short: "A CLI-first notes app",
	Long:  "jot-cli is a fast, context-aware notes tool for the terminal.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var cfgErr error
		cfg, cfgErr = config.Load()
		if cfgErr != nil {
			return fmt.Errorf("loading config: %w", cfgErr)
		}
		if flagDB != "" {
			cfg.DBPath = flagDB
		}

		// Apply config/env JSON default only if --json flag was not explicitly set
		if !cmd.Flags().Changed("json") && cfg.JSON {
			flagJSON = true
		}

		// Set render date format from config
		render.DateFormat = cfg.DateFormat

		// Commands that don't need DB
		if cmd.Name() == "version" || cmd.Name() == "context" || cmd.Name() == "config" || cmd.Name() == "init" {
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
			return tuiCmd.RunE(tuiCmd, args)
		}
		return listCmd.RunE(listCmd, args)
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "Output as JSON")
	rootCmd.PersistentFlags().StringVar(&flagDB, "db", "", "Database path")
	rootCmd.PersistentFlags().BoolVar(&flagVerbose, "verbose", false, "Verbose output")
}
