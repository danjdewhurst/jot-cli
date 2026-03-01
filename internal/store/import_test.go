package store_test

import (
	"testing"
	"time"

	"github.com/danjdewhurst/jot-cli/internal/model"
)

func TestImportNote_New(t *testing.T) {
	s := newTestStore(t)

	now := time.Now().UTC().Truncate(time.Second)
	n := model.Note{
		ID:        "01JEXAMPLE00000000000000001",
		Title:     "Imported Note",
		Body:      "Some content",
		CreatedAt: now.Add(-24 * time.Hour),
		UpdatedAt: now,
		Tags:      []model.Tag{{Key: "project", Value: "alpha"}},
	}

	created, err := s.ImportNote(n)
	if err != nil {
		t.Fatalf("importing note: %v", err)
	}
	if !created {
		t.Fatal("expected note to be created")
	}

	got, err := s.GetNote(n.ID)
	if err != nil {
		t.Fatalf("getting imported note: %v", err)
	}
	if got.Title != n.Title {
		t.Errorf("title = %q, want %q", got.Title, n.Title)
	}
	if got.Body != n.Body {
		t.Errorf("body = %q, want %q", got.Body, n.Body)
	}
	if !got.CreatedAt.Equal(n.CreatedAt) {
		t.Errorf("created_at = %v, want %v", got.CreatedAt, n.CreatedAt)
	}
	if !got.UpdatedAt.Equal(n.UpdatedAt) {
		t.Errorf("updated_at = %v, want %v", got.UpdatedAt, n.UpdatedAt)
	}
	if len(got.Tags) != 1 || got.Tags[0].Key != "project" || got.Tags[0].Value != "alpha" {
		t.Errorf("tags = %v, want [{project alpha}]", got.Tags)
	}
}

func TestImportNote_Duplicate(t *testing.T) {
	s := newTestStore(t)

	now := time.Now().UTC().Truncate(time.Second)
	n := model.Note{
		ID:        "01JEXAMPLE00000000000000002",
		Title:     "Original",
		Body:      "Original body",
		CreatedAt: now,
		UpdatedAt: now,
	}

	created, err := s.ImportNote(n)
	if err != nil {
		t.Fatalf("first import: %v", err)
	}
	if !created {
		t.Fatal("expected first import to create")
	}

	// Import again with different content — should skip
	n.Title = "Modified"
	n.Body = "Modified body"
	created, err = s.ImportNote(n)
	if err != nil {
		t.Fatalf("second import: %v", err)
	}
	if created {
		t.Fatal("expected second import to be skipped")
	}

	// Verify original is unchanged
	got, err := s.GetNote(n.ID)
	if err != nil {
		t.Fatalf("getting note: %v", err)
	}
	if got.Title != "Original" {
		t.Errorf("title = %q, want %q", got.Title, "Original")
	}
}

func TestImportNote_WithTags(t *testing.T) {
	s := newTestStore(t)

	now := time.Now().UTC().Truncate(time.Second)
	n := model.Note{
		ID:        "01JEXAMPLE00000000000000003",
		Title:     "Tagged Note",
		Body:      "Searchable content",
		CreatedAt: now,
		UpdatedAt: now,
		Tags: []model.Tag{
			{Key: "folder", Value: "work"},
			{Key: "project", Value: "beta"},
		},
	}

	created, err := s.ImportNote(n)
	if err != nil {
		t.Fatalf("importing note: %v", err)
	}
	if !created {
		t.Fatal("expected note to be created")
	}

	// Verify tags are stored
	got, err := s.GetNote(n.ID)
	if err != nil {
		t.Fatalf("getting note: %v", err)
	}
	if len(got.Tags) != 2 {
		t.Fatalf("got %d tags, want 2", len(got.Tags))
	}

	// Verify FTS is searchable
	results, err := s.Search("Searchable", nil)
	if err != nil {
		t.Fatalf("searching: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Note.ID != n.ID {
		t.Errorf("search result ID = %q, want %q", results[0].Note.ID, n.ID)
	}
}

func TestImportNote_Archived(t *testing.T) {
	s := newTestStore(t)

	now := time.Now().UTC().Truncate(time.Second)
	n := model.Note{
		ID:        "01JEXAMPLE00000000000000004",
		Title:     "Archived Note",
		Body:      "Old content",
		CreatedAt: now,
		UpdatedAt: now,
		Archived:  true,
	}

	created, err := s.ImportNote(n)
	if err != nil {
		t.Fatalf("importing note: %v", err)
	}
	if !created {
		t.Fatal("expected note to be created")
	}

	// Should not appear in default list (archived=false means exclude archived)
	notes, err := s.ListNotes(model.NoteFilter{})
	if err != nil {
		t.Fatalf("listing notes: %v", err)
	}
	if len(notes) != 0 {
		t.Errorf("got %d notes in default list, want 0", len(notes))
	}

	// Should appear when including archived
	notes, err = s.ListNotes(model.NoteFilter{Archived: true})
	if err != nil {
		t.Fatalf("listing archived notes: %v", err)
	}
	if len(notes) != 1 {
		t.Errorf("got %d notes with archived filter, want 1", len(notes))
	}
}
