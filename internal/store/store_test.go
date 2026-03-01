package store_test

import (
	"path/filepath"
	"testing"

	"github.com/danjdewhurst/jot-cli/internal/model"
	"github.com/danjdewhurst/jot-cli/internal/store"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	// Use a temp file so multiple connections see the same database.
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := store.Open(path)
	if err != nil {
		t.Fatalf("opening store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestCreateAndGetNote(t *testing.T) {
	s := newTestStore(t)

	tags := []model.Tag{{Key: "folder", Value: "work"}}
	note, err := s.CreateNote("Test Note", "Hello world", tags)
	if err != nil {
		t.Fatalf("creating note: %v", err)
	}

	if note.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if note.Title != "Test Note" {
		t.Errorf("title = %q, want %q", note.Title, "Test Note")
	}

	got, err := s.GetNote(note.ID)
	if err != nil {
		t.Fatalf("getting note: %v", err)
	}

	if got.Title != "Test Note" {
		t.Errorf("got title = %q, want %q", got.Title, "Test Note")
	}
	if got.Body != "Hello world" {
		t.Errorf("got body = %q, want %q", got.Body, "Hello world")
	}
	if len(got.Tags) != 1 || got.Tags[0].Key != "folder" {
		t.Errorf("got tags = %v, want [{folder work}]", got.Tags)
	}
}

func TestListNotes(t *testing.T) {
	s := newTestStore(t)

	s.CreateNote("First", "Body 1", nil)
	s.CreateNote("Second", "Body 2", nil)
	s.CreateNote("Third", "Body 3", nil)

	notes, err := s.ListNotes(model.NoteFilter{})
	if err != nil {
		t.Fatalf("listing notes: %v", err)
	}
	if len(notes) != 3 {
		t.Errorf("got %d notes, want 3", len(notes))
	}
}

func TestListNotesWithTagFilter(t *testing.T) {
	s := newTestStore(t)

	s.CreateNote("Work Note", "work stuff", []model.Tag{{Key: "folder", Value: "work"}})
	s.CreateNote("Personal Note", "personal stuff", []model.Tag{{Key: "folder", Value: "home"}})

	notes, err := s.ListNotes(model.NoteFilter{
		Tags: []model.Tag{{Key: "folder", Value: "work"}},
	})
	if err != nil {
		t.Fatalf("listing notes: %v", err)
	}
	if len(notes) != 1 {
		t.Errorf("got %d notes, want 1", len(notes))
	}
	if notes[0].Title != "Work Note" {
		t.Errorf("got title = %q, want %q", notes[0].Title, "Work Note")
	}
}

func TestUpdateNote(t *testing.T) {
	s := newTestStore(t)

	note, _ := s.CreateNote("Original", "Original body", nil)

	updated, err := s.UpdateNote(note.ID, "Updated", "New body")
	if err != nil {
		t.Fatalf("updating note: %v", err)
	}
	if updated.Title != "Updated" {
		t.Errorf("title = %q, want %q", updated.Title, "Updated")
	}
	if updated.Body != "New body" {
		t.Errorf("body = %q, want %q", updated.Body, "New body")
	}
}

func TestArchiveNote(t *testing.T) {
	s := newTestStore(t)

	note, _ := s.CreateNote("To Archive", "body", nil)

	if err := s.ArchiveNote(note.ID); err != nil {
		t.Fatalf("archiving: %v", err)
	}

	// Should not appear in unarchived list
	notes, _ := s.ListNotes(model.NoteFilter{})
	if len(notes) != 0 {
		t.Errorf("got %d notes, want 0 (archived note should be hidden)", len(notes))
	}

	// Should appear with archived filter
	notes, _ = s.ListNotes(model.NoteFilter{Archived: true})
	if len(notes) != 1 {
		t.Errorf("got %d notes, want 1 with archived filter", len(notes))
	}
}

func TestDeleteNote(t *testing.T) {
	s := newTestStore(t)

	note, _ := s.CreateNote("To Delete", "body", nil)

	if err := s.DeleteNote(note.ID); err != nil {
		t.Fatalf("deleting: %v", err)
	}

	_, err := s.GetNote(note.ID)
	if err == nil {
		t.Error("expected error getting deleted note")
	}
}

func TestSearchNotes(t *testing.T) {
	s := newTestStore(t)

	s.CreateNote("Go Programming", "Learn Go concurrency patterns", nil)
	s.CreateNote("Python Basics", "Variables and loops", nil)
	s.CreateNote("Rust Overview", "Memory safety without GC", nil)

	results, err := s.Search("Go concurrency", nil)
	if err != nil {
		t.Fatalf("searching: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
	if results[0].Note.Title != "Go Programming" {
		t.Errorf("first result = %q, want %q", results[0].Note.Title, "Go Programming")
	}
}

func TestSearchByTag(t *testing.T) {
	s := newTestStore(t)

	s.CreateNote("Tagged Note", "content here", []model.Tag{{Key: "project", Value: "alpha"}})
	s.CreateNote("Other Note", "content here too", []model.Tag{{Key: "project", Value: "beta"}})

	results, err := s.Search("content", []model.Tag{{Key: "project", Value: "alpha"}})
	if err != nil {
		t.Fatalf("searching: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("got %d results, want 1", len(results))
	}
}

func TestAddAndRemoveTag(t *testing.T) {
	s := newTestStore(t)

	note, _ := s.CreateNote("Test", "body", nil)

	tag := model.Tag{Key: "priority", Value: "high"}
	if err := s.AddTag(note.ID, tag); err != nil {
		t.Fatalf("adding tag: %v", err)
	}

	got, _ := s.GetNote(note.ID)
	if len(got.Tags) != 1 {
		t.Fatalf("got %d tags, want 1", len(got.Tags))
	}

	if err := s.RemoveTag(note.ID, tag); err != nil {
		t.Fatalf("removing tag: %v", err)
	}

	got, _ = s.GetNote(note.ID)
	if len(got.Tags) != 0 {
		t.Errorf("got %d tags, want 0", len(got.Tags))
	}
}

func TestListTags(t *testing.T) {
	s := newTestStore(t)

	s.CreateNote("A", "", []model.Tag{{Key: "folder", Value: "work"}, {Key: "git_repo", Value: "myapp"}})
	s.CreateNote("B", "", []model.Tag{{Key: "folder", Value: "home"}})

	tags, err := s.ListTags("")
	if err != nil {
		t.Fatalf("listing tags: %v", err)
	}
	if len(tags) != 3 {
		t.Errorf("got %d tags, want 3", len(tags))
	}

	tags, err = s.ListTags("folder")
	if err != nil {
		t.Fatalf("listing tags by key: %v", err)
	}
	if len(tags) != 2 {
		t.Errorf("got %d folder tags, want 2", len(tags))
	}
}
