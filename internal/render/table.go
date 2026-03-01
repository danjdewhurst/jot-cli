package render

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/danjdewhurst/jot-cli/internal/model"
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
			relativeTime(n.CreatedAt),
			strings.Join(tagStrs, ", "),
		)
	}
}

func NoteDetail(w io.Writer, n model.Note) {
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

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

func relativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		return fmt.Sprintf("%dm ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		return fmt.Sprintf("%dh ago", h)
	case d < 30*24*time.Hour:
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	default:
		months := int(d.Hours() / 24 / 30)
		return fmt.Sprintf("%dmo ago", months)
	}
}
