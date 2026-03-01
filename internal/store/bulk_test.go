package store_test

import (
	"testing"

	"github.com/danjdewhurst/jot-cli/internal/model"
)

func TestArchiveNotes(t *testing.T) {
	s := newTestStore(t)

	n1, _ := s.CreateNote("Note 1", "body", nil)
	n2, _ := s.CreateNote("Note 2", "body", nil)
	n3, _ := s.CreateNote("Note 3", "body", nil)

	count, err := s.ArchiveNotes([]string{n1.ID, n3.ID})
	if err != nil {
		t.Fatalf("archiving notes: %v", err)
	}
	if count != 2 {
		t.Errorf("got count %d, want 2", count)
	}

	// Archived notes should not appear in default list
	notes, _ := s.ListNotes(model.NoteFilter{})
	if len(notes) != 1 {
		t.Errorf("got %d notes, want 1", len(notes))
	}
	if notes[0].ID != n2.ID {
		t.Errorf("remaining note = %s, want %s", notes[0].ID, n2.ID)
	}

	// All should appear with archived filter
	notes, _ = s.ListNotes(model.NoteFilter{Archived: true})
	if len(notes) != 3 {
		t.Errorf("got %d notes with archived filter, want 3", len(notes))
	}
}

func TestArchiveNotes_Empty(t *testing.T) {
	s := newTestStore(t)

	count, err := s.ArchiveNotes(nil)
	if err != nil {
		t.Fatalf("archiving empty list: %v", err)
	}
	if count != 0 {
		t.Errorf("got count %d, want 0", count)
	}
}

func TestDeleteNotes(t *testing.T) {
	s := newTestStore(t)

	n1, _ := s.CreateNote("Note 1", "searchable one", []model.Tag{{Key: "project", Value: "a"}})
	n2, _ := s.CreateNote("Note 2", "searchable two", nil)
	_, _ = s.CreateNote("Note 3", "searchable three", nil)

	count, err := s.DeleteNotes([]string{n1.ID, n2.ID})
	if err != nil {
		t.Fatalf("deleting notes: %v", err)
	}
	if count != 2 {
		t.Errorf("got count %d, want 2", count)
	}

	// Only n3 should remain
	notes, _ := s.ListNotes(model.NoteFilter{Archived: true})
	if len(notes) != 1 {
		t.Errorf("got %d notes, want 1", len(notes))
	}

	// FTS should be cleaned up
	results, _ := s.Search("searchable one", nil)
	if len(results) != 0 {
		t.Errorf("got %d search results for deleted note, want 0", len(results))
	}

	// Tags should be cascade-deleted
	tags, _ := s.ListTags("")
	if len(tags) != 0 {
		t.Errorf("got %d tags after bulk delete, want 0", len(tags))
	}
}

func TestDeleteNotes_Empty(t *testing.T) {
	s := newTestStore(t)

	count, err := s.DeleteNotes(nil)
	if err != nil {
		t.Fatalf("deleting empty list: %v", err)
	}
	if count != 0 {
		t.Errorf("got count %d, want 0", count)
	}
}

func TestAddTagToNotes(t *testing.T) {
	s := newTestStore(t)

	n1, _ := s.CreateNote("Note 1", "body one", nil)
	n2, _ := s.CreateNote("Note 2", "body two", nil)
	_, _ = s.CreateNote("Note 3", "body three", nil)

	tag := model.Tag{Key: "status", Value: "done"}
	count, err := s.AddTagToNotes([]string{n1.ID, n2.ID}, tag)
	if err != nil {
		t.Fatalf("adding tag to notes: %v", err)
	}
	if count != 2 {
		t.Errorf("got count %d, want 2", count)
	}

	// Verify tags were added
	got1, _ := s.GetNote(n1.ID)
	found := false
	for _, t := range got1.Tags {
		if t.Key == "status" && t.Value == "done" {
			found = true
		}
	}
	if !found {
		t.Error("tag not found on note 1")
	}

	// Verify tag was added to note 2 as well
	got2, _ := s.GetNote(n2.ID)
	found = false
	for _, t := range got2.Tags {
		if t.Key == "status" && t.Value == "done" {
			found = true
		}
	}
	if !found {
		t.Error("tag not found on note 2")
	}
}

func TestAddTagToNotes_Empty(t *testing.T) {
	s := newTestStore(t)

	count, err := s.AddTagToNotes(nil, model.Tag{Key: "k", Value: "v"})
	if err != nil {
		t.Fatalf("adding tag to empty list: %v", err)
	}
	if count != 0 {
		t.Errorf("got count %d, want 0", count)
	}
}

func TestAddTagToNotes_Idempotent(t *testing.T) {
	s := newTestStore(t)

	n1, _ := s.CreateNote("Note 1", "body", nil)
	tag := model.Tag{Key: "status", Value: "done"}

	// Add same tag twice — should not error
	_, _ = s.AddTagToNotes([]string{n1.ID}, tag)
	count, err := s.AddTagToNotes([]string{n1.ID}, tag)
	if err != nil {
		t.Fatalf("adding duplicate tag: %v", err)
	}
	if count != 1 {
		t.Errorf("got count %d, want 1", count)
	}

	// Should only have one instance of the tag
	got, _ := s.GetNote(n1.ID)
	tagCount := 0
	for _, t := range got.Tags {
		if t.Key == "status" && t.Value == "done" {
			tagCount++
		}
	}
	if tagCount != 1 {
		t.Errorf("got %d instances of tag, want 1", tagCount)
	}
}

func TestPinNotes(t *testing.T) {
	s := newTestStore(t)

	n1, _ := s.CreateNote("Note 1", "body", nil)
	n2, _ := s.CreateNote("Note 2", "body", nil)
	n3, _ := s.CreateNote("Note 3", "body", nil)

	count, err := s.PinNotes([]string{n1.ID, n3.ID})
	if err != nil {
		t.Fatalf("pinning notes: %v", err)
	}
	if count != 2 {
		t.Errorf("got count %d, want 2", count)
	}

	got1, _ := s.GetNote(n1.ID)
	if !got1.Pinned {
		t.Error("note 1 should be pinned")
	}

	got2, _ := s.GetNote(n2.ID)
	if got2.Pinned {
		t.Error("note 2 should not be pinned")
	}

	got3, _ := s.GetNote(n3.ID)
	if !got3.Pinned {
		t.Error("note 3 should be pinned")
	}
}

func TestPinNotes_Empty(t *testing.T) {
	s := newTestStore(t)

	count, err := s.PinNotes(nil)
	if err != nil {
		t.Fatalf("pinning empty list: %v", err)
	}
	if count != 0 {
		t.Errorf("got count %d, want 0", count)
	}
}

func TestUnpinNotes(t *testing.T) {
	s := newTestStore(t)

	n1, _ := s.CreateNote("Note 1", "body", nil)
	n2, _ := s.CreateNote("Note 2", "body", nil)
	_ = s.PinNote(n1.ID)
	_ = s.PinNote(n2.ID)

	count, err := s.UnpinNotes([]string{n1.ID})
	if err != nil {
		t.Fatalf("unpinning notes: %v", err)
	}
	if count != 1 {
		t.Errorf("got count %d, want 1", count)
	}

	got1, _ := s.GetNote(n1.ID)
	if got1.Pinned {
		t.Error("note 1 should be unpinned")
	}

	got2, _ := s.GetNote(n2.ID)
	if !got2.Pinned {
		t.Error("note 2 should still be pinned")
	}
}

func TestUnpinNotes_Empty(t *testing.T) {
	s := newTestStore(t)

	count, err := s.UnpinNotes(nil)
	if err != nil {
		t.Fatalf("unpinning empty list: %v", err)
	}
	if count != 0 {
		t.Errorf("got count %d, want 0", count)
	}
}
