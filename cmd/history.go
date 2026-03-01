package cmd

import (
	"fmt"
	"os"

	"github.com/danjdewhurst/jot-cli/internal/diff"
	"github.com/danjdewhurst/jot-cli/internal/render"
	"github.com/spf13/cobra"
)

var historyCmd = &cobra.Command{
	Use:   "history <id>",
	Short: "Show version history for a note",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		note, err := resolveNote(args[0])
		if err != nil {
			return err
		}

		versionNum, _ := cmd.Flags().GetInt("version")
		showDiff, _ := cmd.Flags().GetBool("diff")

		// Show a specific version
		if versionNum > 0 {
			v, err := db.GetVersion(note.ID, versionNum)
			if err != nil {
				return err
			}

			if flagJSON {
				return render.JSON(os.Stdout, v)
			}

			render.VersionDetail(os.Stdout, v)

			if showDiff {
				// Diff against the previous version or current note
				var oldTitle, oldBody string
				if versionNum > 1 {
					prev, err := db.GetVersion(note.ID, versionNum-1)
					if err != nil {
						return fmt.Errorf("getting previous version: %w", err)
					}
					oldTitle = prev.Title
					oldBody = prev.Body
				}
				// Compare combined title+body
				oldContent := oldTitle + "\n" + oldBody
				newContent := v.Title + "\n" + v.Body
				fmt.Fprintf(os.Stdout, "\n%s", diff.Format(oldContent, newContent))
			}
			return nil
		}

		// List all versions
		versions, err := db.ListVersions(note.ID)
		if err != nil {
			return err
		}

		if flagJSON {
			return render.JSON(os.Stdout, versions)
		}

		// Compute diff summaries between consecutive versions
		summaries := make([]string, len(versions))
		for i, v := range versions {
			// versions are newest first; compare each to its predecessor
			var prevTitle, prevBody string
			if v.Version > 1 {
				// Find the previous version in our list (it's at i+1 since newest first)
				if i+1 < len(versions) {
					prevTitle = versions[i+1].Title
					prevBody = versions[i+1].Body
				}
			}
			oldContent := prevTitle + "\n" + prevBody
			newContent := v.Title + "\n" + v.Body
			summaries[i] = diff.Summary(oldContent, newContent)
		}

		render.HistoryTable(os.Stdout, versions, summaries)
		return nil
	},
}

func init() {
	historyCmd.Flags().Int("version", 0, "show a specific version")
	historyCmd.Flags().Bool("diff", false, "show full diff")
	rootCmd.AddCommand(historyCmd)
}
