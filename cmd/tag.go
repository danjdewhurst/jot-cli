package cmd

import (
	"fmt"
	"os"

	"github.com/danjdewhurst/jot-cli/internal/model"
	"github.com/danjdewhurst/jot-cli/internal/render"
	"github.com/spf13/cobra"
)

var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Manage tags",
}

var tagListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tags",
	RunE: func(cmd *cobra.Command, args []string) error {
		key, _ := cmd.Flags().GetString("key")

		tags, err := db.ListTags(key)
		if err != nil {
			return err
		}

		if flagJSON {
			return render.JSON(os.Stdout, tags)
		}

		render.TagTable(os.Stdout, tags)
		return nil
	},
}

var tagAddCmd = &cobra.Command{
	Use:   "add [<id>...] <key:value>",
	Short: "Add a tag to one or more notes",
	Long:  "Add a tag to notes. Provide note IDs followed by the tag, or use --tag to filter target notes.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		// Last arg is always the tag to add
		tagStr := args[len(args)-1]
		tag, err := model.ParseTag(tagStr)
		if err != nil {
			return fmt.Errorf("invalid tag %q: %w", tagStr, err)
		}

		noteArgs := args[:len(args)-1]

		// If no note IDs provided, use --tag filter from the parent command scope
		var notes []model.Note
		if len(noteArgs) > 0 {
			for _, idOrPrefix := range noteArgs {
				n, err := resolveNote(idOrPrefix)
				if err != nil {
					return err
				}
				notes = append(notes, n)
			}
		} else {
			// Use filter flags from tagAddCmd
			filter, err := buildNoteFilter(cmd)
			if err != nil {
				return err
			}
			if len(filter.Tags) == 0 {
				return fmt.Errorf("provide note IDs or use --tag to filter target notes")
			}
			notes, err = db.ListNotes(filter)
			if err != nil {
				return fmt.Errorf("listing notes: %w", err)
			}
			if len(notes) == 0 {
				return fmt.Errorf("no notes match the given filter")
			}
		}

		if dryRun {
			for _, n := range notes {
				title := n.Title
				if title == "" {
					title = "(untitled)"
				}
				fmt.Fprintf(os.Stderr, "  %s  %s\n", n.ID[:8], title)
			}
			fmt.Fprintf(os.Stderr, "Would add tag %s to %d notes (dry run)\n", tag, len(notes))
			return nil
		}

		// Single note: use existing single-note method for backwards compatibility output
		if len(notes) == 1 {
			if err := db.AddTag(notes[0].ID, tag); err != nil {
				return err
			}
			if flagJSON {
				updated, err := db.GetNote(notes[0].ID)
				if err != nil {
					return fmt.Errorf("retrieving updated note: %w", err)
				}
				return render.JSON(os.Stdout, updated)
			}
			fmt.Fprintf(os.Stderr, "Added tag %s to %s\n", tag, notes[0].ID)
			return nil
		}

		// Bulk
		if !confirmBulk(cmd, fmt.Sprintf("Add tag %s to", tag), len(notes)) {
			fmt.Fprintln(os.Stderr, "Cancelled")
			return nil
		}

		ids := make([]string, len(notes))
		for i, n := range notes {
			ids[i] = n.ID
		}

		count, err := db.AddTagToNotes(ids, tag)
		if err != nil {
			return fmt.Errorf("adding tag: %w", err)
		}
		if flagJSON {
			return render.JSON(os.Stdout, map[string]any{"tagged": count, "tag": tag.String()})
		}
		fmt.Fprintf(os.Stderr, "Added tag %s to %d notes\n", tag, count)
		return nil
	},
}

var tagRmCmd = &cobra.Command{
	Use:   "rm <id> <key:value>",
	Short: "Remove a tag from a note",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		note, err := resolveNote(args[0])
		if err != nil {
			return err
		}

		tag, err := model.ParseTag(args[1])
		if err != nil {
			return err
		}

		if err := db.RemoveTag(note.ID, tag); err != nil {
			return err
		}

		if flagJSON {
			updated, err := db.GetNote(note.ID)
			if err != nil {
				return fmt.Errorf("retrieving updated note: %w", err)
			}
			return render.JSON(os.Stdout, updated)
		}

		fmt.Fprintf(os.Stderr, "Removed tag %s from %s\n", tag, note.ID)
		return nil
	},
}

func init() {
	tagListCmd.Flags().String("key", "", "Filter tags by key")

	tagAddCmd.Flags().Bool("dry-run", false, "Show what would be affected without executing")
	tagAddCmd.Flags().Bool("force", false, "Skip confirmation for bulk operations")
	tagAddCmd.Flags().StringSlice("tag", nil, "Filter target notes by tag (key:value)")
	tagAddCmd.Flags().Bool("folder", false, "Filter by current folder")
	tagAddCmd.Flags().Bool("repo", false, "Filter by current git repo")
	tagAddCmd.Flags().Bool("branch", false, "Filter by current git branch")
	tagAddCmd.Flags().Bool("archived", false, "Include archived notes")
	tagAddCmd.Flags().Int("limit", 0, "Limit number of results")

	tagCmd.AddCommand(tagListCmd, tagAddCmd, tagRmCmd)
	rootCmd.AddCommand(tagCmd)
}
