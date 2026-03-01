# Plan: Note history

**Priority:** Medium effort, high value

## Summary

Track note versions on each edit. Allow viewing history and reverting to previous versions.

## Commands

- `jot history <id>` — List versions with timestamps and diffs
- `jot history <id> --version <n>` — Show a specific version
- `jot revert <id> --version <n>` — Revert note to a previous version
- All support `--json`

## Schema

New migration (005):

```sql
CREATE TABLE note_versions (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    note_id    TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    title      TEXT NOT NULL,
    body       TEXT NOT NULL,
    created_at TEXT NOT NULL,
    version    INTEGER NOT NULL,
    UNIQUE(note_id, version)
);
```

## Behaviour

1. **On update**: before overwriting, snapshot current title/body into `note_versions` with incrementing version number
2. **History list**: show version number, timestamp, and a short diff summary (lines added/removed)
3. **Revert**: create a new version (not destructive) that copies content from the target version

## Implementation

### Store layer

- `SaveVersion(noteID)` — Snapshot current state before update
- `ListVersions(noteID)` → `[]NoteVersion`
- `GetVersion(noteID, version)` → `NoteVersion`
- Modify `UpdateNote()` to call `SaveVersion()` first

### Model

- `NoteVersion` struct: id, note_id, title, body, created_at, version

### CLI

- `cmd/history.go` — History list and single-version display
- `cmd/revert.go` — Revert to version (calls `UpdateNote` with old content, which itself creates a new version)

### Diff

- Simple line-based diff for display (use `internal/diff/` or inline)
- Show `+3 / -1` summary in list, full diff with `--diff` flag

## Complexity

~400 lines. New migration, new store methods, two new commands, simple diff logic.
