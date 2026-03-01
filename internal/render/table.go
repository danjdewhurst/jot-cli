package render

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/danjdewhurst/jot-cli/internal/model"
	"github.com/danjdewhurst/jot-cli/internal/store"
)

func NoteTable(w io.Writer, notes []model.Note) {
	if len(notes) == 0 {
		_, _ = fmt.Fprintln(w, "No notes found.")
		return
	}

	// Header
	_, _ = fmt.Fprintf(w, "%-28s  %-40s  %-12s  %s\n", "ID", "TITLE", "AGE", "TAGS")
	_, _ = fmt.Fprintf(w, "%s\n", strings.Repeat("─", 100))

	for _, n := range notes {
		title := n.Title
		if n.Pinned {
			title = "* " + title
		}
		if len(title) > 40 {
			title = title[:37] + "…"
		}

		var tagStrs []string
		for _, t := range n.Tags {
			tagStrs = append(tagStrs, t.String())
		}

		_, _ = fmt.Fprintf(w, "%-28s  %-40s  %-12s  %s\n",
			shortID(n.ID),
			title,
			FormatTime(n.CreatedAt),
			strings.Join(tagStrs, ", "),
		)
	}
}

func NoteDetail(w io.Writer, n model.Note, backlinks []model.Note) {
	_, _ = fmt.Fprintf(w, "ID:      %s\n", n.ID)
	_, _ = fmt.Fprintf(w, "Title:   %s\n", n.Title)
	_, _ = fmt.Fprintf(w, "Created: %s\n", n.CreatedAt.Format(time.RFC3339))
	_, _ = fmt.Fprintf(w, "Updated: %s\n", n.UpdatedAt.Format(time.RFC3339))

	if n.Pinned {
		_, _ = fmt.Fprintf(w, "Pinned:  yes\n")
	}

	if len(n.Tags) > 0 {
		var tagStrs []string
		for _, t := range n.Tags {
			tagStrs = append(tagStrs, t.String())
		}
		_, _ = fmt.Fprintf(w, "Tags:    %s\n", strings.Join(tagStrs, ", "))
	}

	if n.Body != "" {
		_, _ = fmt.Fprintf(w, "\n%s\n", n.Body)
	}

	if len(backlinks) > 0 {
		_, _ = fmt.Fprintf(w, "\nReferenced by:\n")
		for _, bl := range backlinks {
			title := bl.Title
			if title == "" {
				title = "(untitled)"
			}
			_, _ = fmt.Fprintf(w, "  %s  %s\n", shortID(bl.ID), title)
		}
	}
}

func TagTable(w io.Writer, tags []model.Tag) {
	if len(tags) == 0 {
		_, _ = fmt.Fprintln(w, "No tags found.")
		return
	}

	_, _ = fmt.Fprintf(w, "%-20s  %s\n", "KEY", "VALUE")
	_, _ = fmt.Fprintf(w, "%s\n", strings.Repeat("─", 50))
	for _, t := range tags {
		_, _ = fmt.Fprintf(w, "%-20s  %s\n", t.Key, t.Value)
	}
}

// StatsTable writes a human-readable stats summary.
func StatsTable(w io.Writer, s store.NoteStats) {
	_, _ = fmt.Fprintf(w, "Notes:       %d", s.TotalNotes)
	if s.ArchivedNotes > 0 {
		_, _ = fmt.Fprintf(w, " (%d archived)", s.ArchivedNotes)
	}
	_, _ = fmt.Fprintln(w)

	_, _ = fmt.Fprintf(w, "Pinned:      %d\n", s.PinnedNotes)
	_, _ = fmt.Fprintf(w, "Tags:        %d unique\n", s.UniqueTags)

	if len(s.TopTags) > 0 {
		var parts []string
		for _, tc := range s.TopTags {
			parts = append(parts, fmt.Sprintf("%s:%s (%d)", tc.Key, tc.Value, tc.Count))
		}
		_, _ = fmt.Fprintf(w, "Top tags:    %s\n", strings.Join(parts, ", "))
	}

	_, _ = fmt.Fprintf(w, "This week:   %d notes\n", s.WeeklyCount)
	_, _ = fmt.Fprintf(w, "This month:  %d notes\n", s.MonthlyCount)

	if !s.OldestDate.IsZero() {
		_, _ = fmt.Fprintf(w, "Oldest:      %s\n", s.OldestDate.Format(time.DateOnly))
	}
	if !s.NewestDate.IsZero() {
		_, _ = fmt.Fprintf(w, "Newest:      %s\n", s.NewestDate.Format(time.DateOnly))
	}
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

