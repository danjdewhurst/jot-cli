package cmd

import (
	"time"

	"github.com/danjdewhurst/jot-cli/internal/context"
	"github.com/danjdewhurst/jot-cli/internal/model"
	"github.com/danjdewhurst/jot-cli/internal/render"
	"github.com/spf13/cobra"
)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Show a chronological log of notes",
	Long:  "Display notes in a compact, git-log style chronological view.",
	RunE:  runLog,
}

func runLog(cmd *cobra.Command, args []string) error {
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

	// Default limit of 20 for log
	limit, _ := cmd.Flags().GetInt("limit")
	if !cmd.Flags().Changed("limit") {
		limit = 20
	}
	filter.Limit = limit

	// --since / --until
	sinceStr, _ := cmd.Flags().GetString("since")
	if sinceStr != "" {
		t, err := parseDate(sinceStr)
		if err != nil {
			return err
		}
		filter.Since = &t
	}

	untilStr, _ := cmd.Flags().GetString("until")
	if untilStr != "" {
		t, err := parseDate(untilStr)
		if err != nil {
			return err
		}
		filter.Until = &t
	}

	// --today shorthand
	if today, _ := cmd.Flags().GetBool("today"); today {
		now := time.Now()
		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		filter.Since = &startOfDay
		filter.Limit = 0 // no limit for today
	}

	reverse, _ := cmd.Flags().GetBool("reverse")
	filter.SortAsc = reverse

	notes, err := db.ListNotes(filter)
	if err != nil {
		return err
	}

	w := cmd.OutOrStdout()

	if flagJSON {
		return render.JSON(w, notes)
	}

	render.NoteLog(w, notes)
	return nil
}

func init() {
	logCmd.Flags().StringSlice("tag", nil, "Filter by tag (key:value)")
	logCmd.Flags().Bool("folder", false, "Filter by current folder")
	logCmd.Flags().Bool("repo", false, "Filter by current git repo")
	logCmd.Flags().Bool("branch", false, "Filter by current git branch")
	logCmd.Flags().Bool("archived", false, "Include archived notes")
	logCmd.Flags().Int("limit", 0, "Maximum number of notes (default: 20)")
	logCmd.Flags().String("since", "", "Show notes created after this date (YYYY-MM-DD or RFC 3339)")
	logCmd.Flags().String("until", "", "Show notes created before this date (YYYY-MM-DD or RFC 3339)")
	logCmd.Flags().Bool("reverse", false, "Show oldest notes first")
	logCmd.Flags().Bool("today", false, "Show only today's notes")
	rootCmd.AddCommand(logCmd)
}
