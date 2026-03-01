package cmd

import (
	"fmt"
	"os"

	"github.com/danjdewhurst/jot-cli/internal/render"
	"github.com/spf13/cobra"
)

var pinCmd = &cobra.Command{
	Use:   "pin <id>",
	Short: "Toggle pin on a note",
	Long:  "Pin a note so it appears at the top of lists. Run again to unpin.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		note, err := resolveNote(args[0])
		if err != nil {
			return err
		}
		pinned, err := db.TogglePin(note.ID)
		if err != nil {
			return err
		}
		if flagJSON {
			return render.JSON(os.Stdout, map[string]any{
				"id":     note.ID,
				"pinned": pinned,
			})
		}
		if pinned {
			fmt.Fprintf(os.Stderr, "Pinned note %s\n", note.ID[:8])
		} else {
			fmt.Fprintf(os.Stderr, "Unpinned note %s\n", note.ID[:8])
		}
		return nil
	},
}

var unpinCmd = &cobra.Command{
	Use:   "unpin <id>",
	Short: "Unpin a note",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		note, err := resolveNote(args[0])
		if err != nil {
			return err
		}
		if err := db.UnpinNote(note.ID); err != nil {
			return err
		}
		if flagJSON {
			return render.JSON(os.Stdout, map[string]any{
				"id":     note.ID,
				"pinned": false,
			})
		}
		fmt.Fprintf(os.Stderr, "Unpinned note %s\n", note.ID[:8])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pinCmd)
	rootCmd.AddCommand(unpinCmd)
}
