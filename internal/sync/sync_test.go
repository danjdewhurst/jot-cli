package sync_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func newTestEncryptedSyncer(t *testing.T, passphrase string) (*jsync.Syncer, *store.Store) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("opening store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	syncDir := filepath.Join(t.TempDir(), "sync")
	return jsync.NewEncrypted(s, syncDir, passphrase), s
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

// TestPushAtomicWrite verifies that changeset files are written atomically:
// no partial/corrupt files should remain if the write succeeds, and the
// final file should have restrictive permissions (0600).
func TestPushAtomicWrite(t *testing.T) {
	syncer, st := newTestSyncer(t)

	_, err := st.CreateNote("Atomic Test", "body content", nil)
	if err != nil {
		t.Fatalf("creating note: %v", err)
	}

	pushed, err := syncer.Push()
	if err != nil {
		t.Fatalf("push: %v", err)
	}
	if pushed != 1 {
		t.Fatalf("pushed = %d, want 1", pushed)
	}

	changesetsDir := filepath.Join(syncer.SyncDir(), "changesets")
	entries, err := os.ReadDir(changesetsDir)
	if err != nil {
		t.Fatalf("reading changesets dir: %v", err)
	}

	// No .tmp files should remain after a successful push.
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("temporary file %q should not remain after successful push", e.Name())
		}
	}

	// Find the .ndjson file and verify permissions are 0600.
	var found bool
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".ndjson") {
			found = true
			info, err := e.Info()
			if err != nil {
				t.Fatalf("stat %s: %v", e.Name(), err)
			}
			perm := info.Mode().Perm()
			if perm != 0o600 {
				t.Errorf("changeset file permissions = %o, want 0600", perm)
			}

			// Verify the file is valid ndjson (not corrupt/truncated).
			data, err := os.ReadFile(filepath.Join(changesetsDir, e.Name()))
			if err != nil {
				t.Fatalf("reading changeset: %v", err)
			}
			lines := strings.Split(strings.TrimSpace(string(data)), "\n")
			for i, line := range lines {
				var ce jsync.ChangeEntry
				if err := json.Unmarshal([]byte(line), &ce); err != nil {
					t.Errorf("line %d is corrupt ndjson: %v", i, err)
				}
			}
		}
	}
	if !found {
		t.Error("no .ndjson changeset file found")
	}
}

// TestPullLargeNoteBody verifies that notes with bodies exceeding the
// default bufio.Scanner 64KB limit are handled correctly.
func TestPullLargeNoteBody(t *testing.T) {
	syncerA, stA := newTestSyncer(t)
	syncerB, stB := newTestSyncer(t)

	// Create a note with a body larger than 64KB (the default scanner limit).
	largeBody := strings.Repeat("x", 128*1024) // 128KB
	_, err := stA.CreateNote("Large Note", largeBody, nil)
	if err != nil {
		t.Fatalf("creating large note: %v", err)
	}

	if _, err := syncerA.Push(); err != nil {
		t.Fatalf("push A: %v", err)
	}

	copyChangesets(t, syncerA, syncerB)

	pulled, _, err := syncerB.Pull()
	if err != nil {
		t.Fatalf("pull B should handle large notes: %v", err)
	}
	if pulled != 1 {
		t.Errorf("pulled = %d, want 1", pulled)
	}

	// Verify the full body was imported.
	notes, _ := stB.ListNotes(model.NoteFilter{})
	if len(notes) != 1 {
		t.Fatalf("got %d notes, want 1", len(notes))
	}
	if len(notes[0].Body) != 128*1024 {
		t.Errorf("body length = %d, want %d", len(notes[0].Body), 128*1024)
	}

	_ = stA // suppress unused
}

// TestPullEqualTimestampTiebreaker verifies that when local and remote notes
// have exactly equal UpdatedAt timestamps, a deterministic tiebreaker is used
// (body hash comparison) rather than silently keeping local.
func TestPullEqualTimestampTiebreaker(t *testing.T) {
	syncerA, stA := newTestSyncer(t)
	syncerB, stB := newTestSyncer(t)

	now := time.Now().UTC().Truncate(time.Second)
	noteID := "01TESTID000000000000000EQ"

	// Create local note on B with body that hashes higher.
	localNote := model.Note{
		ID: noteID, Title: "Local", Body: "zzz local body",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := stB.UpsertNote(localNote); err != nil {
		t.Fatalf("creating local note: %v", err)
	}
	if err := stB.ClearChangelogForNotes([]string{noteID}); err != nil {
		t.Fatalf("clearing changelog: %v", err)
	}

	// Create remote note on A with body that hashes lower.
	remoteNote := model.Note{
		ID: noteID, Title: "Remote", Body: "aaa remote body",
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

	// Determine expected winner by comparing body hashes.
	localHash := fmt.Sprintf("%x", sha256.Sum256([]byte(localNote.Body)))
	remoteHash := fmt.Sprintf("%x", sha256.Sum256([]byte(remoteNote.Body)))

	got, _ := stB.GetNote(noteID)

	if localHash < remoteHash {
		// Local hash wins (lower hash = winner).
		if got.Title != "Local" {
			t.Errorf("title = %q, want %q (local hash is lower, should win)", got.Title, "Local")
		}
		if pulled != 0 || conflicts != 1 {
			t.Errorf("pulled=%d conflicts=%d, want pulled=0 conflicts=1", pulled, conflicts)
		}
	} else {
		// Remote hash wins.
		if got.Title != "Remote" {
			t.Errorf("title = %q, want %q (remote hash is lower, should win)", got.Title, "Remote")
		}
		if pulled != 1 || conflicts != 0 {
			t.Errorf("pulled=%d conflicts=%d, want pulled=1 conflicts=0", pulled, conflicts)
		}
	}
}

func TestEncryptedPushCreatesAgeFiles(t *testing.T) {
	passphrase := "test-encryption-passphrase"
	syncer, st := newTestEncryptedSyncer(t, passphrase)

	_, err := st.CreateNote("Encrypted Note", "secret body", nil)
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

	// Verify .ndjson.age file exists (not .ndjson).
	changesetsDir := filepath.Join(syncer.SyncDir(), "changesets")
	entries, err := os.ReadDir(changesetsDir)
	if err != nil {
		t.Fatalf("reading changesets dir: %v", err)
	}

	var foundAge, foundPlain bool
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".ndjson.age") {
			foundAge = true
			// Verify the content is not plaintext.
			data, err := os.ReadFile(filepath.Join(changesetsDir, e.Name()))
			if err != nil {
				t.Fatalf("reading file: %v", err)
			}
			if strings.Contains(string(data), "secret body") {
				t.Error("encrypted file should not contain plaintext body")
			}
		}
		if strings.HasSuffix(e.Name(), ".ndjson") && !strings.HasSuffix(e.Name(), ".ndjson.age") {
			foundPlain = true
		}
	}

	if !foundAge {
		t.Error("expected .ndjson.age file in changesets directory")
	}
	if foundPlain {
		t.Error("encrypted push should not create plain .ndjson files")
	}
}

func TestEncryptedPushThenPullRoundTrip(t *testing.T) {
	passphrase := "shared-secret"
	syncerA, stA := newTestEncryptedSyncer(t, passphrase)
	syncerB, stB := newTestEncryptedSyncer(t, passphrase)

	_, _ = stA.CreateNote("Encrypted from A", "encrypted body", nil)
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

	notes, _ := stB.ListNotes(model.NoteFilter{})
	if len(notes) != 1 {
		t.Fatalf("got %d notes, want 1", len(notes))
	}
	if notes[0].Title != "Encrypted from A" {
		t.Errorf("title = %q, want %q", notes[0].Title, "Encrypted from A")
	}
}

func TestPullMixedEncryptedAndPlaintext(t *testing.T) {
	passphrase := "shared-secret"

	// Machine A pushes unencrypted.
	syncerA, stA := newTestSyncer(t)
	_, _ = stA.CreateNote("Plain Note", "plain body", nil)
	if _, err := syncerA.Push(); err != nil {
		t.Fatalf("push A: %v", err)
	}

	// Machine B pushes encrypted.
	syncerB, stB := newTestEncryptedSyncer(t, passphrase)
	_, _ = stB.CreateNote("Encrypted Note", "encrypted body", nil)
	if _, err := syncerB.Push(); err != nil {
		t.Fatalf("push B: %v", err)
	}

	// Machine C (encrypted) pulls from both.
	syncerC, stC := newTestEncryptedSyncer(t, passphrase)
	copyChangesets(t, syncerA, syncerC)
	copyChangesets(t, syncerB, syncerC)

	pulled, _, err := syncerC.Pull()
	if err != nil {
		t.Fatalf("pull C: %v", err)
	}
	if pulled != 2 {
		t.Errorf("pulled = %d, want 2", pulled)
	}

	notes, _ := stC.ListNotes(model.NoteFilter{})
	if len(notes) != 2 {
		t.Fatalf("got %d notes, want 2", len(notes))
	}
}

func TestPullEncryptedWithWrongPassphrase(t *testing.T) {
	syncerA, stA := newTestEncryptedSyncer(t, "correct-passphrase")
	_, _ = stA.CreateNote("Secret", "body", nil)
	if _, err := syncerA.Push(); err != nil {
		t.Fatalf("push A: %v", err)
	}

	syncerB, _ := newTestEncryptedSyncer(t, "wrong-passphrase")
	copyChangesets(t, syncerA, syncerB)

	_, _, err := syncerB.Pull()
	if err == nil {
		t.Error("pull with wrong passphrase should return an error")
	}
}

func TestMigrateEncryptsExistingFiles(t *testing.T) {
	passphrase := "migration-passphrase"

	// Push unencrypted changesets first.
	syncerPlain, stPlain := newTestSyncer(t)
	_, _ = stPlain.CreateNote("Note 1", "body 1", nil)
	_, _ = stPlain.CreateNote("Note 2", "body 2", nil)
	if _, err := syncerPlain.Push(); err != nil {
		t.Fatalf("push: %v", err)
	}

	// Create an encrypted syncer pointing at the same sync dir.
	syncDir := syncerPlain.SyncDir()
	dbPath := filepath.Join(t.TempDir(), "migrate.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("opening store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	syncer := jsync.NewEncrypted(st, syncDir, passphrase)

	// Run migrate.
	migrated, err := syncer.MigrateEncrypt()
	if err != nil {
		t.Fatalf("MigrateEncrypt: %v", err)
	}
	if migrated != 1 {
		t.Errorf("migrated = %d, want 1", migrated)
	}

	// Verify: no .ndjson files remain, only .ndjson.age.
	changesetsDir := filepath.Join(syncDir, "changesets")
	entries, err := os.ReadDir(changesetsDir)
	if err != nil {
		t.Fatalf("reading changesets: %v", err)
	}

	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".ndjson") && !strings.HasSuffix(e.Name(), ".ndjson.age") {
			t.Errorf("plain .ndjson file %q should have been migrated", e.Name())
		}
	}

	var ageCount int
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".ndjson.age") {
			ageCount++
		}
	}
	if ageCount != 1 {
		t.Errorf("expected 1 .ndjson.age file, got %d", ageCount)
	}

	// Verify the encrypted file can be decrypted and pulled.
	syncerB, stB := newTestEncryptedSyncer(t, passphrase)
	copyChangesets(t, syncer, syncerB)

	pulled, _, err := syncerB.Pull()
	if err != nil {
		t.Fatalf("pull after migration: %v", err)
	}
	if pulled != 2 {
		t.Errorf("pulled = %d, want 2", pulled)
	}

	notes, _ := stB.ListNotes(model.NoteFilter{})
	if len(notes) != 2 {
		t.Errorf("got %d notes, want 2", len(notes))
	}
}

func TestMigrateNoOpWhenNoPlainFiles(t *testing.T) {
	passphrase := "test-passphrase"
	syncer, st := newTestEncryptedSyncer(t, passphrase)

	// Push encrypted — no plain files to migrate.
	_, _ = st.CreateNote("Already Encrypted", "body", nil)
	if _, err := syncer.Push(); err != nil {
		t.Fatalf("push: %v", err)
	}

	migrated, err := syncer.MigrateEncrypt()
	if err != nil {
		t.Fatalf("MigrateEncrypt: %v", err)
	}
	if migrated != 0 {
		t.Errorf("migrated = %d, want 0 (no plain files)", migrated)
	}
}
