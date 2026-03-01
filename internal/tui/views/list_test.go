package views_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/danjdewhurst/jot-cli/internal/model"
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

// ── Search mode tests ──────────────────────────────────────────────────

func sampleNotes() []model.Note {
	return []model.Note{
		{ID: "1", Title: "First note", Body: "body one"},
		{ID: "2", Title: "Second note", Body: "body two"},
		{ID: "3", Title: "Third note", Body: "body three"},
	}
}

func TestListView_EnterSearch_TogglesMode(t *testing.T) {
	lv := views.NewListView()
	lv.SetSize(80, 24)
	lv.SetNotes(sampleNotes())

	if lv.IsSearching() {
		t.Fatal("expected IsSearching to be false initially")
	}

	lv.EnterSearch()
	if !lv.IsSearching() {
		t.Fatal("expected IsSearching to be true after EnterSearch")
	}

	lv.ExitSearch()
	if lv.IsSearching() {
		t.Fatal("expected IsSearching to be false after ExitSearch")
	}
}

func TestListView_SearchQuery_ReturnsInputValue(t *testing.T) {
	lv := views.NewListView()
	lv.SetSize(80, 24)
	lv.SetNotes(sampleNotes())
	lv.EnterSearch()

	// Type characters into the search input
	lv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	lv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")})

	q := lv.SearchQuery()
	if q != "go" {
		t.Fatalf("expected SearchQuery() = %q, got %q", "go", q)
	}
}

func TestListView_SetSearchResults_ReplacesNotes(t *testing.T) {
	lv := views.NewListView()
	lv.SetSize(80, 24)
	lv.SetNotes(sampleNotes())
	lv.EnterSearch()

	results := []model.Note{{ID: "2", Title: "Second note"}}
	lv.SetSearchResults(results)

	note, ok := lv.SelectedNote()
	if !ok {
		t.Fatal("expected a selected note after SetSearchResults")
	}
	if note.ID != "2" {
		t.Fatalf("expected selected note ID %q, got %q", "2", note.ID)
	}
}

func TestListView_SetSearchResults_ResetsCursor(t *testing.T) {
	lv := views.NewListView()
	lv.SetSize(80, 24)
	lv.SetNotes(sampleNotes())

	// Move cursor down
	lv.Update(tea.KeyMsg{Type: tea.KeyDown})
	lv.Update(tea.KeyMsg{Type: tea.KeyDown})

	lv.EnterSearch()
	results := []model.Note{{ID: "3", Title: "Third note"}}
	lv.SetSearchResults(results)

	note, ok := lv.SelectedNote()
	if !ok {
		t.Fatal("expected a selected note")
	}
	if note.ID != "3" {
		t.Fatalf("expected cursor reset to first result, got ID %q", note.ID)
	}
}

func TestListView_ExitSearch_RestoresOriginalNotes(t *testing.T) {
	lv := views.NewListView()
	lv.SetSize(80, 24)
	original := sampleNotes()
	lv.SetNotes(original)

	lv.EnterSearch()
	lv.SetSearchResults([]model.Note{{ID: "2", Title: "Second note"}})
	lv.ExitSearch()

	// Should restore the full list
	note, ok := lv.SelectedNote()
	if !ok {
		t.Fatal("expected a selected note after ExitSearch")
	}
	if note.ID != "1" {
		t.Fatalf("expected first original note after ExitSearch, got ID %q", note.ID)
	}
}

func TestListView_Update_ReturnsTickCmd_WhenSearching(t *testing.T) {
	lv := views.NewListView()
	lv.SetSize(80, 24)
	lv.SetNotes(sampleNotes())
	lv.EnterSearch()

	cmd := lv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if cmd == nil {
		t.Fatal("expected a tea.Cmd (debounce tick) when typing in search mode")
	}
}

func TestListView_Escape_ExitsSearchMode(t *testing.T) {
	lv := views.NewListView()
	lv.SetSize(80, 24)
	lv.SetNotes(sampleNotes())
	lv.EnterSearch()

	lv.Update(tea.KeyMsg{Type: tea.KeyEscape})

	if lv.IsSearching() {
		t.Fatal("expected search mode to exit on Escape")
	}
}

func TestListView_SearchNavigation_UpDown(t *testing.T) {
	lv := views.NewListView()
	lv.SetSize(80, 24)
	lv.SetNotes(sampleNotes())
	lv.EnterSearch()
	lv.SetSearchResults(sampleNotes())

	// Move down
	lv.Update(tea.KeyMsg{Type: tea.KeyDown})
	note, _ := lv.SelectedNote()
	if note.ID != "2" {
		t.Fatalf("expected note ID %q after down, got %q", "2", note.ID)
	}

	// Move up
	lv.Update(tea.KeyMsg{Type: tea.KeyUp})
	note, _ = lv.SelectedNote()
	if note.ID != "1" {
		t.Fatalf("expected note ID %q after up, got %q", "1", note.ID)
	}
}

func TestListView_ResultCount(t *testing.T) {
	lv := views.NewListView()
	lv.SetSize(80, 24)
	lv.SetNotes(sampleNotes())
	lv.EnterSearch()

	// Type a query
	lv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	lv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	lv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	lv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})

	results := []model.Note{{ID: "1"}, {ID: "2"}}
	lv.SetSearchResults(results)

	count, query := lv.ResultCount()
	if count != 2 {
		t.Fatalf("expected count 2, got %d", count)
	}
	if query != "test" {
		t.Fatalf("expected query %q, got %q", "test", query)
	}
}

func TestListView_View_ShowsSearchInput(t *testing.T) {
	lv := views.NewListView()
	lv.SetSize(80, 24)
	lv.SetNotes(sampleNotes())
	lv.EnterSearch()

	out := lv.View()
	if out == "" {
		t.Fatal("expected non-empty view in search mode")
	}
}

func TestListView_Update_ReturnsCmd(t *testing.T) {
	// Verify the Update signature returns tea.Cmd
	lv := views.NewListView()
	lv.SetSize(80, 24)
	lv.SetNotes(sampleNotes())

	cmd := lv.Update(tea.KeyMsg{Type: tea.KeyDown})
	// In non-search mode, cmd should be nil
	if cmd != nil {
		t.Error("expected nil cmd in non-search mode")
	}
}
