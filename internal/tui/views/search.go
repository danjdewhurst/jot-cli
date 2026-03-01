package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/danjdewhurst/jot-cli/internal/model"
	"github.com/danjdewhurst/jot-cli/internal/render"
	"github.com/danjdewhurst/jot-cli/internal/tui/theme"
)

type SearchView struct {
	input   textinput.Model
	results []model.Note
	cursor  int
	width   int
	height  int
	lastQ   string
}

func NewSearchView() SearchView {
	ti := textinput.New()
	ti.Placeholder = "Search notes…"
	ti.TextStyle = lipgloss.NewStyle().Foreground(theme.Text)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(theme.Overlay0)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(theme.Lavender)
	ti.Focus()
	return SearchView{input: ti}
}

func (s *SearchView) Reset() {
	s.input.Reset()
	s.results = nil
	s.cursor = 0
	s.lastQ = ""
	s.input.Focus()
}

func (s *SearchView) SetResults(notes []model.Note) {
	s.results = notes
	if s.cursor >= len(notes) && len(notes) > 0 {
		s.cursor = len(notes) - 1
	}
}

func (s *SearchView) SetSize(w, h int) {
	s.width = w
	s.height = h
	s.input.Width = w - 4
}

func (s *SearchView) Query() string {
	q := strings.TrimSpace(s.input.Value())
	if q == s.lastQ {
		return ""
	}
	s.lastQ = q
	return q
}

func (s *SearchView) SelectedNote() (model.Note, bool) {
	if s.cursor < len(s.results) {
		return s.results[s.cursor], true
	}
	return model.Note{}, false
}

func (s *SearchView) Update(msg tea.Msg) tea.Cmd {
	if kmsg, ok := msg.(tea.KeyMsg); ok {
		switch kmsg.String() {
		case "up", "ctrl+p":
			if s.cursor > 0 {
				s.cursor--
			}
			return nil
		case "down", "ctrl+n":
			if s.cursor < len(s.results)-1 {
				s.cursor++
			}
			return nil
		}
	}

	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	return cmd
}

func (s SearchView) View() string {
	var b strings.Builder

	b.WriteString(theme.SearchPrompt.Render("Search: "))
	b.WriteString(s.input.View())
	b.WriteString("\n\n")

	if len(s.results) == 0 {
		if s.lastQ != "" {
			b.WriteString(theme.SearchDim.Render("No results."))
		}
		return b.String()
	}

	b.WriteString(theme.SearchDim.Render(fmt.Sprintf("%d results", len(s.results))))
	b.WriteString("\n\n")

	for i, n := range s.results {
		title := n.Title
		if title == "" {
			title = render.Truncate(n.Body, 50)
		}

		if i == s.cursor {
			cursor := theme.SearchCursor.Render("▸")
			row := fmt.Sprintf("%s %s", cursor, title)
			b.WriteString(theme.SearchSelected.Width(s.width).Render(row))
		} else {
			_, _ = fmt.Fprintf(&b, "  %s", title)
		}
		b.WriteString("\n")
	}

	return b.String()
}
