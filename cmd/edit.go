package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/danjdewhurst/jot-cli/internal/editor"
	"github.com/danjdewhurst/jot-cli/internal/render"
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit a note",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		note, err := resolveNote(args[0])
		if err != nil {
			return err
		}

		title, _ := cmd.Flags().GetString("title")
		body, _ := cmd.Flags().GetString("message")

		if title == "" && body == "" {
			// Open editor with current content
			initial := ""
			if note.Title != "" {
				initial = "# " + note.Title + "\n\n"
			}
			initial += note.Body

			edited, err := editor.Edit(initial, cfg.Editor)
			if err != nil {
				return fmt.Errorf("editor: %w", err)
			}
			edited = strings.TrimSpace(edited)
			if edited == "" {
				return fmt.Errorf("empty note, aborting")
			}

			title, body = extractTitle(edited)
			if title == "" {
				title = note.Title
			}
		} else {
			if title == "" {
				title = note.Title
			}
			if body == "" {
				body = note.Body
			}
		}

		updated, err := db.UpdateNote(note.ID, title, body)
		if err != nil {
			return fmt.Errorf("updating note: %w", err)
		}

		syncNoteRefs(updated.ID, body)

		if flagJSON {
			return render.JSON(os.Stdout, updated)
		}

		fmt.Fprintf(os.Stderr, "Updated note %s\n", updated.ID)
		return nil
	},
}

func init() {
	editCmd.Flags().StringP("title", "t", "", "New title")
	editCmd.Flags().StringP("message", "m", "", "New body")
	rootCmd.AddCommand(editCmd)
}
