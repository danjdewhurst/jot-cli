package cmd

import (
	"fmt"
	"os"

	"github.com/danjdewhurst/jot-cli/internal/render"
	"github.com/spf13/cobra"
)

var revertCmd = &cobra.Command{
	Use:   "revert <id>",
	Short: "Revert a note to a previous version",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		note, err := resolveNote(args[0])
		if err != nil {
			return err
		}

		versionNum, _ := cmd.Flags().GetInt("version")
		if versionNum < 1 {
			return fmt.Errorf("--version is required and must be >= 1")
		}

		v, err := db.GetVersion(note.ID, versionNum)
		if err != nil {
			return err
		}

		// UpdateNote creates a new version snapshot before overwriting
		updated, err := db.UpdateNote(note.ID, v.Title, v.Body)
		if err != nil {
			return fmt.Errorf("reverting note: %w", err)
		}

		if flagJSON {
			return render.JSON(os.Stdout, updated)
		}

		fmt.Fprintf(os.Stderr, "Reverted to version %d.\n", versionNum)
		return nil
	},
}

func init() {
	revertCmd.Flags().Int("version", 0, "version number to revert to")
	_ = revertCmd.MarkFlagRequired("version")
	rootCmd.AddCommand(revertCmd)
}
