package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/danjdewhurst/jot-cli/internal/model"
)

var (
	detailTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	detailMetaStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
)

type DetailView struct {
	note   model.Note
	width  int
	height int
	scroll int
}

func NewDetailView() DetailView {
	return DetailView{}
}

func (d *DetailView) SetNote(n model.Note) {
	d.note = n
	d.scroll = 0
}

func (d *DetailView) Note() model.Note {
	return d.note
}

func (d *DetailView) SetSize(w, h int) {
	d.width = w
	d.height = h
}

func (d *DetailView) Update(msg tea.Msg) {
	if kmsg, ok := msg.(tea.KeyMsg); ok {
		switch kmsg.String() {
		case "up", "k":
			if d.scroll > 0 {
				d.scroll--
			}
		case "down", "j":
			d.scroll++
		case "ctrl+u", "pgup":
			d.scroll -= 10
			if d.scroll < 0 {
				d.scroll = 0
			}
		case "ctrl+d", "pgdown":
			d.scroll += 10
		}
	}
}

func (d DetailView) View() string {
	var b strings.Builder

	title := d.note.Title
	if title == "" {
		title = "(untitled)"
	}
	b.WriteString(detailTitleStyle.Render(title))
	b.WriteString("\n")

	meta := fmt.Sprintf("Created: %s  Updated: %s", d.note.CreatedAt.Format("2006-01-02 15:04"), d.note.UpdatedAt.Format("2006-01-02 15:04"))
	b.WriteString(detailMetaStyle.Render(meta))
	b.WriteString("\n")

	if len(d.note.Tags) > 0 {
		var tagParts []string
		for _, t := range d.note.Tags {
			tagParts = append(tagParts, t.String())
		}
		b.WriteString(detailMetaStyle.Render("Tags: " + strings.Join(tagParts, ", ")))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(d.note.Body)

	lines := strings.Split(b.String(), "\n")
	if d.scroll >= len(lines) {
		d.scroll = len(lines) - 1
	}
	if d.scroll < 0 {
		d.scroll = 0
	}

	visible := lines[d.scroll:]
	if len(visible) > d.height {
		visible = visible[:d.height]
	}

	return strings.Join(visible, "\n")
}
