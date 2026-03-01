package cmd

import (
	"fmt"
	"os"

	"github.com/danjdewhurst/jot-cli/internal/render"
	"github.com/spf13/cobra"
)

var pinCmd = &cobra.Command{
	Use:   "pin [<id>...]",
	Short: "Pin notes",
	Long:  "Pin one or more notes so they appear at the top of lists. With a single ID, toggles pin state.",
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		// Single note: toggle behaviour (backwards compatible)
		if len(args) == 1 && !cmd.Flags().Changed("tag") {
			note, err := resolveNote(args[0])
			if err != nil {
				return err
			}
			if dryRun {
				action := "Pin"
				if note.Pinned {
					action = "Unpin"
				}
				fmt.Fprintf(os.Stderr, "Would %s note %s (dry run)\n", action, note.ID[:8])
				return nil
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
		}

		// Bulk pin
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
			fmt.Fprintf(os.Stderr, "Would pin %d notes (dry run)\n", len(notes))
			return nil
		}

		if len(notes) > 1 && !confirmBulk(cmd, "Pin", len(notes)) {
			fmt.Fprintln(os.Stderr, "Cancelled")
			return nil
		}

		count, err := db.PinNotes(ids)
		if err != nil {
			return fmt.Errorf("pinning notes: %w", err)
		}
		if flagJSON {
			return render.JSON(os.Stdout, map[string]any{"pinned": count})
		}
		fmt.Fprintf(os.Stderr, "Pinned %d notes\n", count)
		return nil
	},
}

var unpinCmd = &cobra.Command{
	Use:   "unpin [<id>...]",
	Short: "Unpin notes",
	Long:  "Unpin one or more notes.",
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
			fmt.Fprintf(os.Stderr, "Would unpin %d notes (dry run)\n", len(notes))
			return nil
		}

		if len(notes) > 1 && !confirmBulk(cmd, "Unpin", len(notes)) {
			fmt.Fprintln(os.Stderr, "Cancelled")
			return nil
		}

		count, err := db.UnpinNotes(ids)
		if err != nil {
			return fmt.Errorf("unpinning notes: %w", err)
		}
		if flagJSON {
			return render.JSON(os.Stdout, map[string]any{"unpinned": count})
		}
		fmt.Fprintf(os.Stderr, "Unpinned %d notes\n", count)
		return nil
	},
}

func addBulkFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("dry-run", false, "Show what would be affected without executing")
	cmd.Flags().Bool("force", false, "Skip confirmation for bulk operations")
	cmd.Flags().StringSlice("tag", nil, "Filter by tag (key:value)")
	cmd.Flags().Bool("folder", false, "Filter by current folder")
	cmd.Flags().Bool("repo", false, "Filter by current git repo")
	cmd.Flags().Bool("branch", false, "Filter by current git branch")
	cmd.Flags().Bool("archived", false, "Include archived notes")
	cmd.Flags().Int("limit", 0, "Limit number of results")
}

func init() {
	addBulkFlags(pinCmd)
	addBulkFlags(unpinCmd)
	rootCmd.AddCommand(pinCmd)
	rootCmd.AddCommand(unpinCmd)
}
