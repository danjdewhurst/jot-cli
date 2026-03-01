package views_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/danjdewhurst/jot-cli/internal/tui/views"
)

func TestListView_EmptyList_PgDown_NoPanic(t *testing.T) {
	lv := views.NewListView()
	lv.SetSize(80, 24)
	// No notes set — list is empty.
	// These keys previously set cursor to len(notes)-1 == -1, causing a panic.
	keys := []string{"pgdown", "ctrl+d", "end", "G", "home", "g", "up", "k", "down", "j", "pgup", "ctrl+u"}
	for _, k := range keys {
		t.Run(k, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic on key %q with empty list: %v", k, r)
				}
			}()
			lv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
		})
	}
}

func TestListView_EmptyList_SelectedNote_ReturnsFalse(t *testing.T) {
	lv := views.NewListView()
	lv.SetSize(80, 24)
	_, ok := lv.SelectedNote()
	if ok {
		t.Error("expected no selected note on empty list")
	}
}

func TestListView_EmptyList_View_NoEmptyPanic(t *testing.T) {
	lv := views.NewListView()
	lv.SetSize(80, 24)
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic rendering empty list: %v", r)
		}
	}()
	out := lv.View()
	if out == "" {
		t.Error("expected non-empty view for empty notes list")
	}
}
