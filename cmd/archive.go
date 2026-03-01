package cmd

import (
	"fmt"
	"os"

	"github.com/danjdewhurst/jot-cli/internal/render"
	"github.com/spf13/cobra"
)

var archiveCmd = &cobra.Command{
	Use:   "archive [<id>...]",
	Short: "Archive notes",
	Long:  "Archive one or more notes. Use --tag to filter by tag.",
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		notes, err := collectNotes(cmd, args)
		if err != nil {
			return err
		}

		ids := make([]string, len(notes))
		for i, n := range notes {
			ids[i] = n.ID
		}

		if dryRun {
			for _, n := range notes {
				title := n.Title
				if title == "" {
					title = "(untitled)"
				}
				fmt.Fprintf(os.Stderr, "  %s  %s\n", n.ID[:8], title)
			}
			fmt.Fprintf(os.Stderr, "Would archive %d notes (dry run)\n", len(notes))
			return nil
		}

		if len(notes) > 1 && !confirmBulk(cmd, "Archive", len(notes)) {
			fmt.Fprintln(os.Stderr, "Cancelled")
			return nil
		}

		count, err := db.ArchiveNotes(ids)
		if err != nil {
			return fmt.Errorf("archiving notes: %w", err)
		}
		if flagJSON {
			return render.JSON(os.Stdout, map[string]any{"archived": count})
		}
		fmt.Fprintf(os.Stderr, "Archived %d notes\n", count)
		return nil
	},
}

func init() {
	addBulkFlags(archiveCmd)
	rootCmd.AddCommand(archiveCmd)
}
