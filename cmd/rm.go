package cmd

import (
	"fmt"
	"os"

	"github.com/danjdewhurst/jot-cli/internal/render"
	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:   "rm [<id>...]",
	Short: "Archive or delete notes",
	Long:  "Archive one or more notes. Use --tag to filter by tag. Use --purge --force to permanently delete.",
	RunE: func(cmd *cobra.Command, args []string) error {
		purge, _ := cmd.Flags().GetBool("purge")
		force, _ := cmd.Flags().GetBool("force")
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
			action := "Archive"
			if purge {
				action = "Delete"
			}
			fmt.Fprintf(os.Stderr, "Would %s %d notes (dry run)\n", action, len(notes))
			return nil
		}

		if purge {
			if len(notes) > 1 && !force {
				return fmt.Errorf("use --force to permanently delete %d notes", len(notes))
			}
			if len(notes) == 1 && !force {
				return fmt.Errorf("use --force to permanently delete note %s", notes[0].ID)
			}
			count, err := db.DeleteNotes(ids)
			if err != nil {
				return fmt.Errorf("deleting note: %w", err)
			}
			if flagJSON {
				return render.JSON(os.Stdout, map[string]any{"deleted": count})
			}
			fmt.Fprintf(os.Stderr, "Deleted %d notes\n", count)
		} else {
			if len(notes) > 1 && !confirmBulk(cmd, "Archive", len(notes)) {
				fmt.Fprintln(os.Stderr, "Cancelled")
				return nil
			}
			count, err := db.ArchiveNotes(ids)
			if err != nil {
				return fmt.Errorf("archiving note: %w", err)
			}
			if flagJSON {
				return render.JSON(os.Stdout, map[string]any{"archived": count})
			}
			fmt.Fprintf(os.Stderr, "Archived %d notes\n", count)
		}
		return nil
	},
}

func init() {
	rmCmd.Flags().Bool("purge", false, "Permanently delete instead of archiving")
	rmCmd.Flags().Bool("force", false, "Confirm destructive bulk operations")
	rmCmd.Flags().Bool("dry-run", false, "Show what would be affected without executing")
	rmCmd.Flags().StringSlice("tag", nil, "Filter by tag (key:value)")
	rmCmd.Flags().Bool("folder", false, "Filter by current folder")
	rmCmd.Flags().Bool("repo", false, "Filter by current git repo")
	rmCmd.Flags().Bool("branch", false, "Filter by current git branch")
	rmCmd.Flags().Bool("archived", false, "Include archived notes")
	rmCmd.Flags().Int("limit", 0, "Limit number of results")
	rootCmd.AddCommand(rmCmd)
}
