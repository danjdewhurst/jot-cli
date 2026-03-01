package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/danjdewhurst/jot-cli/internal/linking"
	"github.com/danjdewhurst/jot-cli/internal/model"
	"github.com/danjdewhurst/jot-cli/internal/tui/theme"
)

type DetailView struct {
	note      model.Note
	backlinks []model.Note
	width     int
	height    int
	scroll    int
}

func NewDetailView() DetailView {
	return DetailView{}
}

func (d *DetailView) SetNote(n model.Note) {
	d.note = n
	d.backlinks = nil
	d.scroll = 0
}

func (d *DetailView) SetBacklinks(notes []model.Note) {
	d.backlinks = notes
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

	// Clamp scroll to valid range. Compute total lines from rendered content
	// so the clamping happens on the real struct, not a value-receiver copy.
	totalLines := d.contentLineCount()
	maxScroll := totalLines - d.height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if d.scroll > maxScroll {
		d.scroll = maxScroll
	}
	if d.scroll < 0 {
		d.scroll = 0
	}
}

// renderContent builds the full detail content string (shared by View and contentLineCount).
func (d DetailView) renderContent() string {
	var b strings.Builder

	title := d.note.Title
	if title == "" {
		title = "(untitled)"
	}
	if d.note.Pinned {
		title = theme.DetailPin.Render("♦") + " " + title
	}
	b.WriteString(theme.DetailTitle.Render(title))
	b.WriteString("\n")

	meta := fmt.Sprintf("Created: %s  Updated: %s", d.note.CreatedAt.Format("2006-01-02 15:04"), d.note.UpdatedAt.Format("2006-01-02 15:04"))
	b.WriteString(theme.DetailMeta.Render(meta))
	b.WriteString("\n")

	if len(d.note.Tags) > 0 {
		var pills []string
		for _, t := range d.note.Tags {
			pills = append(pills, theme.TagPill.Render(t.String()))
		}
		b.WriteString(" " + strings.Join(pills, " "))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(theme.DetailBody.Render(highlightRefs(d.note.Body)))

	if len(d.backlinks) > 0 {
		b.WriteString("\n")
		b.WriteString(theme.DetailBacklinkHeader.Render("Referenced by:"))
		b.WriteString("\n")
		for _, bl := range d.backlinks {
			title := bl.Title
			if title == "" {
				title = "(untitled)"
			}
			id := bl.ID
			if len(id) > 8 {
				id = id[:8]
			}
			b.WriteString(theme.DetailBacklink.Render(
				theme.DetailRef.Render(id) + "  " + title,
			))
			b.WriteString("\n")
		}
	}

	return b.String()
}

// highlightRefs replaces @<prefix> references in body text with styled versions.
func highlightRefs(body string) string {
	refs := linking.ExtractRefs(body)
	if len(refs) == 0 {
		return body
	}
	result := body
	for _, ref := range refs {
		styled := theme.DetailRef.Render("@" + ref)
		result = strings.ReplaceAll(result, "@"+ref, styled)
	}
	return result
}

// contentLineCount returns the number of lines in the rendered content.
func (d DetailView) contentLineCount() int {
	return len(strings.Split(d.renderContent(), "\n"))
}

func (d DetailView) View() string {
	lines := strings.Split(d.renderContent(), "\n")

	scroll := d.scroll
	if scroll >= len(lines) {
		scroll = len(lines) - 1
	}
	if scroll < 0 {
		scroll = 0
	}

	visible := lines[scroll:]
	if len(visible) > d.height {
		visible = visible[:d.height]
	}

	return strings.Join(visible, "\n")
}
