package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/danjdewhurst/jot-cli/internal/model"
	"github.com/danjdewhurst/jot-cli/internal/store"
)

func newTestApp(t *testing.T) (App, *store.Store) {
	t.Helper()
	dbPath := tempDB(t)
	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("opening store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	app := newApp(s)
	return app, s
}

func tempDB(t *testing.T) string {
	t.Helper()
	return t.TempDir() + "/test.db"
}

// ── App initialisation ─────────────────────────────────────────────────

func TestNewApp(t *testing.T) {
	app, _ := newTestApp(t)

	if app.view != viewList {
		t.Errorf("initial view = %d, want viewList", app.view)
	}
	if len(app.viewStack) != 0 {
		t.Error("expected empty viewStack on init")
	}
	if app.contextFilter {
		t.Error("expected contextFilter to be false on init")
	}
}

func TestApp_Init(t *testing.T) {
	app, _ := newTestApp(t)
	cmd := app.Init()

	if cmd == nil {
		t.Error("expected non-nil cmd from Init (loadNotes)")
	}
}

// ── View navigation ────────────────────────────────────────────────────

func TestPushView(t *testing.T) {
	app, _ := newTestApp(t)

	app.pushView(viewDetail)
	if app.view != viewDetail {
		t.Errorf("view = %d, want viewDetail", app.view)
	}
	if len(app.viewStack) != 1 || app.viewStack[0] != viewList {
		t.Errorf("viewStack = %v, want [viewList]", app.viewStack)
	}

	app.pushView(viewCompose)
	if app.view != viewCompose {
		t.Errorf("view = %d, want viewCompose", app.view)
	}
	if len(app.viewStack) != 2 {
		t.Errorf("viewStack length = %d, want 2", len(app.viewStack))
	}
}

func TestPopView(t *testing.T) {
	app, _ := newTestApp(t)

	// Push some views
	app.pushView(viewDetail)
	app.pushView(viewCompose)

	// Pop back to detail
	app.popView()
	if app.view != viewDetail {
		t.Errorf("view = %d, want viewDetail after first pop", app.view)
	}

	// Pop back to list
	app.popView()
	if app.view != viewList {
		t.Errorf("view = %d, want viewList after second pop", app.view)
	}

	// Pop when empty should be safe (no panic)
	app.popView()
	if app.view != viewList {
		t.Error("popping empty stack should not change view")
	}
}

// ── Window size handling ───────────────────────────────────────────────

func TestApp_WindowSizeMsg(t *testing.T) {
	app, _ := newTestApp(t)

	newModel, cmd := app.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	newApp := newModel.(App)

	if newApp.width != 100 {
		t.Errorf("width = %d, want 100", newApp.width)
	}
	if newApp.height != 40 {
		t.Errorf("height = %d, want 40", newApp.height)
	}
	if cmd != nil {
		t.Error("expected nil cmd from WindowSizeMsg")
	}
}

// ── Message handling ───────────────────────────────────────────────────

func TestApp_NotesLoadedMsg(t *testing.T) {
	app, _ := newTestApp(t)
	notes := []model.Note{
		{ID: "1", Title: "Note 1"},
		{ID: "2", Title: "Note 2"},
	}

	newModel, cmd := app.Update(notesLoadedMsg{notes: notes})
	newApp := newModel.(App)

	// List should have the notes
	selected, ok := newApp.list.SelectedNote()
	if !ok || selected.ID != "1" {
		t.Error("expected first note to be selected")
	}
	if cmd != nil {
		t.Error("expected nil cmd from notesLoadedMsg")
	}
}

func TestApp_NoteCreatedMsg(t *testing.T) {
	app, _ := newTestApp(t)
	app.pushView(viewCompose) // Simulate being in compose view

	newModel, cmd := app.Update(noteCreatedMsg{note: model.Note{Title: "New Note"}})
	newApp := newModel.(App)

	if newApp.statusMsg != "Created: New Note" {
		t.Errorf("statusMsg = %q, want %q", newApp.statusMsg, "Created: New Note")
	}
	if newApp.view != viewList {
		t.Error("expected to return to list view after create")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd (reload notes + clear status)")
	}
}

func TestApp_NoteUpdatedMsg(t *testing.T) {
	app, _ := newTestApp(t)
	app.pushView(viewCompose)

	newModel, cmd := app.Update(noteUpdatedMsg{note: model.Note{Title: "Updated Note"}})
	newApp := newModel.(App)

	if newApp.statusMsg != "Updated: Updated Note" {
		t.Errorf("statusMsg = %q, want %q", newApp.statusMsg, "Updated: Updated Note")
	}
	if newApp.view != viewList {
		t.Error("expected to return to list view after update")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd")
	}
}

func TestApp_NoteArchivedMsg(t *testing.T) {
	app, _ := newTestApp(t)

	newModel, cmd := app.Update(noteArchivedMsg{id: "01TEST123"})
	newApp := newModel.(App)

	if newApp.statusMsg != "Note archived" {
		t.Errorf("statusMsg = %q, want %q", newApp.statusMsg, "Note archived")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd")
	}
}

func TestApp_BulkArchivedMsg(t *testing.T) {
	app, _ := newTestApp(t)
	app.list.ToggleSelection() // Simulate having selections

	newModel, cmd := app.Update(bulkArchivedMsg{count: 5})
	newApp := newModel.(App)

	if newApp.statusMsg != "Archived 5 notes" {
		t.Errorf("statusMsg = %q, want %q", newApp.statusMsg, "Archived 5 notes")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd")
	}
}

func TestApp_BulkDeletedMsg(t *testing.T) {
	app, _ := newTestApp(t)
	app.list.ToggleSelection()

	newModel, cmd := app.Update(bulkDeletedMsg{count: 3})
	newApp := newModel.(App)

	if newApp.statusMsg != "Deleted 3 notes" {
		t.Errorf("statusMsg = %q, want %q", newApp.statusMsg, "Deleted 3 notes")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd")
	}
}

func TestApp_BulkPinnedMsg(t *testing.T) {
	app, _ := newTestApp(t)
	app.list.ToggleSelection()

	t.Run("pinned", func(t *testing.T) {
		newModel, _ := app.Update(bulkPinnedMsg{count: 2, pinned: true})
		newApp := newModel.(App)
		if newApp.statusMsg != "Pinned 2 notes" {
			t.Errorf("statusMsg = %q, want %q", newApp.statusMsg, "Pinned 2 notes")
		}
	})

	t.Run("unpinned", func(t *testing.T) {
		newModel, _ := app.Update(bulkPinnedMsg{count: 2, pinned: false})
		newApp := newModel.(App)
		if newApp.statusMsg != "Unpinned 2 notes" {
			t.Errorf("statusMsg = %q, want %q", newApp.statusMsg, "Unpinned 2 notes")
		}
	})
}

func TestApp_NotePinnedMsg(t *testing.T) {
	app, _ := newTestApp(t)

	t.Run("pinned", func(t *testing.T) {
		newModel, _ := app.Update(notePinnedMsg{id: "01TEST", pinned: true})
		newApp := newModel.(App)
		if newApp.statusMsg != "Note pinned" {
			t.Errorf("statusMsg = %q, want %q", newApp.statusMsg, "Note pinned")
		}
	})

	t.Run("unpinned", func(t *testing.T) {
		newModel, _ := app.Update(notePinnedMsg{id: "01TEST", pinned: false})
		newApp := newModel.(App)
		if newApp.statusMsg != "Note unpinned" {
			t.Errorf("statusMsg = %q, want %q", newApp.statusMsg, "Note unpinned")
		}
	})
}

func TestApp_BacklinksLoadedMsg(t *testing.T) {
	app, _ := newTestApp(t)
	backlinks := []model.Note{
		{ID: "1", Title: "Linking Note"},
	}

	newModel, cmd := app.Update(backlinksLoadedMsg{backlinks: backlinks})
	newApp := newModel.(App)

	// Detail view should have the backlinks
	// We can't easily inspect this, but we can verify no error
	_ = newApp
	if cmd != nil {
		t.Error("expected nil cmd from backlinksLoadedMsg")
	}
}

func TestApp_StatusMsg(t *testing.T) {
	app, _ := newTestApp(t)

	newModel, cmd := app.Update(statusMsg("Custom status"))
	newApp := newModel.(App)

	if newApp.statusMsg != "Custom status" {
		t.Errorf("statusMsg = %q, want %q", newApp.statusMsg, "Custom status")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd (clear status after delay)")
	}
}

func TestApp_ClearStatusMsg(t *testing.T) {
	app, _ := newTestApp(t)
	app.statusMsg = "Some message"

	newModel, cmd := app.Update(clearStatusMsg{})
	newApp := newModel.(App)

	if newApp.statusMsg != "" {
		t.Errorf("statusMsg = %q, want empty", newApp.statusMsg)
	}
	if cmd != nil {
		t.Error("expected nil cmd from clearStatusMsg")
	}
}

func TestApp_SearchResultsMsg(t *testing.T) {
	app, _ := newTestApp(t)
	app.list.EnterSearch()

	results := []store.SearchResult{
		{Note: model.Note{ID: "1", Title: "Result 1"}},
		{Note: model.Note{ID: "2", Title: "Result 2"}},
	}

	newModel, cmd := app.Update(searchResultsMsg{results: results})
	newApp := newModel.(App)

	// List should have search results
	note, ok := newApp.list.SelectedNote()
	if !ok || note.ID != "1" {
		t.Error("expected first result to be selected")
	}
	if cmd != nil {
		t.Error("expected nil cmd from searchResultsMsg")
	}
}

// ── Global key handling ─────────────────────────────────────────────────

func TestApp_QuitKey(t *testing.T) {
	app, _ := newTestApp(t)

	// Press 'q' in list view should quit
	newModel, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	_ = newModel.(App)

	// cmd should be tea.Quit - we can't easily check this, but we can verify it returns a cmd
	if cmd == nil {
		t.Error("expected quit cmd from 'q' key in list view")
	}
}

func TestApp_BackKey_NonListView(t *testing.T) {
	app, _ := newTestApp(t)
	app.pushView(viewDetail)

	// Press esc in detail view should pop back to list
	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	newApp := newModel.(App)

	if newApp.view != viewList {
		t.Errorf("view = %d, want viewList after esc", newApp.view)
	}
}

func TestApp_BackKey_ListView(t *testing.T) {
	app, _ := newTestApp(t)

	// Press esc in list view should quit
	newModel, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	_ = newModel.(App)

	if cmd == nil {
		t.Error("expected quit cmd from esc in list view")
	}
}

func TestApp_HelpKey(t *testing.T) {
	app, _ := newTestApp(t)

	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	newApp := newModel.(App)

	if newApp.view != viewHelp {
		t.Errorf("view = %d, want viewHelp", newApp.view)
	}
}

// ── View rendering ─────────────────────────────────────────────────────

func TestApp_View(t *testing.T) {
	app, _ := newTestApp(t)

	// Set a size so the views can render
	app.width = 80
	app.height = 24
	app.list.SetSize(80, 22)

	view := app.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
}

func TestApp_View_DifferentViews(t *testing.T) {
	app, _ := newTestApp(t)
	app.width = 80
	app.height = 24
	app.list.SetSize(80, 22)
	app.detail.SetSize(80, 22)
	app.compose.SetSize(80, 22)

	views := []viewID{viewList, viewDetail, viewCompose, viewHelp}
	for _, v := range views {
		app.view = v
		view := app.View()
		if view == "" {
			t.Errorf("expected non-empty view for viewID %d", v)
		}
	}
}

// ── Helper functions ───────────────────────────────────────────────────

func TestClearStatusAfter(t *testing.T) {
	cmd := clearStatusAfter(10 * time.Millisecond)
	if cmd == nil {
		t.Error("expected non-nil cmd from clearStatusAfter")
	}

	// Execute the cmd to verify it returns clearStatusMsg
	msg := cmd()
	if _, ok := msg.(clearStatusMsg); !ok {
		t.Errorf("cmd() returned %T, want clearStatusMsg", msg)
	}
}
