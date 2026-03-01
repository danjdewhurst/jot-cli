package cmd

import (
	"os"

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

		if flagJSON {
			return render.JSON(os.Stdout, note)
		}

		render.NoteDetail(os.Stdout, note)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
}
