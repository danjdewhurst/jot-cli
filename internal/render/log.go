package render

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/danjdewhurst/jot-cli/internal/model"
)

var (
	logHashStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("214")) // amber
	logTimestampStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // dim grey
	logTitleStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))  // bright white
	logTagKeyStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("243")) // mid grey
	logTagValueStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("110")) // soft blue
)

// NoteLog renders notes in a compact, git-log style chronological format.
func NoteLog(w io.Writer, notes []model.Note) {
	if len(notes) == 0 {
		fmt.Fprintln(w, "No notes found.")
		return
	}

	for i, n := range notes {
		id := logHashStyle.Render(shortID(n.ID))
		ts := logTimestampStyle.Render(n.CreatedAt.Format("2006-01-02 15:04"))

		title := n.Title
		if title == "" {
			title = truncateLog(n.Body, 60)
		}
		if title == "" {
			title = "(empty)"
		}
		title = logTitleStyle.Render(title)

		fmt.Fprintf(w, "%s  %s  %s\n", id, ts, title)

		if len(n.Tags) > 0 {
			var parts []string
			for _, t := range n.Tags {
				parts = append(parts, logTagKeyStyle.Render(t.Key+":")+logTagValueStyle.Render(t.Value))
			}
			indent := strings.Repeat(" ", 28)
			fmt.Fprintf(w, "%s%s\n", indent, strings.Join(parts, "  "))
		}

		if i < len(notes)-1 {
			fmt.Fprintln(w)
		}
	}
}

func truncateLog(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	if len(s) > max {
		return s[:max-1] + "..."
	}
	return s
}
