package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/danjdewhurst/jot-cli/internal/model"
	"github.com/danjdewhurst/jot-cli/internal/render"
	"github.com/danjdewhurst/jot-cli/internal/tui/theme"
)

// SearchTickMsg is fired when the debounce timer expires.
// App.go compares Query against the current input to decide whether to search.
type SearchTickMsg struct {
	Query string
}

const searchDebounce = 150 * time.Millisecond

type ListView struct {
	notes         []model.Note
	cursor        int
	width         int
	height        int
	offset        int
	contextFilter bool
	selected      map[string]bool

	// Search mode
	searching bool
	input     textinput.Model
	allNotes  []model.Note // snapshot of notes before search, restored on exit
}

func NewListView() ListView {
	ti := textinput.New()
	ti.Placeholder = "Filter notes…"
	ti.TextStyle = lipgloss.NewStyle().Foreground(theme.Text)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(theme.Overlay0)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(theme.Lavender)
	return ListView{
		input:    ti,
		selected: make(map[string]bool),
	}
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
	l.input.Width = w - 12
}

func (l *ListView) SelectedNote() (model.Note, bool) {
	if l.cursor < len(l.notes) {
		return l.notes[l.cursor], true
	}
	return model.Note{}, false
}

// ── Search ──────────────────────────────────────────────────────────────

func (l *ListView) EnterSearch() {
	l.searching = true
	l.allNotes = make([]model.Note, len(l.notes))
	copy(l.allNotes, l.notes)
	l.input.Reset()
	l.input.Focus()
}

func (l *ListView) ExitSearch() {
	l.searching = false
	l.input.Blur()
	l.input.Reset()
	if l.allNotes != nil {
		l.notes = l.allNotes
		l.allNotes = nil
	}
	l.cursor = 0
	l.offset = 0
}

func (l *ListView) IsSearching() bool {
	return l.searching
}

func (l *ListView) SearchQuery() string {
	return strings.TrimSpace(l.input.Value())
}

func (l *ListView) SetSearchResults(notes []model.Note) {
	l.notes = notes
	l.cursor = 0
	l.offset = 0
}

func (l *ListView) ResultCount() (int, string) {
	return len(l.notes), l.SearchQuery()
}

// ── Multi-select ────────────────────────────────────────────────────────

// ToggleSelection toggles selection for the note at the cursor.
func (l *ListView) ToggleSelection() {
	if l.cursor >= len(l.notes) {
		return
	}
	id := l.notes[l.cursor].ID
	if l.selected[id] {
		delete(l.selected, id)
	} else {
		l.selected[id] = true
	}
}

// SelectAll selects all visible notes.
func (l *ListView) SelectAll() {
	for _, n := range l.notes {
		l.selected[n.ID] = true
	}
}

// ClearSelection removes all selections.
func (l *ListView) ClearSelection() {
	l.selected = make(map[string]bool)
}

// HasSelection returns true if any notes are selected.
func (l *ListView) HasSelection() bool {
	return len(l.selected) > 0
}

// SelectionCount returns the number of selected notes.
func (l *ListView) SelectionCount() int {
	return len(l.selected)
}

// SelectedIDs returns the IDs of all selected notes.
func (l *ListView) SelectedIDs() []string {
	ids := make([]string, 0, len(l.selected))
	for id := range l.selected {
		ids = append(ids, id)
	}
	return ids
}

// ── Update ──────────────────────────────────────────────────────────────

func (l *ListView) Update(msg tea.Msg) tea.Cmd {
	if l.searching {
		return l.updateSearch(msg)
	}

	if kmsg, ok := msg.(tea.KeyMsg); ok {
		if len(l.notes) == 0 {
			return nil
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
	l.scrollToCursor()
	return nil
}

func (l *ListView) updateSearch(msg tea.Msg) tea.Cmd {
	if kmsg, ok := msg.(tea.KeyMsg); ok {
		switch kmsg.Type {
		case tea.KeyEscape:
			l.ExitSearch()
			return nil
		case tea.KeyUp:
			if l.cursor > 0 {
				l.cursor--
			}
			l.scrollToCursor()
			return nil
		case tea.KeyDown:
			if l.cursor < len(l.notes)-1 {
				l.cursor++
			}
			l.scrollToCursor()
			return nil
		}
	}

	prevValue := l.input.Value()
	var cmd tea.Cmd
	l.input, cmd = l.input.Update(msg)

	// If input changed, emit a debounce tick
	if l.input.Value() != prevValue {
		query := l.SearchQuery()
		tickCmd := tea.Tick(searchDebounce, func(_ time.Time) tea.Msg {
			return SearchTickMsg{Query: query}
		})
		return tea.Batch(cmd, tickCmd)
	}

	return cmd
}

func (l *ListView) scrollToCursor() {
	visibleLines := l.visibleLines()
	if l.cursor < l.offset {
		l.offset = l.cursor
	}
	if l.cursor >= l.offset+visibleLines {
		l.offset = l.cursor - visibleLines + 1
	}
}

func (l ListView) visibleLines() int {
	v := l.height - 3
	if l.searching {
		v -= 2 // search input + gap
	}
	if v < 1 {
		v = 1
	}
	return v
}

// ── View ────────────────────────────────────────────────────────────────

func (l ListView) View() string {
	if l.searching {
		return l.viewSearch()
	}

	if len(l.notes) == 0 {
		return theme.ListTitle.Render("jot") + "\n\nNo notes yet. Press n to create one."
	}

	var b strings.Builder
	title := fmt.Sprintf("jot — %d notes", len(l.notes))
	if l.contextFilter {
		title += " (this project)"
	}
	if len(l.selected) > 0 {
		title += fmt.Sprintf(" [%d selected]", len(l.selected))
	}
	b.WriteString(theme.ListTitle.Render(title))
	b.WriteString("\n")

	l.renderNotes(&b)
	return b.String()
}

func (l ListView) viewSearch() string {
	var b strings.Builder

	b.WriteString(theme.SearchPrompt.Render("Filter: "))
	b.WriteString(l.input.View())
	b.WriteString("\n")

	q := strings.TrimSpace(l.input.Value())
	if len(l.notes) == 0 {
		if q != "" {
			b.WriteString(theme.SearchDim.Render("No results."))
		}
		return b.String()
	}

	b.WriteString(theme.SearchDim.Render(fmt.Sprintf("%d results", len(l.notes))))
	b.WriteString("\n")

	l.renderNotes(&b)
	return b.String()
}

func (l ListView) renderNotes(b *strings.Builder) {
	visibleLines := l.visibleLines()
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

		check := " "
		if l.selected[n.ID] {
			check = theme.ListCheck.Render("✓")
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
			row := fmt.Sprintf("%s%s %s"+titleFmt+"  %s  %s", cursor, check, pin, render.Truncate(title, titleWidth), age, tags)
			line := theme.ListSelected.Width(l.width).Render(row)
			b.WriteString(line)
		} else {
			line := fmt.Sprintf(" %s %s"+titleFmt+"  %s  %s", check, pin, render.Truncate(title, titleWidth), theme.ListDim.Render(age), theme.ListTag.Render(tags))
			b.WriteString(line)
		}

		if i < end-1 {
			b.WriteString("\n")
		}
	}
}
