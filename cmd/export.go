package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/danjdewhurst/jot-cli/internal/model"
	"github.com/danjdewhurst/jot-cli/internal/render"
	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export notes to a file",
	RunE:  runExport,
}

func init() {
	exportCmd.Flags().StringP("output", "o", "", "Output file path (default: stdout)")
	exportCmd.Flags().StringP("format", "f", "json", "Export format: json, md")
	exportCmd.Flags().StringSlice("tag", nil, "Filter by tag (key:value)")
	exportCmd.Flags().Bool("archived", false, "Include archived notes")
	exportCmd.Flags().StringP("search", "s", "", "Filter by search query")
	exportCmd.Flags().String("since", "", "Only notes created after this date (RFC 3339 or YYYY-MM-DD)")
	exportCmd.Flags().String("until", "", "Only notes created before this date (RFC 3339 or YYYY-MM-DD)")
	rootCmd.AddCommand(exportCmd)
}

func runExport(cmd *cobra.Command, args []string) error {
	format, _ := cmd.Flags().GetString("format")
	if format != "json" && format != "md" {
		return fmt.Errorf("unsupported format %q (use json or md)", format)
	}

	tagStrs, _ := cmd.Flags().GetStringSlice("tag")
	archived, _ := cmd.Flags().GetBool("archived")
	search, _ := cmd.Flags().GetString("search")
	sinceStr, _ := cmd.Flags().GetString("since")
	untilStr, _ := cmd.Flags().GetString("until")
	output, _ := cmd.Flags().GetString("output")

	// Parse tags
	var tags []model.Tag
	for _, s := range tagStrs {
		t, err := model.ParseTag(s)
		if err != nil {
			return err
		}
		tags = append(tags, t)
	}

	// Parse date filters
	since, err := parseDate(sinceStr)
	if err != nil {
		return fmt.Errorf("invalid --since value: %w", err)
	}
	until, err := parseDate(untilStr)
	if err != nil {
		return fmt.Errorf("invalid --until value: %w", err)
	}

	// Fetch notes
	var notes []model.Note
	if search != "" {
		results, err := db.Search(search, tags)
		if err != nil {
			return fmt.Errorf("searching notes: %w", err)
		}
		for _, r := range results {
			notes = append(notes, r.Note)
		}
	} else {
		filter := model.NoteFilter{
			Tags:     tags,
			Archived: archived,
		}
		notes, err = db.ListNotes(filter)
		if err != nil {
			return fmt.Errorf("listing notes: %w", err)
		}
	}

	// Apply date-range filtering in-memory
	notes = filterByDateRange(notes, since, until)

	// Determine output writer
	w := os.Stdout
	if output != "" {
		f, err := os.Create(output)
		if err != nil {
			return fmt.Errorf("creating output file: %w", err)
		}
		defer f.Close() //nolint:errcheck
		w = f
	}

	// Render
	switch format {
	case "md":
		if err := render.Markdown(w, notes); err != nil {
			return fmt.Errorf("rendering markdown: %w", err)
		}
	case "json":
		envelope := model.ExportEnvelope{
			Version:    model.ExportVersion,
			ExportedAt: time.Now().UTC(),
			Count:      len(notes),
			Notes:      notes,
		}
		if err := render.JSON(w, envelope); err != nil {
			return fmt.Errorf("rendering JSON: %w", err)
		}
	}

	dest := "stdout"
	if output != "" {
		dest = output
	}
	fmt.Fprintf(os.Stderr, "Exported %d notes to %s\n", len(notes), dest)

	return nil
}

// parseDate parses a date string in RFC 3339 or YYYY-MM-DD format.
// Returns zero time for empty input.
func parseDate(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("cannot parse %q as RFC 3339 or YYYY-MM-DD", s)
}

func filterByDateRange(notes []model.Note, since, until time.Time) []model.Note {
	if since.IsZero() && until.IsZero() {
		return notes
	}
	var filtered []model.Note
	for _, n := range notes {
		if !since.IsZero() && n.CreatedAt.Before(since) {
			continue
		}
		if !until.IsZero() && n.CreatedAt.After(until) {
			continue
		}
		filtered = append(filtered, n)
	}
	return filtered
}
