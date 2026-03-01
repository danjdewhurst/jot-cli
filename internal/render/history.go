package render

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/danjdewhurst/jot-cli/internal/model"
)

// HistoryTable writes a table of note versions with timestamps and diff summaries.
func HistoryTable(w io.Writer, versions []model.NoteVersion, diffSummaries []string) {
	if len(versions) == 0 {
		_, _ = fmt.Fprintln(w, "No history found.")
		return
	}

	_, _ = fmt.Fprintf(w, "%-8s  %-20s  %-12s  %s\n", "VERSION", "DATE", "AGE", "CHANGES")
	_, _ = fmt.Fprintf(w, "%s\n", strings.Repeat("─", 60))

	for i, v := range versions {
		summary := ""
		if i < len(diffSummaries) {
			summary = diffSummaries[i]
		}
		_, _ = fmt.Fprintf(w, "%-8d  %-20s  %-12s  %s\n",
			v.Version,
			v.CreatedAt.Format(time.DateTime),
			FormatTime(v.CreatedAt),
			summary,
		)
	}
}

// VersionDetail writes the full content of a specific version.
func VersionDetail(w io.Writer, v model.NoteVersion) {
	_, _ = fmt.Fprintf(w, "Version: %d\n", v.Version)
	_, _ = fmt.Fprintf(w, "Date:    %s\n", v.CreatedAt.Format(time.RFC3339))
	_, _ = fmt.Fprintf(w, "Title:   %s\n", v.Title)
	if v.Body != "" {
		_, _ = fmt.Fprintf(w, "\n%s\n", v.Body)
	}
}
