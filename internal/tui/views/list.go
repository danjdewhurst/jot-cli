package views

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/danjdewhurst/jot-cli/internal/model"
)

var (
	listSelectedStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("236")).
				Foreground(lipgloss.Color("15"))
	listTagStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))
	listDimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
	listTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12")).
			Padding(0, 0, 1, 0)
)

type ListView struct {
	notes         []model.Note
	cursor        int
	width         int
	height        int
	offset        int
	contextFilter bool
}

func NewListView() ListView {
	return ListView{}
}

func (l *ListView) SetNotes(notes []model.Note) {
	l.notes = notes
	if l.cursor >= len(notes) && len(notes) > 0 {
		l.cursor = len(notes) - 1
	}
}

func (l *ListView) SetContextFilter(enabled bool) {
	l.contextFilter = enabled
}

func (l *ListView) SetSize(w, h int) {
	l.width = w
	l.height = h
}

func (l *ListView) SelectedNote() (model.Note, bool) {
	if l.cursor < len(l.notes) {
		return l.notes[l.cursor], true
	}
	return model.Note{}, false
}

func (l *ListView) Update(msg tea.Msg) {
	if kmsg, ok := msg.(tea.KeyMsg); ok {
		switch kmsg.String() {
		case "up", "k":
			if l.cursor > 0 {
				l.cursor--
			}
		case "down", "j":
			if l.cursor < len(l.notes)-1 {
				l.cursor++
			}
		case "ctrl+u", "pgup":
			l.cursor -= 10
			if l.cursor < 0 {
				l.cursor = 0
			}
		case "ctrl+d", "pgdown":
			l.cursor += 10
			if l.cursor >= len(l.notes) {
				l.cursor = len(l.notes) - 1
			}
		case "home", "g":
			l.cursor = 0
		case "end", "G":
			l.cursor = len(l.notes) - 1
		}
	}

	// Keep cursor in view
	visibleLines := l.height - 3
	if visibleLines < 1 {
		visibleLines = 1
	}
	if l.cursor < l.offset {
		l.offset = l.cursor
	}
	if l.cursor >= l.offset+visibleLines {
		l.offset = l.cursor - visibleLines + 1
	}
}

func (l ListView) View() string {
	if len(l.notes) == 0 {
		return listTitleStyle.Render("jot") + "\n\nNo notes yet. Press n to create one."
	}

	var b strings.Builder
	title := fmt.Sprintf("jot — %d notes", len(l.notes))
	if l.contextFilter {
		title += " (this project)"
	}
	b.WriteString(listTitleStyle.Render(title))
	b.WriteString("\n")

	visibleLines := l.height - 3
	if visibleLines < 1 {
		visibleLines = 1
	}

	end := l.offset + visibleLines
	if end > len(l.notes) {
		end = len(l.notes)
	}

	for i := l.offset; i < end; i++ {
		n := l.notes[i]
		title := n.Title
		if title == "" {
			title = truncate(n.Body, 50)
		}
		if title == "" {
			title = "(empty)"
		}

		age := relativeTime(n.CreatedAt)
		var tagParts []string
		for _, t := range n.Tags {
			tagParts = append(tagParts, t.String())
		}
		tags := strings.Join(tagParts, " ")

		line := fmt.Sprintf("  %-50s  %s  %s", truncate(title, 50), listDimStyle.Render(age), listTagStyle.Render(tags))

		if i == l.cursor {
			line = listSelectedStyle.Width(l.width).Render(fmt.Sprintf("▸ %-50s  %s  %s", truncate(title, 50), age, tags))
		}

		b.WriteString(line)
		if i < end-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func truncate(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > max {
		return s[:max-1] + "…"
	}
	return s
}

func relativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	default:
		return fmt.Sprintf("%dmo", int(d.Hours()/24/30))
	}
}
