package store_test

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

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
	t.Cleanup(func() { _ = s.Close() })
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

	_, _ = s.CreateNote("First", "Body 1", nil)
	_, _ = s.CreateNote("Second", "Body 2", nil)
	_, _ = s.CreateNote("Third", "Body 3", nil)

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

	_, _ = s.CreateNote("Work Note", "work stuff", []model.Tag{{Key: "folder", Value: "work"}})
	_, _ = s.CreateNote("Personal Note", "personal stuff", []model.Tag{{Key: "folder", Value: "home"}})

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

	_, _ = s.CreateNote("Go Programming", "Learn Go concurrency patterns", nil)
	_, _ = s.CreateNote("Python Basics", "Variables and loops", nil)
	_, _ = s.CreateNote("Rust Overview", "Memory safety without GC", nil)

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

	_, _ = s.CreateNote("Tagged Note", "content here", []model.Tag{{Key: "project", Value: "alpha"}})
	_, _ = s.CreateNote("Other Note", "content here too", []model.Tag{{Key: "project", Value: "beta"}})

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

func TestPinNote(t *testing.T) {
	s := newTestStore(t)

	note, _ := s.CreateNote("To Pin", "body", nil)
	if note.Pinned {
		t.Fatal("new note should not be pinned")
	}

	if err := s.PinNote(note.ID); err != nil {
		t.Fatalf("pinning note: %v", err)
	}

	got, err := s.GetNote(note.ID)
	if err != nil {
		t.Fatalf("getting note: %v", err)
	}
	if !got.Pinned {
		t.Error("expected note to be pinned")
	}
}

func TestUnpinNote(t *testing.T) {
	s := newTestStore(t)

	note, _ := s.CreateNote("To Unpin", "body", nil)
	_ = s.PinNote(note.ID)

	if err := s.UnpinNote(note.ID); err != nil {
		t.Fatalf("unpinning note: %v", err)
	}

	got, _ := s.GetNote(note.ID)
	if got.Pinned {
		t.Error("expected note to be unpinned")
	}
}

func TestTogglePin(t *testing.T) {
	s := newTestStore(t)

	note, _ := s.CreateNote("Toggle Me", "body", nil)

	pinned, err := s.TogglePin(note.ID)
	if err != nil {
		t.Fatalf("first toggle: %v", err)
	}
	if !pinned {
		t.Error("expected pinned after first toggle")
	}

	pinned, err = s.TogglePin(note.ID)
	if err != nil {
		t.Fatalf("second toggle: %v", err)
	}
	if pinned {
		t.Error("expected unpinned after second toggle")
	}
}

func TestPinnedNotesFloatToTop(t *testing.T) {
	s := newTestStore(t)

	oldest, _ := s.CreateNote("Oldest", "body", nil)
	_, _ = s.CreateNote("Middle", "body", nil)
	_, _ = s.CreateNote("Newest", "body", nil)

	_ = s.PinNote(oldest.ID)

	notes, err := s.ListNotes(model.NoteFilter{})
	if err != nil {
		t.Fatalf("listing notes: %v", err)
	}
	if len(notes) != 3 {
		t.Fatalf("got %d notes, want 3", len(notes))
	}
	if notes[0].ID != oldest.ID {
		t.Errorf("expected pinned note first, got %q", notes[0].Title)
	}
}

func TestPinnedOnlyFilter(t *testing.T) {
	s := newTestStore(t)

	pinned, _ := s.CreateNote("Pinned", "body", nil)
	_, _ = s.CreateNote("Not Pinned", "body", nil)

	_ = s.PinNote(pinned.ID)

	notes, err := s.ListNotes(model.NoteFilter{PinnedOnly: true})
	if err != nil {
		t.Fatalf("listing notes: %v", err)
	}
	if len(notes) != 1 {
		t.Errorf("got %d notes, want 1", len(notes))
	}
	if notes[0].Title != "Pinned" {
		t.Errorf("got title %q, want %q", notes[0].Title, "Pinned")
	}
}

func TestPinNonExistentNote(t *testing.T) {
	s := newTestStore(t)

	if err := s.PinNote("nonexistent"); err == nil {
		t.Error("expected error pinning non-existent note")
	}
}

func TestArchivedAndPinned(t *testing.T) {
	s := newTestStore(t)

	note, _ := s.CreateNote("Pinned and Archived", "body", nil)
	_ = s.PinNote(note.ID)
	_ = s.ArchiveNote(note.ID)

	// Should not appear in default list
	notes, _ := s.ListNotes(model.NoteFilter{})
	if len(notes) != 0 {
		t.Errorf("got %d notes, want 0 (archived pinned note should be hidden)", len(notes))
	}

	// Should appear with archived filter and still be pinned
	notes, _ = s.ListNotes(model.NoteFilter{Archived: true})
	if len(notes) != 1 {
		t.Fatalf("got %d notes, want 1 with archived filter", len(notes))
	}
	if !notes[0].Pinned {
		t.Error("expected note to still be pinned after archiving")
	}
}

func TestListTags(t *testing.T) {
	s := newTestStore(t)

	_, _ = s.CreateNote("A", "", []model.Tag{{Key: "folder", Value: "work"}, {Key: "git_repo", Value: "myapp"}})
	_, _ = s.CreateNote("B", "", []model.Tag{{Key: "folder", Value: "home"}})

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

func TestListNotes_Since(t *testing.T) {
	s := newTestStore(t)

	// Create notes — all created "now" by the store
	_, _ = s.CreateNote("Old", "body", nil)
	time.Sleep(10 * time.Millisecond)
	_, _ = s.CreateNote("New", "body", nil)

	// Since = a moment ago should return both
	past := time.Now().UTC().Add(-1 * time.Hour)
	notes, err := s.ListNotes(model.NoteFilter{Since: &past})
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if len(notes) != 2 {
		t.Errorf("got %d notes, want 2", len(notes))
	}

	// Since = future should return none
	future := time.Now().UTC().Add(1 * time.Hour)
	notes, err = s.ListNotes(model.NoteFilter{Since: &future})
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if len(notes) != 0 {
		t.Errorf("got %d notes, want 0", len(notes))
	}
}

func TestListNotes_Until(t *testing.T) {
	s := newTestStore(t)

	_, _ = s.CreateNote("Note", "body", nil)

	// Until = future should return the note
	future := time.Now().UTC().Add(1 * time.Hour)
	notes, err := s.ListNotes(model.NoteFilter{Until: &future})
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if len(notes) != 1 {
		t.Errorf("got %d notes, want 1", len(notes))
	}

	// Until = past should return none
	past := time.Now().UTC().Add(-1 * time.Hour)
	notes, err = s.ListNotes(model.NoteFilter{Until: &past})
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if len(notes) != 0 {
		t.Errorf("got %d notes, want 0", len(notes))
	}
}

func TestListNotes_SinceAndUntil(t *testing.T) {
	s := newTestStore(t)

	_, _ = s.CreateNote("Note", "body", nil)

	// Range that includes now
	past := time.Now().UTC().Add(-1 * time.Hour)
	future := time.Now().UTC().Add(1 * time.Hour)
	notes, err := s.ListNotes(model.NoteFilter{Since: &past, Until: &future})
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if len(notes) != 1 {
		t.Errorf("got %d notes, want 1", len(notes))
	}

	// Range entirely in the past
	pastStart := time.Now().UTC().Add(-2 * time.Hour)
	pastEnd := time.Now().UTC().Add(-1 * time.Hour)
	notes, err = s.ListNotes(model.NoteFilter{Since: &pastStart, Until: &pastEnd})
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if len(notes) != 0 {
		t.Errorf("got %d notes, want 0", len(notes))
	}
}

func TestForeignKeysEnabled(t *testing.T) {
	s := newTestStore(t)

	// Foreign keys should be ON — deleting a note should cascade-delete its tags
	note, err := s.CreateNote("FK Test", "body", []model.Tag{{Key: "project", Value: "test"}})
	if err != nil {
		t.Fatalf("creating note: %v", err)
	}

	tags, _ := s.ListTags("")
	if len(tags) == 0 {
		t.Fatal("expected at least one tag after create")
	}

	if err := s.DeleteNote(note.ID); err != nil {
		t.Fatalf("deleting note: %v", err)
	}

	// Tags should be cascade-deleted if foreign_keys is ON
	tags, err = s.ListTags("")
	if err != nil {
		t.Fatalf("listing tags: %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("got %d tags after delete, want 0 (foreign key cascade should have removed them)", len(tags))
	}
}

func TestDeleteNoteClearsFTS(t *testing.T) {
	s := newTestStore(t)

	note, _ := s.CreateNote("Searchable Title", "unique searchable body content", nil)

	if err := s.DeleteNote(note.ID); err != nil {
		t.Fatalf("deleting note: %v", err)
	}

	results, err := s.Search("searchable", nil)
	if err != nil {
		t.Fatalf("searching after delete: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %d search results after delete, want 0", len(results))
	}
}

func TestUpdateNoteKeepsFTSInSync(t *testing.T) {
	s := newTestStore(t)

	note, _ := s.CreateNote("Original Title", "original body content", nil)

	_, err := s.UpdateNote(note.ID, "Updated Title", "completely new body")
	if err != nil {
		t.Fatalf("updating note: %v", err)
	}

	// Old content should not match
	results, err := s.Search("original", nil)
	if err != nil {
		t.Fatalf("searching for old content: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %d results for old content, want 0", len(results))
	}

	// New content should match
	results, err = s.Search("completely new", nil)
	if err != nil {
		t.Fatalf("searching for new content: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("got %d results for new content, want 1", len(results))
	}
}

func TestErrNoteNotFound(t *testing.T) {
	s := newTestStore(t)

	_, err := s.GetNote("nonexistent")
	if !errors.Is(err, store.ErrNoteNotFound) {
		t.Errorf("GetNote error = %v, want ErrNoteNotFound", err)
	}

	_, err = s.UpdateNote("nonexistent", "t", "b")
	if !errors.Is(err, store.ErrNoteNotFound) {
		t.Errorf("UpdateNote error = %v, want ErrNoteNotFound", err)
	}

	err = s.DeleteNote("nonexistent")
	if !errors.Is(err, store.ErrNoteNotFound) {
		t.Errorf("DeleteNote error = %v, want ErrNoteNotFound", err)
	}

	err = s.PinNote("nonexistent")
	if !errors.Is(err, store.ErrNoteNotFound) {
		t.Errorf("PinNote error = %v, want ErrNoteNotFound", err)
	}

	_, err = s.TogglePin("nonexistent")
	if !errors.Is(err, store.ErrNoteNotFound) {
		t.Errorf("TogglePin error = %v, want ErrNoteNotFound", err)
	}
}

func TestSyncRefs(t *testing.T) {
	s := newTestStore(t)

	noteA, _ := s.CreateNote("Note A", "body", nil)
	noteB, _ := s.CreateNote("Note B", "body", nil)
	noteC, _ := s.CreateNote("Note C", "body", nil)

	// Sync refs: noteA references noteB and noteC
	if err := s.SyncRefs(noteA.ID, []string{noteB.ID, noteC.ID}); err != nil {
		t.Fatalf("syncing refs: %v", err)
	}

	// noteA should have ref tags
	got, _ := s.GetNote(noteA.ID)
	refCount := 0
	for _, tag := range got.Tags {
		if tag.Key == "ref" {
			refCount++
		}
	}
	if refCount != 2 {
		t.Errorf("got %d ref tags, want 2", refCount)
	}
}

func TestSyncRefs_Reconcile(t *testing.T) {
	s := newTestStore(t)

	noteA, _ := s.CreateNote("Note A", "body", nil)
	noteB, _ := s.CreateNote("Note B", "body", nil)
	noteC, _ := s.CreateNote("Note C", "body", nil)

	// First sync: A -> B
	if err := s.SyncRefs(noteA.ID, []string{noteB.ID}); err != nil {
		t.Fatalf("first sync: %v", err)
	}

	// Second sync: A -> C (removes B, adds C)
	if err := s.SyncRefs(noteA.ID, []string{noteC.ID}); err != nil {
		t.Fatalf("second sync: %v", err)
	}

	got, _ := s.GetNote(noteA.ID)
	var refs []string
	for _, tag := range got.Tags {
		if tag.Key == "ref" {
			refs = append(refs, tag.Value)
		}
	}
	if len(refs) != 1 || refs[0] != noteC.ID {
		t.Errorf("expected ref to %s only, got %v", noteC.ID, refs)
	}
}

func TestSyncRefs_Empty(t *testing.T) {
	s := newTestStore(t)

	noteA, _ := s.CreateNote("Note A", "body", nil)
	noteB, _ := s.CreateNote("Note B", "body", nil)

	// Add a ref, then clear all
	_ = s.SyncRefs(noteA.ID, []string{noteB.ID})
	if err := s.SyncRefs(noteA.ID, nil); err != nil {
		t.Fatalf("clearing refs: %v", err)
	}

	got, _ := s.GetNote(noteA.ID)
	for _, tag := range got.Tags {
		if tag.Key == "ref" {
			t.Error("expected no ref tags after clearing")
		}
	}
}

func TestSyncRefs_PreservesOtherTags(t *testing.T) {
	s := newTestStore(t)

	noteA, _ := s.CreateNote("Note A", "body", []model.Tag{{Key: "folder", Value: "work"}})
	noteB, _ := s.CreateNote("Note B", "body", nil)

	if err := s.SyncRefs(noteA.ID, []string{noteB.ID}); err != nil {
		t.Fatalf("syncing refs: %v", err)
	}

	got, _ := s.GetNote(noteA.ID)
	hasFolderTag := false
	hasRefTag := false
	for _, tag := range got.Tags {
		if tag.Key == "folder" && tag.Value == "work" {
			hasFolderTag = true
		}
		if tag.Key == "ref" {
			hasRefTag = true
		}
	}
	if !hasFolderTag {
		t.Error("folder tag was removed during ref sync")
	}
	if !hasRefTag {
		t.Error("ref tag was not added")
	}
}

func TestReferencesTo(t *testing.T) {
	s := newTestStore(t)

	noteA, _ := s.CreateNote("Note A", "body", nil)
	noteB, _ := s.CreateNote("Note B", "body", nil)
	noteC, _ := s.CreateNote("Note C", "body", nil)

	// A and C both reference B
	_ = s.SyncRefs(noteA.ID, []string{noteB.ID})
	_ = s.SyncRefs(noteC.ID, []string{noteB.ID})

	refs, err := s.ReferencesTo(noteB.ID)
	if err != nil {
		t.Fatalf("getting references: %v", err)
	}
	if len(refs) != 2 {
		t.Errorf("got %d references, want 2", len(refs))
	}
}

func TestReferencesTo_NoneFound(t *testing.T) {
	s := newTestStore(t)

	note, _ := s.CreateNote("Lonely Note", "body", nil)

	refs, err := s.ReferencesTo(note.ID)
	if err != nil {
		t.Fatalf("getting references: %v", err)
	}
	if len(refs) != 0 {
		t.Errorf("got %d references, want 0", len(refs))
	}
}

func TestReferencesTo_DeletedSourceCleansUp(t *testing.T) {
	s := newTestStore(t)

	noteA, _ := s.CreateNote("Note A", "body", nil)
	noteB, _ := s.CreateNote("Note B", "body", nil)

	_ = s.SyncRefs(noteA.ID, []string{noteB.ID})

	// Delete A — cascade should remove the ref tag
	_ = s.DeleteNote(noteA.ID)

	refs, err := s.ReferencesTo(noteB.ID)
	if err != nil {
		t.Fatalf("getting references: %v", err)
	}
	if len(refs) != 0 {
		t.Errorf("got %d references after source deleted, want 0", len(refs))
	}
}

func TestListNotes_SortAsc(t *testing.T) {
	s := newTestStore(t)

	first, _ := s.CreateNote("First", "body", nil)
	time.Sleep(1100 * time.Millisecond) // ensure different created_at second
	second, _ := s.CreateNote("Second", "body", nil)

	// Default: newest first (DESC)
	notes, err := s.ListNotes(model.NoteFilter{})
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if len(notes) != 2 {
		t.Fatalf("got %d notes, want 2", len(notes))
	}
	if notes[0].ID != second.ID {
		t.Errorf("default order: first note = %q, want %q", notes[0].Title, "Second")
	}

	// SortAsc: oldest first
	notes, err = s.ListNotes(model.NoteFilter{SortAsc: true})
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if notes[0].ID != first.ID {
		t.Errorf("asc order: first note = %q, want %q", notes[0].Title, "First")
	}
}

func TestNoteExistsByContent(t *testing.T) {
	s := newTestStore(t)

	_, _ = s.CreateNote("Unique Title", "Unique body", nil)

	exists, err := s.NoteExistsByContent("Unique Title", "Unique body")
	if err != nil {
		t.Fatalf("checking existence: %v", err)
	}
	if !exists {
		t.Error("expected note to exist")
	}

	exists, err = s.NoteExistsByContent("Different Title", "Different body")
	if err != nil {
		t.Fatalf("checking existence: %v", err)
	}
	if exists {
		t.Error("expected note not to exist")
	}

	// Same title, different body — should not match
	exists, err = s.NoteExistsByContent("Unique Title", "Different body")
	if err != nil {
		t.Fatalf("checking existence: %v", err)
	}
	if exists {
		t.Error("expected no match with same title but different body")
	}
}
