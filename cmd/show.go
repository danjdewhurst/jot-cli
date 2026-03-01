package cmd

import (
	"fmt"
	"os"

	"github.com/danjdewhurst/jot-cli/internal/model"
	"github.com/danjdewhurst/jot-cli/internal/render"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a note",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		note, err := resolveNote(args[0])
		if err != nil {
			return err
		}

		backlinks, err := db.ReferencesTo(note.ID)
		if err != nil {
			return fmt.Errorf("querying backlinks: %w", err)
		}

		if flagJSON {
			type showOutput struct {
				Note      model.Note   `json:"note"`
				Backlinks []model.Note `json:"backlinks,omitempty"`
			}
			return render.JSON(os.Stdout, showOutput{Note: note, Backlinks: backlinks})
		}

		render.NoteDetail(os.Stdout, note, backlinks)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
}
