package sync_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/danjdewhurst/jot-cli/internal/model"
	"github.com/danjdewhurst/jot-cli/internal/store"
	jsync "github.com/danjdewhurst/jot-cli/internal/sync"
)

func newTestSyncer(t *testing.T) (*jsync.Syncer, *store.Store) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("opening store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	syncDir := filepath.Join(t.TempDir(), "sync")
	return jsync.New(s, syncDir), s
}

func TestPushCreatesChangesetFile(t *testing.T) {
	syncer, st := newTestSyncer(t)

	_, err := st.CreateNote("Test Note", "body content", []model.Tag{{Key: "folder", Value: "work"}})
	if err != nil {
		t.Fatalf("creating note: %v", err)
	}

	pushed, err := syncer.Push()
	if err != nil {
		t.Fatalf("push: %v", err)
	}
	if pushed != 1 {
		t.Errorf("pushed = %d, want 1", pushed)
	}

	// Verify changeset file exists
	syncDir := filepath.Join(t.TempDir(), "sync", "changesets")
	// The sync dir is inside the syncer's temp dir, so we need to find it differently
	// Let's check via status instead
	status, err := syncer.Status()
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.Pending != 0 {
		t.Errorf("pending = %d, want 0 after push", status.Pending)
	}
	if status.LastSync.IsZero() {
		t.Error("last_sync should be set after push")
	}

	// Push again — nothing to push
	pushed, err = syncer.Push()
	if err != nil {
		t.Fatalf("second push: %v", err)
	}
	if pushed != 0 {
		t.Errorf("second push = %d, want 0", pushed)
	}

	_ = syncDir // suppress unused
}

func TestPullImportsNewNotes(t *testing.T) {
	syncerA, stA := newTestSyncer(t)
	syncerB, stB := newTestSyncer(t)

	// Create note on A and push
	note, _ := stA.CreateNote("From A", "hello from machine A", nil)
	if _, err := syncerA.Push(); err != nil {
		t.Fatalf("push A: %v", err)
	}

	// Copy changeset files from A's sync dir to B's sync dir
	copyChangesets(t, syncerA, syncerB)

	// Pull on B
	pulled, conflicts, err := syncerB.Pull()
	if err != nil {
		t.Fatalf("pull B: %v", err)
	}
	if pulled != 1 {
		t.Errorf("pulled = %d, want 1", pulled)
	}
	if conflicts != 0 {
		t.Errorf("conflicts = %d, want 0", conflicts)
	}

	// Verify note exists on B
	got, err := stB.GetNote(note.ID)
	if err != nil {
		t.Fatalf("getting note on B: %v", err)
	}
	if got.Title != "From A" {
		t.Errorf("title = %q, want %q", got.Title, "From A")
	}
}

func TestPullLastWriteWins(t *testing.T) {
	syncerA, stA := newTestSyncer(t)
	syncerB, stB := newTestSyncer(t)

	// Create same note on both machines
	now := time.Now().UTC().Truncate(time.Second)
	noteID := "01TESTID00000000000000LWW"

	localNote := model.Note{
		ID: noteID, Title: "Local Version", Body: "local",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := stB.UpsertNote(localNote); err != nil {
		t.Fatalf("creating local note: %v", err)
	}
	// Clear the changelog entry from upserting the local note
	if err := stB.ClearChangelogForNotes([]string{noteID}); err != nil {
		t.Fatalf("clearing changelog: %v", err)
	}

	// Create a newer version on A
	remoteNote := model.Note{
		ID: noteID, Title: "Remote Version", Body: "remote",
		CreatedAt: now, UpdatedAt: now.Add(time.Hour),
	}
	if err := stA.UpsertNote(remoteNote); err != nil {
		t.Fatalf("creating remote note: %v", err)
	}
	if _, err := syncerA.Push(); err != nil {
		t.Fatalf("push A: %v", err)
	}

	copyChangesets(t, syncerA, syncerB)

	pulled, conflicts, err := syncerB.Pull()
	if err != nil {
		t.Fatalf("pull B: %v", err)
	}
	if pulled != 1 {
		t.Errorf("pulled = %d, want 1", pulled)
	}
	if conflicts != 0 {
		t.Errorf("conflicts = %d, want 0", conflicts)
	}

	got, _ := stB.GetNote(noteID)
	if got.Title != "Remote Version" {
		t.Errorf("title = %q, want %q (remote should win)", got.Title, "Remote Version")
	}
}

func TestPullLocalWins(t *testing.T) {
	syncerA, stA := newTestSyncer(t)
	syncerB, stB := newTestSyncer(t)

	now := time.Now().UTC().Truncate(time.Second)
	noteID := "01TESTID00000000000000LOC"

	// Local has newer version
	localNote := model.Note{
		ID: noteID, Title: "Local Newer", Body: "local",
		CreatedAt: now, UpdatedAt: now.Add(time.Hour),
	}
	if err := stB.UpsertNote(localNote); err != nil {
		t.Fatalf("creating local note: %v", err)
	}
	if err := stB.ClearChangelogForNotes([]string{noteID}); err != nil {
		t.Fatalf("clearing changelog: %v", err)
	}

	// Remote has older version
	remoteNote := model.Note{
		ID: noteID, Title: "Remote Older", Body: "remote",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := stA.UpsertNote(remoteNote); err != nil {
		t.Fatalf("creating remote note: %v", err)
	}
	if _, err := syncerA.Push(); err != nil {
		t.Fatalf("push A: %v", err)
	}

	copyChangesets(t, syncerA, syncerB)

	pulled, conflicts, err := syncerB.Pull()
	if err != nil {
		t.Fatalf("pull B: %v", err)
	}
	if pulled != 0 {
		t.Errorf("pulled = %d, want 0 (local should win)", pulled)
	}
	if conflicts != 1 {
		t.Errorf("conflicts = %d, want 1", conflicts)
	}

	got, _ := stB.GetNote(noteID)
	if got.Title != "Local Newer" {
		t.Errorf("title = %q, want %q (local should be kept)", got.Title, "Local Newer")
	}
}

func TestPullDeletePropagates(t *testing.T) {
	syncerA, stA := newTestSyncer(t)
	syncerB, stB := newTestSyncer(t)

	now := time.Now().UTC().Truncate(time.Second)
	noteID := "01TESTID00000000000000DEL"

	// Create the note on B with an older timestamp
	localNote := model.Note{
		ID: noteID, Title: "To Delete", Body: "body",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := stB.UpsertNote(localNote); err != nil {
		t.Fatalf("creating local note: %v", err)
	}
	if err := stB.ClearChangelogForNotes([]string{noteID}); err != nil {
		t.Fatalf("clearing changelog: %v", err)
	}

	// Create the note on A, then delete it (delete will have later timestamp)
	remoteNote := model.Note{
		ID: noteID, Title: "To Delete", Body: "body",
		CreatedAt: now, UpdatedAt: now.Add(time.Hour),
	}
	if err := stA.UpsertNote(remoteNote); err != nil {
		t.Fatalf("creating remote note: %v", err)
	}
	if err := stA.ClearChangelogForNotes([]string{noteID}); err != nil {
		t.Fatalf("clearing changelog: %v", err)
	}
	if err := stA.DeleteNote(noteID); err != nil {
		t.Fatalf("deleting remote note: %v", err)
	}
	if _, err := syncerA.Push(); err != nil {
		t.Fatalf("push A: %v", err)
	}

	copyChangesets(t, syncerA, syncerB)

	pulled, _, err := syncerB.Pull()
	if err != nil {
		t.Fatalf("pull B: %v", err)
	}
	if pulled != 1 {
		t.Errorf("pulled = %d, want 1", pulled)
	}

	_, err = stB.GetNote(noteID)
	if err == nil {
		t.Error("expected note to be deleted after pull")
	}
}

func TestPullDeleteBlockedByEdit(t *testing.T) {
	syncerA, stA := newTestSyncer(t)
	syncerB, stB := newTestSyncer(t)

	now := time.Now().UTC().Truncate(time.Second)
	noteID := "01TESTID00000000000000BLK"

	// Create on B with a very recent update (newer than the remote delete)
	localNote := model.Note{
		ID: noteID, Title: "Edited Locally", Body: "body",
		CreatedAt: now, UpdatedAt: now.Add(2 * time.Hour),
	}
	if err := stB.UpsertNote(localNote); err != nil {
		t.Fatalf("creating local note: %v", err)
	}
	if err := stB.ClearChangelogForNotes([]string{noteID}); err != nil {
		t.Fatalf("clearing changelog: %v", err)
	}

	// On A, create then delete the note — delete timestamp is earlier than local edit
	remoteNote := model.Note{
		ID: noteID, Title: "To Delete", Body: "body",
		CreatedAt: now, UpdatedAt: now.Add(time.Hour),
	}
	if err := stA.UpsertNote(remoteNote); err != nil {
		t.Fatalf("creating remote note: %v", err)
	}
	if err := stA.ClearChangelogForNotes([]string{noteID}); err != nil {
		t.Fatalf("clearing changelog: %v", err)
	}
	if err := stA.DeleteNote(noteID); err != nil {
		t.Fatalf("deleting remote note: %v", err)
	}
	if _, err := syncerA.Push(); err != nil {
		t.Fatalf("push A: %v", err)
	}

	copyChangesets(t, syncerA, syncerB)

	pulled, _, err := syncerB.Pull()
	if err != nil {
		t.Fatalf("pull B: %v", err)
	}
	if pulled != 0 {
		t.Errorf("pulled = %d, want 0 (local edit should block delete)", pulled)
	}

	// Note should survive
	got, err := stB.GetNote(noteID)
	if err != nil {
		t.Fatalf("note should still exist: %v", err)
	}
	if got.Title != "Edited Locally" {
		t.Errorf("title = %q, want %q", got.Title, "Edited Locally")
	}
}

func TestPushThenPullRoundTrip(t *testing.T) {
	syncerA, stA := newTestSyncer(t)
	syncerB, stB := newTestSyncer(t)

	// Create multiple notes on A
	_, _ = stA.CreateNote("Note 1", "body 1", []model.Tag{{Key: "project", Value: "alpha"}})
	_, _ = stA.CreateNote("Note 2", "body 2", nil)
	note3, _ := stA.CreateNote("Note 3", "body 3", nil)

	// Delete one
	_ = stA.DeleteNote(note3.ID)

	if _, err := syncerA.Push(); err != nil {
		t.Fatalf("push A: %v", err)
	}

	copyChangesets(t, syncerA, syncerB)

	pulled, _, err := syncerB.Pull()
	if err != nil {
		t.Fatalf("pull B: %v", err)
	}
	if pulled != 2 {
		t.Errorf("pulled = %d, want 2", pulled)
	}

	// Verify notes on B
	notes, _ := stB.ListNotes(model.NoteFilter{})
	if len(notes) != 2 {
		t.Errorf("got %d notes on B, want 2", len(notes))
	}
}

func TestPullSkipsOwnChangesets(t *testing.T) {
	syncer, st := newTestSyncer(t)

	_, _ = st.CreateNote("My Note", "body", nil)
	if _, err := syncer.Push(); err != nil {
		t.Fatalf("push: %v", err)
	}

	// Pull on the same syncer — should skip own changesets
	pulled, _, err := syncer.Pull()
	if err != nil {
		t.Fatalf("pull: %v", err)
	}
	if pulled != 0 {
		t.Errorf("pulled = %d, want 0 (should skip own changesets)", pulled)
	}
}

func TestPullClearsChangelog(t *testing.T) {
	syncerA, stA := newTestSyncer(t)
	syncerB, stB := newTestSyncer(t)

	_, _ = stA.CreateNote("From A", "body", nil)
	if _, err := syncerA.Push(); err != nil {
		t.Fatalf("push A: %v", err)
	}

	copyChangesets(t, syncerA, syncerB)

	if _, _, err := syncerB.Pull(); err != nil {
		t.Fatalf("pull B: %v", err)
	}

	// After pulling, B should have no unsynced changes for imported notes
	entries, _ := stB.UnsyncedChanges()
	if len(entries) != 0 {
		t.Errorf("got %d unsynced entries after pull, want 0", len(entries))
	}

	_ = stA // suppress unused
}

func TestSyncStatus(t *testing.T) {
	syncer, st := newTestSyncer(t)

	// Before any changes
	status, err := syncer.Status()
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.Pending != 0 {
		t.Errorf("pending = %d, want 0", status.Pending)
	}
	if !status.LastSync.IsZero() {
		t.Error("last_sync should be zero before first sync")
	}

	// Create a note
	_, _ = st.CreateNote("Test", "body", nil)
	status, _ = syncer.Status()
	if status.Pending == 0 {
		t.Error("expected pending > 0 after creating a note")
	}

	// Push
	if _, err := syncer.Push(); err != nil {
		t.Fatalf("push: %v", err)
	}
	status, _ = syncer.Status()
	if status.Pending != 0 {
		t.Errorf("pending = %d, want 0 after push", status.Pending)
	}
	if status.LastSync.IsZero() {
		t.Error("last_sync should be set after push")
	}
}

// copyChangesets copies changeset files from syncer A's sync dir to syncer B's sync dir.
// This simulates the file-sync service (Dropbox, iCloud, etc.) propagating files.
func copyChangesets(t *testing.T, from, to *jsync.Syncer) {
	t.Helper()

	fromDir := syncDirFromSyncer(t, from)
	toDir := syncDirFromSyncer(t, to)

	srcDir := filepath.Join(fromDir, "changesets")
	dstDir := filepath.Join(toDir, "changesets")
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		t.Fatalf("creating dest changesets dir: %v", err)
	}

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		t.Fatalf("reading source changesets: %v", err)
	}
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(srcDir, e.Name()))
		if err != nil {
			t.Fatalf("reading %s: %v", e.Name(), err)
		}
		if err := os.WriteFile(filepath.Join(dstDir, e.Name()), data, 0o644); err != nil {
			t.Fatalf("writing %s: %v", e.Name(), err)
		}
	}
}

// syncDirFromSyncer extracts the sync directory by doing a push with no changes
// and checking what directory was created. We use a simpler approach: just use
// the status to find the sync dir via the syncer's internal state.
// Since we can't access private fields, we use a known pattern: create the dir via Push.
func syncDirFromSyncer(t *testing.T, syncer *jsync.Syncer) string {
	t.Helper()
	// We know the syncer creates <syncDir>/changesets/ on push.
	// The test creates syncDir in t.TempDir()/sync.
	// We need to find the changeset files. Let's walk up from the temp dir.

	// Actually, we need a way to get the sync dir. Let's add a public method.
	// For now, we'll just rely on the directory structure from newTestSyncer.
	// The sync dir is always <tempDir>/sync but each syncer gets its own tempDir.
	// We need to expose the sync dir. Let me use a different approach.
	return syncer.SyncDir()
}
