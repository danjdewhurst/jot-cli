package cmd

import (
	"os"

	"github.com/danjdewhurst/jot-cli/internal/context"
	"github.com/danjdewhurst/jot-cli/internal/model"
	"github.com/danjdewhurst/jot-cli/internal/render"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List notes",
	RunE: func(cmd *cobra.Command, args []string) error {
		filter := model.NoteFilter{}

		tagStrs, _ := cmd.Flags().GetStringSlice("tag")
		for _, ts := range tagStrs {
			t, err := model.ParseTag(ts)
			if err != nil {
				return err
			}
			filter.Tags = append(filter.Tags, t)
		}

		if f, _ := cmd.Flags().GetBool("folder"); f {
			if folder, err := context.DetectFolder(); err == nil && folder != "" {
				filter.Tags = append(filter.Tags, model.Tag{Key: "folder", Value: folder})
			}
		}
		if r, _ := cmd.Flags().GetBool("repo"); r {
			if repo, err := context.DetectRepo(); err == nil && repo != "" {
				filter.Tags = append(filter.Tags, model.Tag{Key: "git_repo", Value: repo})
			}
		}
		if b, _ := cmd.Flags().GetBool("branch"); b {
			if branch, err := context.DetectBranch(); err == nil && branch != "" {
				filter.Tags = append(filter.Tags, model.Tag{Key: "git_branch", Value: branch})
			}
		}

		archived, _ := cmd.Flags().GetBool("archived")
		filter.Archived = archived

		if pinned, _ := cmd.Flags().GetBool("pinned"); pinned {
			filter.PinnedOnly = true
		}

		limit, _ := cmd.Flags().GetInt("limit")
		filter.Limit = limit

		notes, err := db.ListNotes(filter)
		if err != nil {
			return err
		}

		if flagJSON {
			return render.JSON(os.Stdout, notes)
		}

		render.NoteTable(os.Stdout, notes)
		return nil
	},
}

func init() {
	listCmd.Flags().StringSlice("tag", nil, "Filter by tag (key:value)")
	listCmd.Flags().Bool("folder", false, "Filter by current folder")
	listCmd.Flags().Bool("repo", false, "Filter by current git repo")
	listCmd.Flags().Bool("branch", false, "Filter by current git branch")
	listCmd.Flags().Bool("archived", false, "Include archived notes")
	listCmd.Flags().Bool("pinned", false, "Show only pinned notes")
	listCmd.Flags().Int("limit", 0, "Limit number of results")
	rootCmd.AddCommand(listCmd)
}
