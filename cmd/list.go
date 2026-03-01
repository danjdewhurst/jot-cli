package cmd

import (
	"os"

	"github.com/danjdewhurst/jot-cli/internal/render"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List notes",
	RunE: func(cmd *cobra.Command, args []string) error {
		filter, err := buildNoteFilter(cmd)
		if err != nil {
			return err
		}

		notes, err := db.ListNotes(filter)
		if err != nil {
			return err
		}

		if flagJSON {
			return render.JSON(os.Stdout, notes)
		}

		render.NoteTable(os.Stdout, notes)
		return nil
	},
}

func init() {
	listCmd.Flags().StringSlice("tag", nil, "Filter by tag (key:value)")
	listCmd.Flags().Bool("folder", false, "Filter by current folder")
	listCmd.Flags().Bool("repo", false, "Filter by current git repo")
	listCmd.Flags().Bool("branch", false, "Filter by current git branch")
	listCmd.Flags().Bool("archived", false, "Include archived notes")
	listCmd.Flags().Bool("pinned", false, "Show only pinned notes")
	listCmd.Flags().Int("limit", 0, "Limit number of results")
	rootCmd.AddCommand(listCmd)
}
