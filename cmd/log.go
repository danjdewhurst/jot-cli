package cmd

import (
	"fmt"
	"time"

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
	filter, err := buildNoteFilter(cmd)
	if err != nil {
		return err
	}

	// Default limit of 20 for log
	if !cmd.Flags().Changed("limit") {
		filter.Limit = 20
	}

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

	// --today shorthand — conflicts with --since/--until
	if today, _ := cmd.Flags().GetBool("today"); today {
		if cmd.Flags().Changed("since") || cmd.Flags().Changed("until") {
			return fmt.Errorf("--today cannot be combined with --since or --until")
		}
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
