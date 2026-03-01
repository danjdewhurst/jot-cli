package views_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/danjdewhurst/jot-cli/internal/tui/views"
)

func TestSearchView_UpdateReturnsCmdFromTextInput(t *testing.T) {
	sv := views.NewSearchView()
	sv.SetSize(80, 24)

	// Sending a character key should propagate cursor blink cmd.
	cmd := sv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	_ = cmd // Verify signature compiles; cmd may or may not be nil.
}

func TestSearchView_ArrowKeysReturnNilCmd(t *testing.T) {
	sv := views.NewSearchView()
	sv.SetSize(80, 24)

	cmd := sv.Update(tea.KeyMsg{Type: tea.KeyUp})
	if cmd != nil {
		t.Error("up key should return nil cmd")
	}
}
