package render

import (
	"fmt"
	"io"
	"strings"

	"github.com/danjdewhurst/jot-cli/internal/model"
	"github.com/danjdewhurst/jot-cli/internal/tui/theme"
)

// NoteLog renders notes in a compact, git-log style chronological format.
func NoteLog(w io.Writer, notes []model.Note) {
	if len(notes) == 0 {
		_, _ = fmt.Fprintln(w, "No notes found.")
		return
	}

	for i, n := range notes {
		id := theme.LogHash.Render(shortID(n.ID))
		ts := theme.LogTimestamp.Render(n.CreatedAt.Format("2006-01-02 15:04"))

		title := n.Title
		if title == "" {
			title = TruncateLog(n.Body, 60)
		}
		if title == "" {
			title = "(empty)"
		}
		title = theme.LogTitle.Render(title)

		_, _ = fmt.Fprintf(w, "%s  %s  %s\n", id, ts, title)

		if len(n.Tags) > 0 {
			var parts []string
			for _, t := range n.Tags {
				parts = append(parts, theme.LogTagKey.Render(t.Key+":")+theme.LogTagValue.Render(t.Value))
			}
			indent := strings.Repeat(" ", 28)
			_, _ = fmt.Fprintf(w, "%s%s\n", indent, strings.Join(parts, "  "))
		}

		if i < len(notes)-1 {
			_, _ = fmt.Fprintln(w)
		}
	}
}

