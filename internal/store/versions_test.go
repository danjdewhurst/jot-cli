package store_test

import (
	"errors"
	"testing"

	"github.com/danjdewhurst/jot-cli/internal/store"
)

func TestSaveVersion_CreatesSnapshotOnUpdate(t *testing.T) {
	s := newTestStore(t)

	note, _ := s.CreateNote("Original Title", "Original body", nil)

	// First update should snapshot the original state as version 1
	_, err := s.UpdateNote(note.ID, "Updated Title", "Updated body")
	if err != nil {
		t.Fatalf("updating note: %v", err)
	}

	versions, err := s.ListVersions(note.ID)
	if err != nil {
		t.Fatalf("listing versions: %v", err)
	}
	if len(versions) != 1 {
		t.Fatalf("got %d versions, want 1", len(versions))
	}
	if versions[0].Version != 1 {
		t.Errorf("version = %d, want 1", versions[0].Version)
	}
	if versions[0].Title != "Original Title" {
		t.Errorf("title = %q, want %q", versions[0].Title, "Original Title")
	}
	if versions[0].Body != "Original body" {
		t.Errorf("body = %q, want %q", versions[0].Body, "Original body")
	}
}

func TestSaveVersion_IncrementsVersionNumber(t *testing.T) {
	s := newTestStore(t)

	note, _ := s.CreateNote("V0", "Body 0", nil)

	// Two updates should create versions 1 and 2
	_, _ = s.UpdateNote(note.ID, "V1", "Body 1")
	_, _ = s.UpdateNote(note.ID, "V2", "Body 2")

	versions, err := s.ListVersions(note.ID)
	if err != nil {
		t.Fatalf("listing versions: %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("got %d versions, want 2", len(versions))
	}

	// Versions should be ordered by version number descending (newest first)
	if versions[0].Version != 2 {
		t.Errorf("first version = %d, want 2", versions[0].Version)
	}
	if versions[0].Title != "V1" {
		t.Errorf("version 2 title = %q, want %q", versions[0].Title, "V1")
	}
	if versions[1].Version != 1 {
		t.Errorf("second version = %d, want 1", versions[1].Version)
	}
	if versions[1].Title != "V0" {
		t.Errorf("version 1 title = %q, want %q", versions[1].Title, "V0")
	}
}

func TestGetVersion(t *testing.T) {
	s := newTestStore(t)

	note, _ := s.CreateNote("Original", "Original body", nil)
	_, _ = s.UpdateNote(note.ID, "Updated", "Updated body")

	v, err := s.GetVersion(note.ID, 1)
	if err != nil {
		t.Fatalf("getting version: %v", err)
	}
	if v.Title != "Original" {
		t.Errorf("title = %q, want %q", v.Title, "Original")
	}
	if v.Body != "Original body" {
		t.Errorf("body = %q, want %q", v.Body, "Original body")
	}
	if v.NoteID != note.ID {
		t.Errorf("note_id = %q, want %q", v.NoteID, note.ID)
	}
}

func TestGetVersion_NotFound(t *testing.T) {
	s := newTestStore(t)

	note, _ := s.CreateNote("Test", "body", nil)

	_, err := s.GetVersion(note.ID, 999)
	if !errors.Is(err, store.ErrVersionNotFound) {
		t.Errorf("error = %v, want ErrVersionNotFound", err)
	}
}

func TestGetVersion_NoteNotFound(t *testing.T) {
	s := newTestStore(t)

	_, err := s.GetVersion("nonexistent", 1)
	if !errors.Is(err, store.ErrVersionNotFound) {
		t.Errorf("error = %v, want ErrVersionNotFound", err)
	}
}

func TestListVersions_Empty(t *testing.T) {
	s := newTestStore(t)

	note, _ := s.CreateNote("No Updates", "body", nil)

	versions, err := s.ListVersions(note.ID)
	if err != nil {
		t.Fatalf("listing versions: %v", err)
	}
	if len(versions) != 0 {
		t.Errorf("got %d versions, want 0", len(versions))
	}
}

func TestVersions_CascadeDeletedWithNote(t *testing.T) {
	s := newTestStore(t)

	note, _ := s.CreateNote("To Delete", "body", nil)
	_, _ = s.UpdateNote(note.ID, "Updated", "new body")

	// Verify version exists
	versions, _ := s.ListVersions(note.ID)
	if len(versions) != 1 {
		t.Fatalf("got %d versions before delete, want 1", len(versions))
	}

	// Delete the note — versions should cascade
	if err := s.DeleteNote(note.ID); err != nil {
		t.Fatalf("deleting note: %v", err)
	}

	versions, err := s.ListVersions(note.ID)
	if err != nil {
		t.Fatalf("listing versions after delete: %v", err)
	}
	if len(versions) != 0 {
		t.Errorf("got %d versions after delete, want 0", len(versions))
	}
}
