package cmd

import (
	"fmt"
	"os"

	"github.com/danjdewhurst/jot-cli/internal/render"
	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:   "rm <id>",
	Short: "Archive or delete a note",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		note, err := resolveNote(args[0])
		if err != nil {
			return err
		}

		purge, _ := cmd.Flags().GetBool("purge")
		force, _ := cmd.Flags().GetBool("force")

		if purge {
			if !force {
				return fmt.Errorf("use --force to permanently delete note %s", note.ID)
			}
			if err := db.DeleteNote(note.ID); err != nil {
				return err
			}
			if flagJSON {
				return render.JSON(os.Stdout, map[string]string{"deleted": note.ID})
			}
			fmt.Fprintf(os.Stderr, "Deleted note %s\n", note.ID)
		} else {
			if err := db.ArchiveNote(note.ID); err != nil {
				return err
			}
			if flagJSON {
				return render.JSON(os.Stdout, map[string]string{"archived": note.ID})
			}
			fmt.Fprintf(os.Stderr, "Archived note %s\n", note.ID)
		}
		return nil
	},
}

func init() {
	rmCmd.Flags().Bool("purge", false, "Permanently delete instead of archiving")
	rmCmd.Flags().Bool("force", false, "Confirm permanent deletion")
	rootCmd.AddCommand(rmCmd)
}
