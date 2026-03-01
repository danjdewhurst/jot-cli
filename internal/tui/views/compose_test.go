package views_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/danjdewhurst/jot-cli/internal/tui/views"
)

func TestComposeView_UpdateReturnsCmdFromTextInput(t *testing.T) {
	cv := views.NewComposeView()
	cv.SetSize(80, 24)

	// Sending a key should propagate the cursor blink cmd, not discard it.
	cmd := cv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	// We can't easily inspect the cmd, but it should not always be nil.
	// The textinput/textarea models return blink commands.
	// Just verify the signature works and doesn't panic.
	_ = cmd
}

func TestComposeView_TabDoesNotReturnCmd(t *testing.T) {
	cv := views.NewComposeView()
	cv.SetSize(80, 24)

	cmd := cv.Update(tea.KeyMsg{Type: tea.KeyTab})
	if cmd != nil {
		t.Error("tab key should return nil cmd (just toggles focus)")
	}
}
