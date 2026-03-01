package cmd

import (
	"os"
	"strings"

	"github.com/danjdewhurst/jot-cli/internal/model"
	"github.com/danjdewhurst/jot-cli/internal/render"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Full-text search notes",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.Join(args, " ")

		tags, err := parseTags(cmd)
		if err != nil {
			return err
		}

		results, err := db.Search(query, tags)
		if err != nil {
			return err
		}

		if flagJSON {
			return render.JSON(os.Stdout, results)
		}

		// Extract just the notes for table display
		var notes []model.Note
		for _, r := range results {
			notes = append(notes, r.Note)
		}
		render.NoteTable(os.Stdout, notes)
		return nil
	},
}

func init() {
	searchCmd.Flags().StringSlice("tag", nil, "Filter by tag (key:value)")
	rootCmd.AddCommand(searchCmd)
}
