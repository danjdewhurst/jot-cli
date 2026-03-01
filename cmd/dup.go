package cmd

import (
	"fmt"
	"os"

	"github.com/danjdewhurst/jot-cli/internal/context"
	"github.com/danjdewhurst/jot-cli/internal/model"
	"github.com/danjdewhurst/jot-cli/internal/render"
	"github.com/spf13/cobra"
)

// autoContextKeys are tag keys managed by environment detection.
// These are excluded when copying tags from the original note.
var autoContextKeys = map[string]bool{
	"folder":     true,
	"git_repo":   true,
	"git_branch": true,
}

var dupCmd = &cobra.Command{
	Use:   "dup <id>",
	Short: "Duplicate a note",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		orig, err := resolveNote(args[0])
		if err != nil {
			return err
		}

		// Copy user tags, skip auto-context keys
		tags := context.AutoTags()
		for _, t := range orig.Tags {
			if !autoContextKeys[t.Key] {
				tags = append(tags, model.Tag{Key: t.Key, Value: t.Value})
			}
		}

		dup, err := db.CreateNote(orig.Title, orig.Body, tags)
		if err != nil {
			return fmt.Errorf("duplicating note: %w", err)
		}

		if flagJSON {
			return render.JSON(os.Stdout, dup)
		}

		fmt.Fprintf(os.Stderr, "Duplicated note %s\n", dup.ID[:8])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(dupCmd)
}
