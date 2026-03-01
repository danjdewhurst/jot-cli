package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/danjdewhurst/jot-cli/internal/model"
	"github.com/danjdewhurst/jot-cli/internal/render"
	"github.com/danjdewhurst/jot-cli/internal/tui/theme"
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
		if len(l.notes) == 0 {
			return
		}
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
		return theme.ListTitle.Render("jot") + "\n\nNo notes yet. Press n to create one."
	}

	var b strings.Builder
	title := fmt.Sprintf("jot — %d notes", len(l.notes))
	if l.contextFilter {
		title += " (this project)"
	}
	b.WriteString(theme.ListTitle.Render(title))
	b.WriteString("\n")

	visibleLines := l.height - 3
	if visibleLines < 1 {
		visibleLines = 1
	}

	end := l.offset + visibleLines
	if end > len(l.notes) {
		end = len(l.notes)
	}

	// Dynamic title width: use ~60% of terminal width, minimum 20
	titleWidth := l.width*60/100 - 6
	if titleWidth < 20 {
		titleWidth = 20
	}

	for i := l.offset; i < end; i++ {
		n := l.notes[i]
		title := n.Title
		if title == "" {
			title = render.Truncate(n.Body, titleWidth)
		}
		if title == "" {
			title = "(empty)"
		}

		pin := "  "
		if n.Pinned {
			pin = theme.ListPin.Render("♦ ")
		}

		age := render.RelativeTimeShort(n.CreatedAt)
		var tagParts []string
		for _, t := range n.Tags {
			tagParts = append(tagParts, t.String())
		}
		tags := strings.Join(tagParts, " ")

		titleFmt := fmt.Sprintf("%%-%ds", titleWidth)

		if i == l.cursor {
			cursor := theme.ListCursor.Render("▸")
			row := fmt.Sprintf("%s %s"+titleFmt+"  %s  %s", cursor, pin, render.Truncate(title, titleWidth), age, tags)
			line := theme.ListSelected.Width(l.width).Render(row)
			b.WriteString(line)
		} else {
			line := fmt.Sprintf("  %s"+titleFmt+"  %s  %s", pin, render.Truncate(title, titleWidth), theme.ListDim.Render(age), theme.ListTag.Render(tags))
			b.WriteString(line)
		}

		if i < end-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

