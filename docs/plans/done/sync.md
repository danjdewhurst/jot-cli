# Sync Feature — Implementation Plan

## Overview

Add a `jot sync` command that synchronises notes between machines using a shared directory (e.g. Dropbox, Syncthing, iCloud Drive, or a mounted network share). The approach uses a **file-based sync directory** containing NDJSON (newline-delimited JSON) changeset files. Each machine dumps its changes since the last sync, then ingests changesets from other machines.

This avoids running a server, avoids git complexity for non-developer users, and keeps the design simple enough for a CLI notes app.

## Design Decisions

### Why file-based sync directory (not git, not dump/restore)?

| Approach | Pros | Cons |
|---|---|---|
| **Git-backed** | Built-in history, merging | Requires git on every machine; heavyweight for notes; merge conflicts in JSON are painful |
| **Dump/restore** | Dead simple | Destructive — full overwrite loses edits on the other machine; no incremental sync |
| **File-based sync dir** | Incremental; no server; works with any file-sync service; simple to reason about | Needs change tracking in the schema |

The file-based approach is the sweet spot: incremental, non-destructive, and works with tools people already use for file synchronisation.

### Conflict resolution: last-write-wins by `updated_at`

Notes are identified by their ULID primary key, which is globally unique (no coordination needed). When two machines edit the same note, the version with the later `updated_at` wins. This is acceptable for a personal notes app — the user is unlikely to be editing the same note on two machines simultaneously, and if they do, the most recent edit is almost certainly the one they want.

Deletes are tracked as tombstones so they propagate across machines.

---

## 1. Schema Changes

### New migration: `internal/store/migrations/002_sync.sql`

```sql
-- Track which notes have changed since last sync.
-- Triggers capture changes automatically so existing Go code is unaffected.

CREATE TABLE IF NOT EXISTS sync_meta (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- key: 'machine_id' => ULID generated on first sync
-- key: 'last_sync'  => RFC 3339 timestamp of last successful sync

CREATE TABLE IF NOT EXISTS sync_changelog (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    note_id    TEXT NOT NULL,
    action     TEXT NOT NULL CHECK (action IN ('upsert', 'delete')),
    changed_at TEXT NOT NULL,
    synced     INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_sync_changelog_synced
    ON sync_changelog(synced, id);

-- Trigger: after INSERT on notes, log an upsert
CREATE TRIGGER IF NOT EXISTS trg_sync_note_insert
AFTER INSERT ON notes
BEGIN
    INSERT INTO sync_changelog (note_id, action, changed_at)
    VALUES (NEW.id, 'upsert', NEW.updated_at);
END;

-- Trigger: after UPDATE on notes, log an upsert
CREATE TRIGGER IF NOT EXISTS trg_sync_note_update
AFTER UPDATE ON notes
BEGIN
    INSERT INTO sync_changelog (note_id, action, changed_at)
    VALUES (NEW.id, 'upsert', NEW.updated_at);
END;

-- Trigger: after DELETE on notes, log a delete
CREATE TRIGGER IF NOT EXISTS trg_sync_note_delete
AFTER DELETE ON notes
BEGIN
    INSERT INTO sync_changelog (note_id, action, changed_at)
    VALUES (OLD.id, 'delete', OLD.updated_at);
END;

-- Trigger: after INSERT on tags, log an upsert for the parent note
CREATE TRIGGER IF NOT EXISTS trg_sync_tag_insert
AFTER INSERT ON tags
BEGIN
    INSERT INTO sync_changelog (note_id, action, changed_at)
    VALUES (NEW.note_id, 'upsert',
        (SELECT updated_at FROM notes WHERE id = NEW.note_id));
END;

-- Trigger: after DELETE on tags, log an upsert for the parent note
CREATE TRIGGER IF NOT EXISTS trg_sync_tag_delete
AFTER DELETE ON tags
BEGIN
    INSERT INTO sync_changelog (note_id, action, changed_at)
    VALUES (OLD.note_id, 'upsert',
        (SELECT updated_at FROM notes WHERE id = OLD.note_id));
END;
```

### Migration runner refactor

The existing `Store.migrate()` reads only `001_initial.sql`. It needs to support multiple ordered migrations using `PRAGMA user_version`:

```go
func (s *Store) migrate() error {
    entries, err := migrations.ReadDir("migrations")
    // sort by filename
    // read PRAGMA user_version
    // run each migration whose number exceeds current version
    // set PRAGMA user_version after each
}
```

This must land before anything else in this plan.

---

## 2. Sync Directory Layout

The sync directory defaults to `$XDG_DATA_HOME/jot/sync/` but is configurable via `--sync-dir` flag or `JOT_SYNC_DIR` environment variable.

```
<sync-dir>/
  changesets/
    <machine_id>_<timestamp>.ndjson   # one file per sync push
  .lock                                # advisory lock file
```

Each `.ndjson` file contains one JSON object per line:

```json
{"action":"upsert","note":{"id":"01J...","title":"...","body":"...","created_at":"...","updated_at":"...","archived":false,"tags":[{"key":"folder","value":"work"}]}}
{"action":"delete","note_id":"01J...","deleted_at":"2026-03-01T12:00:00Z"}
```

---

## 3. New Package: `internal/sync`

### `internal/sync/changeset.go`

```go
package sync

import "github.com/danjdewhurst/jot-cli/internal/model"

// ChangeEntry represents a single change in a changeset file.
type ChangeEntry struct {
    Action    string      `json:"action"`              // "upsert" or "delete"
    Note      *model.Note `json:"note,omitempty"`      // full note for upserts
    NoteID    string      `json:"note_id,omitempty"`   // just the ID for deletes
    DeletedAt string      `json:"deleted_at,omitempty"` // timestamp for deletes
}
```

### `internal/sync/sync.go`

```go
package sync

import (
    "time"
    "github.com/danjdewhurst/jot-cli/internal/store"
)

// SyncResult summarises what happened during a sync operation.
type SyncResult struct {
    Pushed    int       `json:"pushed"`
    Pulled    int       `json:"pulled"`
    Conflicts int       `json:"conflicts"`
    SyncedAt  time.Time `json:"synced_at"`
}

// Syncer coordinates push/pull operations against a sync directory.
type Syncer struct {
    store   *store.Store
    syncDir string
}

func New(s *store.Store, syncDir string) *Syncer

// Sync performs a full push-then-pull cycle.
func (s *Syncer) Sync() (SyncResult, error)

// Push exports unsynced changes to the sync directory.
func (s *Syncer) Push() (int, error)

// Pull imports changesets from other machines.
// Returns (pulled, conflicts, err).
func (s *Syncer) Pull() (int, int, error)

// Status returns a summary of pending changes.
func (s *Syncer) Status() (pending int, lastSync time.Time, err error)
```

### `internal/sync/push.go`

The push operation:

1. Query `sync_changelog` for rows where `synced = 0`.
2. For each `upsert` entry, fetch the full note (with tags) from the store.
3. For each `delete` entry, emit a delete record with the note ID.
4. Deduplicate: if the same `note_id` appears multiple times, keep only the latest entry.
5. Write all entries to a new `.ndjson` file named `<machine_id>_<RFC3339 timestamp>.ndjson`.
6. Mark the changelog rows as `synced = 1`.
7. Update `sync_meta` key `last_sync`.

### `internal/sync/pull.go`

The pull operation:

1. List all `.ndjson` files in `<sync-dir>/changesets/`.
2. Skip files produced by this machine (prefix matches `machine_id`).
3. Skip files already processed (timestamp in filename is before `last_sync`).
4. For each file, read line by line and apply the conflict resolution algorithm (see below).
5. After processing, clean up trigger-generated changelog entries for imported notes to avoid re-exporting them.

---

## 4. Conflict Resolution

```
for each entry in changeset:
    if entry.action == "upsert":
        local = store.GetNote(entry.note.id)
        if local not found:
            store.ImportNote(entry.note)     // new note, just import
        else if entry.note.updated_at > local.updated_at:
            store.ImportNote(entry.note)     // remote is newer, overwrite
        else:
            skip                             // local is newer, keep ours
            conflicts++

    if entry.action == "delete":
        local = store.GetNote(entry.note_id)
        if local not found:
            skip                             // already gone
        else if entry.deleted_at > local.updated_at:
            store.DeleteNote(entry.note_id)  // delete propagates
        else:
            skip                             // note was edited after delete, keep it
```

**Edge case:** if a note is deleted on machine A and edited on machine B, the edit wins (because its `updated_at` will be later than the delete timestamp). This feels correct — if you edited it, you probably want to keep it.

---

## 5. Store Layer Additions

### `internal/store/sync.go` (new file)

```go
package store

// SyncChangelogEntry represents a row in the sync_changelog table.
type SyncChangelogEntry struct {
    ID        int64
    NoteID    string
    Action    string // "upsert" or "delete"
    ChangedAt string
}

// UnsyncedChanges returns changelog entries that haven't been synced yet.
func (s *Store) UnsyncedChanges() ([]SyncChangelogEntry, error)

// MarkChangesSynced marks changelog entries up to a given ID as synced.
func (s *Store) MarkChangesSynced(upToID int64) error

// ClearChangelogForNotes removes unsynced changelog entries for specific note IDs
// (used after pulling to avoid re-exporting imported changes).
func (s *Store) ClearChangelogForNotes(noteIDs []string) error

// GetSyncMeta reads a value from the sync_meta table.
func (s *Store) GetSyncMeta(key string) (string, error)

// SetSyncMeta writes a value to the sync_meta table.
func (s *Store) SetSyncMeta(key, value string) error
```

### `internal/store/notes.go` — add `ImportNote`

```go
// ImportNote inserts or replaces a note with a specific ID, timestamps, and tags.
// Used by sync to import notes from other machines.
func (s *Store) ImportNote(n model.Note) error {
    // Uses INSERT ... ON CONFLICT(id) DO UPDATE SET ...
    // Replaces tags, updates FTS index, all within a transaction.
}
```

**Note:** This is distinct from the export/import plan's `ImportNote` which uses `INSERT OR IGNORE` (skip on conflict). For sync, we need `ON CONFLICT ... DO UPDATE` (overwrite on conflict) since the conflict resolution has already determined the incoming version should win. If both features are implemented, the method signature should be unified — e.g. an `overwrite bool` parameter, or two separate methods (`ImportNote` for skip, `UpsertNote` for overwrite).

---

## 6. Config Changes

Update `internal/config/config.go`:

```go
type Config struct {
    DBPath  string
    SyncDir string
}

func syncDir() string {
    if p := os.Getenv("JOT_SYNC_DIR"); p != "" {
        return p
    }
    return filepath.Join(dataDir(), "sync")
}
```

---

## 7. Machine Identity

On first sync, generate a ULID as the machine identifier and store it in `sync_meta` with key `machine_id`. This is used to name changeset files so a machine skips its own files on pull.

```go
func (s *Syncer) machineID() (string, error) {
    id, err := s.store.GetSyncMeta("machine_id")
    if err == nil && id != "" {
        return id, nil
    }
    id = ulid.Make().String()
    return id, s.store.SetSyncMeta("machine_id", id)
}
```

---

## 8. File Locking

Use an advisory lock file (`<sync-dir>/.lock`) with `syscall.Flock` (via `golang.org/x/sys/unix`) to prevent two jot instances from writing to the sync directory simultaneously. On failure to acquire the lock, wait briefly and retry, then error with a clear message.

For cross-platform support, use `Flock` on darwin/linux. On other platforms, skip locking with a warning.

---

## 9. Handling the Changelog Trigger Loop

When pulling changes, the triggers on the `notes` table will fire and create new `sync_changelog` entries for the imported notes. These must not be re-exported on the next push.

**Solution:** After importing a batch of notes in the pull operation, delete the changelog entries that were just created:

```go
// Inside the pull operation, after all imports:
store.ClearChangelogForNotes(importedNoteIDs)
```

This keeps the schema minimal and the logic contained within the pull operation.

---

## 10. CLI Command: `cmd/sync.go`

```
jot sync                # full push + pull
jot sync status         # show pending changes and last sync time
jot sync push           # push only
jot sync pull           # pull only
jot sync --sync-dir /mnt/nas/jot   # override sync directory
jot sync --json         # machine-readable output
```

```go
var syncCmd = &cobra.Command{
    Use:   "sync",
    Short: "Synchronise notes with a shared directory",
    Long: `Push local changes and pull remote changes from a sync directory.

The sync directory can be any shared folder (Dropbox, Syncthing, iCloud Drive,
a mounted network share, etc.). Set it with --sync-dir or JOT_SYNC_DIR.`,
    RunE: runSync,
}

var syncStatusCmd = &cobra.Command{
    Use:   "status",
    Short: "Show sync status",
    RunE:  runSyncStatus,
}

var syncPushCmd = &cobra.Command{
    Use:   "push",
    Short: "Push local changes to sync directory",
    RunE:  runSyncPush,
}

var syncPullCmd = &cobra.Command{
    Use:   "pull",
    Short: "Pull remote changes from sync directory",
    RunE:  runSyncPull,
}

func init() {
    syncCmd.PersistentFlags().String("sync-dir", "",
        "Sync directory path (default: $JOT_SYNC_DIR or XDG data dir)")
    syncCmd.AddCommand(syncStatusCmd, syncPushCmd, syncPullCmd)
    rootCmd.AddCommand(syncCmd)
}
```

---

## 11. Testing Strategy

### Store tests — `internal/store/store_test.go` additions

| Test | Description |
|------|-------------|
| `TestImportNoteNew` | ImportNote with a new ID creates the note and tags. |
| `TestImportNoteOverwrite` | ImportNote with an existing ID replaces content and tags. |
| `TestSyncChangelog` | Create/update/delete notes, verify changelog entries are created by triggers. |
| `TestMarkChangesSynced` | Verify entries are marked and excluded from `UnsyncedChanges`. |
| `TestSyncMeta` | Get/Set round-trip for sync metadata. |

### Sync package tests — `internal/sync/sync_test.go`

| Test | Description |
|------|-------------|
| `TestPushCreatesChangesetFile` | Create notes, call Push(), verify `.ndjson` file appears with correct content. |
| `TestPullImportsNewNotes` | Place a changeset file in the sync dir, call Pull(), verify notes appear locally. |
| `TestPullLastWriteWins` | Remote note has later `updated_at` — verify remote version wins. |
| `TestPullLocalWins` | Local note has later `updated_at` — verify local version is kept. |
| `TestPullDeletePropagates` | Delete entry with later timestamp — verify note is deleted locally. |
| `TestPullDeleteBlockedByEdit` | Delete entry with older timestamp than local edit — verify note survives. |
| `TestPushThenPullRoundTrip` | Two stores, push from A, pull into B, verify notes match. |
| `TestPullSkipsOwnChangesets` | Push then pull on same machine — verify no self-import. |
| `TestPullClearsChangelog` | After pulling, verify no unsynced changelog entries for imported notes. |
| `TestSyncStatus` | Verify Status() returns correct pending count and last sync time. |

### Test helper

```go
func newTestSyncer(t *testing.T) (*sync.Syncer, *store.Store) {
    t.Helper()
    dbPath := filepath.Join(t.TempDir(), "test.db")
    s, err := store.Open(dbPath)
    // ...
    syncDir := filepath.Join(t.TempDir(), "sync")
    return sync.New(s, syncDir), s
}
```

---

## 12. Implementation Sequence

1. **Refactor migration runner** — Update `Store.migrate()` to support multiple ordered migrations using `PRAGMA user_version`.
2. **Add `002_sync.sql`** — Sync schema (changelog, meta, triggers).
3. **Add store sync methods** — `internal/store/sync.go` with `ImportNote`, `UnsyncedChanges`, `MarkChangesSynced`, etc. Write tests.
4. **Add `internal/sync` package** — Syncer, push, pull, changeset I/O. Write tests.
5. **Add config changes** — `SyncDir` in Config, `JOT_SYNC_DIR` env var.
6. **Add `cmd/sync.go`** — CLI commands.

**Dependency on export/import:** Both plans need an `ImportNote` store method. If export/import lands first, sync can reuse or extend that method. If building sync standalone, the `ON CONFLICT ... DO UPDATE` variant is needed regardless.

---

## 13. Future Considerations (Out of Scope)

- **Encryption at rest** — Changeset files contain plaintext note content. Could add `age` encryption later.
- **Selective sync** — Sync only notes matching certain tags (`--tag` filter).
- **Automatic sync** — Background watcher that syncs on file changes. Too complex for v1.
- **Conflict UI** — `jot sync conflicts` command showing what was overwritten. Needs a conflict log.
- **Changeset compaction** — Over time, the changesets directory grows. A compaction step could merge old files.
