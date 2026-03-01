package store_test

import (
	"testing"
	"time"

	"github.com/danjdewhurst/jot-cli/internal/model"
)

func TestSyncChangelog(t *testing.T) {
	s := newTestStore(t)

	// Create a note — trigger should log an upsert
	note, err := s.CreateNote("Test", "body", []model.Tag{{Key: "folder", Value: "work"}})
	if err != nil {
		t.Fatalf("creating note: %v", err)
	}

	entries, err := s.UnsyncedChanges()
	if err != nil {
		t.Fatalf("getting unsynced changes: %v", err)
	}
	// Expect at least 1 entry from the note insert (plus tag insert trigger)
	if len(entries) < 1 {
		t.Fatalf("got %d entries, want >= 1", len(entries))
	}
	if entries[0].NoteID != note.ID {
		t.Errorf("note_id = %q, want %q", entries[0].NoteID, note.ID)
	}
	if entries[0].Action != "upsert" {
		t.Errorf("action = %q, want %q", entries[0].Action, "upsert")
	}

	// Update note — should add another entry
	_, err = s.UpdateNote(note.ID, "Updated", "new body")
	if err != nil {
		t.Fatalf("updating note: %v", err)
	}

	entries, err = s.UnsyncedChanges()
	if err != nil {
		t.Fatalf("getting unsynced changes: %v", err)
	}
	if len(entries) < 2 {
		t.Fatalf("got %d entries after update, want >= 2", len(entries))
	}

	// Delete note — should log a delete
	if err := s.DeleteNote(note.ID); err != nil {
		t.Fatalf("deleting note: %v", err)
	}

	entries, err = s.UnsyncedChanges()
	if err != nil {
		t.Fatalf("getting unsynced changes: %v", err)
	}
	hasDelete := false
	for _, e := range entries {
		if e.Action == "delete" && e.NoteID == note.ID {
			hasDelete = true
			break
		}
	}
	if !hasDelete {
		t.Error("expected a delete changelog entry after DeleteNote")
	}
}

func TestMarkChangesSynced(t *testing.T) {
	s := newTestStore(t)

	_, _ = s.CreateNote("A", "body", nil)
	_, _ = s.CreateNote("B", "body", nil)

	entries, _ := s.UnsyncedChanges()
	if len(entries) == 0 {
		t.Fatal("expected unsynced entries")
	}

	maxID := entries[len(entries)-1].ID
	if err := s.MarkChangesSynced(maxID); err != nil {
		t.Fatalf("marking synced: %v", err)
	}

	remaining, _ := s.UnsyncedChanges()
	if len(remaining) != 0 {
		t.Errorf("got %d remaining, want 0", len(remaining))
	}
}

func TestSyncMeta(t *testing.T) {
	s := newTestStore(t)

	if err := s.SetSyncMeta("machine_id", "ABC123"); err != nil {
		t.Fatalf("setting meta: %v", err)
	}

	got, err := s.GetSyncMeta("machine_id")
	if err != nil {
		t.Fatalf("getting meta: %v", err)
	}
	if got != "ABC123" {
		t.Errorf("got %q, want %q", got, "ABC123")
	}

	// Overwrite
	if err := s.SetSyncMeta("machine_id", "DEF456"); err != nil {
		t.Fatalf("overwriting meta: %v", err)
	}
	got, _ = s.GetSyncMeta("machine_id")
	if got != "DEF456" {
		t.Errorf("got %q, want %q", got, "DEF456")
	}
}

func TestUpsertNoteNew(t *testing.T) {
	s := newTestStore(t)

	now := time.Now().UTC().Truncate(time.Second)
	n := model.Note{
		ID:        "01TESTID00000000000000001",
		Title:     "Imported",
		Body:      "From another machine",
		CreatedAt: now,
		UpdatedAt: now,
		Tags:      []model.Tag{{Key: "source", Value: "remote"}},
	}

	if err := s.UpsertNote(n); err != nil {
		t.Fatalf("upserting note: %v", err)
	}

	got, err := s.GetNote(n.ID)
	if err != nil {
		t.Fatalf("getting note: %v", err)
	}
	if got.Title != "Imported" {
		t.Errorf("title = %q, want %q", got.Title, "Imported")
	}
	if len(got.Tags) != 1 || got.Tags[0].Key != "source" {
		t.Errorf("tags = %v, want [{source remote}]", got.Tags)
	}
}

func TestUpsertNoteOverwrite(t *testing.T) {
	s := newTestStore(t)

	// Create original
	note, _ := s.CreateNote("Original", "original body", []model.Tag{{Key: "folder", Value: "work"}})

	// Upsert with same ID but different content
	now := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	updated := model.Note{
		ID:        note.ID,
		Title:     "Remote Update",
		Body:      "updated from remote",
		CreatedAt: note.CreatedAt,
		UpdatedAt: now,
		Tags:      []model.Tag{{Key: "folder", Value: "home"}, {Key: "source", Value: "sync"}},
	}

	if err := s.UpsertNote(updated); err != nil {
		t.Fatalf("upserting note: %v", err)
	}

	got, err := s.GetNote(note.ID)
	if err != nil {
		t.Fatalf("getting note: %v", err)
	}
	if got.Title != "Remote Update" {
		t.Errorf("title = %q, want %q", got.Title, "Remote Update")
	}
	if got.Body != "updated from remote" {
		t.Errorf("body = %q, want %q", got.Body, "updated from remote")
	}
	if len(got.Tags) != 2 {
		t.Errorf("got %d tags, want 2", len(got.Tags))
	}
}

func TestClearChangelogForNotes(t *testing.T) {
	s := newTestStore(t)

	noteA, _ := s.CreateNote("A", "body", nil)
	noteB, _ := s.CreateNote("B", "body", nil)

	// Clear only A's changelog
	if err := s.ClearChangelogForNotes([]string{noteA.ID}); err != nil {
		t.Fatalf("clearing changelog: %v", err)
	}

	entries, _ := s.UnsyncedChanges()
	for _, e := range entries {
		if e.NoteID == noteA.ID {
			t.Error("expected no unsynced entries for note A after clear")
		}
	}
	// B should still have entries
	hasB := false
	for _, e := range entries {
		if e.NoteID == noteB.ID {
			hasB = true
			break
		}
	}
	if !hasB {
		t.Error("expected unsynced entries for note B")
	}
}
