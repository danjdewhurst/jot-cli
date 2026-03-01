# Architecture Overview

This document describes jot-cli's internals for contributors and curious users.

---

## Project layout

```
cmd/                    CLI commands (one file per command, Cobra)
internal/
  model/                Domain types — Note, Tag, NoteFilter, NoteVersion, ExportEnvelope
  store/                SQLite data layer — CRUD, tags, FTS5, sync changelog, versions
  sync/                 File-based sync — push, pull, conflict resolution, encryption
  context/              Environment detection (folder, git repo, branch) via filesystem
  linking/              Note reference extraction (@prefix regex)
  editor/               $EDITOR integration (temp file approach)
  render/               Stateless output formatters — JSON, table, log, detail, history
  config/               XDG path resolution and TOML config loading
  diff/                 Unified diff formatting for version history
  tui/                  Bubbletea TUI app
    views/              View components — list, detail, compose, help
    theme/              Centralised Catppuccin Frappé colour palette
```

---

## Domain model

### Note

```go
type Note struct {
    ID        string    // ULID primary key
    Title     string
    Body      string
    CreatedAt time.Time
    UpdatedAt time.Time
    Archived  bool
    Pinned    bool
    Tags      []Tag     // populated on read, not stored in notes table
}
```

### Tag

```go
type Tag struct {
    Key   string    // e.g. "folder", "git_repo", "project"
    Value string    // e.g. "jot-cli", "danjdewhurst/jot-cli"
}
```

Tags are `key:value` pairs stored in a separate `tags` table with a unique constraint on `(note_id, key, value)`. They are additive and idempotent.

### NoteFilter

```go
type NoteFilter struct {
    Tags       []Tag
    Archived   bool
    PinnedOnly bool
    Limit      int
    Offset     int
    Since      *time.Time
    Until      *time.Time
    SortAsc    bool
}
```

Used by `ListNotes` to build dynamic SQL queries.

### NoteVersion

```go
type NoteVersion struct {
    ID        int
    NoteID    string
    Title     string
    Body      string
    CreatedAt time.Time
    Version   int       // sequential per note
}
```

A snapshot of a note's title and body at a point in time. Created automatically on each `UpdateNote` call.

### ExportEnvelope

```go
type ExportEnvelope struct {
    Version    int       // currently 1
    ExportedAt time.Time
    Count      int
    Notes      []Note
}
```

---

## Database

### Schema

The database uses five migrations:

**001 — Core tables:**

- `notes` — `id TEXT PK`, `title`, `body`, `created_at`, `updated_at`, `archived INTEGER`
- `tags` — `id INTEGER PK`, `note_id FK`, `key`, `value`, `UNIQUE(note_id, key, value)`
- `notes_fts` — FTS5 virtual table with columns `note_id UNINDEXED`, `title`, `body`, `tags`. Tokeniser: `unicode61 remove_diacritics 2`

**002 — Sync:**

- `sync_meta` — key-value store for sync state (`machine_id`, `last_sync`, `encrypt`)
- `sync_changelog` — log of note mutations with `action` (`upsert`/`delete`), `changed_at`, `synced` flag
- Index `idx_sync_changelog_synced` on `sync_changelog(synced, id)`
- Triggers on `notes` and `tags` tables that automatically log changes to `sync_changelog`

**003 — Pinned:**

- Adds `pinned INTEGER NOT NULL DEFAULT 0` column to `notes`

**004 — Indexes:**

- `idx_notes_created_at` on `notes(created_at)`
- `idx_sync_changelog_synced_note_id` on `sync_changelog(synced, note_id)`

**005 — Versions:**

- `note_versions` — `id INTEGER PK`, `note_id FK`, `title`, `body`, `created_at`, `version`, `UNIQUE(note_id, version)`

### Migrations

Migrations are embedded SQL files (`//go:embed migrations/*.sql`) sorted by filename prefix (`001_`, `002_`, etc.). The database tracks its current version via `PRAGMA user_version`. Each migration's SQL is split on semicolons (respecting `BEGIN...END` blocks for triggers) and executed statement-by-statement.

### FTS5 — manual sync pattern

The `notes_fts` table is a **standalone** FTS5 table (not content-synced). This means jot must manually insert, update, and delete FTS rows in the same transaction as note mutations. The `syncFTS` step is a critical invariant — every note write must update both the `notes` table and the `notes_fts` table.

The FTS index includes the note's title, body, and a space-joined string of all tags for that note, enabling search across all three.

### Single-connection pool

```go
db.SetMaxOpenConns(1)
```

SQLite doesn't support concurrent writers. Additionally, per-connection pragmas (`foreign_keys`, `synchronous`, `journal_mode`) must be set on every connection. Limiting to one connection ensures pragmas are always active and avoids "database is locked" errors.

### Pragmas

```sql
PRAGMA journal_mode=WAL;
PRAGMA synchronous=NORMAL;
PRAGMA foreign_keys=ON;
PRAGMA busy_timeout=5000;
```

WAL mode enables concurrent reads while writing. `NORMAL` sync is a good balance of safety and performance.

---

## Store API

Key public methods grouped by concern:

**Notes:**

| Method | Description |
|--------|-------------|
| `CreateNote(title, body, tags)` | Insert note + tags + FTS, returns Note |
| `GetNote(id)` | Fetch by exact ID |
| `UpdateNote(id, title, body)` | Update + create version snapshot + sync FTS |
| `ArchiveNote(id)` | Set `archived = 1` |
| `DeleteNote(id)` | Permanent delete (cascade removes tags, FTS) |
| `ListNotes(filter)` | Dynamic query with tag joins, date filtering, sort |
| `TogglePin(id)` | Flip pinned state, return new value |
| `NoteExistsByContent(title, body)` | Dedup check for Markdown import |

**Bulk:**

| Method | Description |
|--------|-------------|
| `ArchiveNotes(ids)` | Archive multiple notes |
| `DeleteNotes(ids)` | Permanently delete multiple notes |
| `AddTagToNotes(ids, tag)` | Add a tag to multiple notes |
| `PinNotes(ids)` / `UnpinNotes(ids)` | Bulk pin/unpin |

**Tags:**

| Method | Description |
|--------|-------------|
| `AddTag(noteID, tag)` | Add tag (idempotent) |
| `RemoveTag(noteID, tag)` | Remove tag |
| `ListTags(key)` | List distinct tags, optionally filtered by key |

**Search:**

| Method | Description |
|--------|-------------|
| `Search(query, tags)` | FTS5 MATCH + optional tag filter, returns ranked results with snippets |

**References:**

| Method | Description |
|--------|-------------|
| `SyncRefs(noteID, refIDs)` | Reconcile `ref:*` tags for a note — adds new, removes stale |
| `ReferencesTo(noteID)` | Find notes that reference this one (backlinks) |

**Versions:**

| Method | Description |
|--------|-------------|
| `ListVersions(noteID)` | All versions, newest first |
| `GetVersion(noteID, version)` | Specific version by number |

**Sync:**

| Method | Description |
|--------|-------------|
| `UnsyncedChanges()` | Changelog entries where `synced = 0` |
| `MarkChangesSynced(upToID)` | Mark changelog entries as synced |
| `UpsertNote(note)` | Insert or replace (used by sync pull) |
| `ClearChangelogForNotes(ids)` | Remove changelog entries for imported notes |
| `GetSyncMeta(key)` / `SetSyncMeta(key, value)` | Key-value metadata |

**Import:**

| Method | Description |
|--------|-------------|
| `ImportNote(note)` | Insert with `INSERT OR IGNORE` on existing ID |

---

## Context detection

The `internal/context` package detects the user's environment using filesystem reads only — it never shells out to `git` or any other binary.

**Folder:** `os.Getwd()` → `filepath.Base()`

**Git repo:** Walks up from the working directory looking for `.git` (directory or file). If `.git` is a file (worktree), reads the `gitdir:` pointer. Then parses `.git/config` for `[remote "origin"]` URL and extracts `owner/repo`. Falls back to the parent directory name if no remote is found.

**Git branch:** Reads `.git/HEAD`. If it starts with `ref: refs/heads/`, extracts the branch name. Otherwise (detached HEAD), returns the first 8 characters of the commit hash.

URL parsing handles SCP-style SSH (`git@github.com:user/repo.git`) and protocol URLs (`https://`, `ssh://`, `git://`), stripping `.git` suffixes.

---

## Sync subsystem

### Push flow

1. Read unsynced changelog entries from `sync_changelog`
2. Deduplicate: keep only the latest entry per `note_id`
3. For `upsert` entries, fetch the full note (skip if note was deleted since)
4. For `delete` entries, include the `deleted_at` timestamp
5. Write NDJSON to `<sync-dir>/changesets/<machine_id>_<timestamp>.ndjson[.age]`
6. File is written atomically (tmp + fsync + rename)
7. Mark changelog entries as synced

### Pull flow

1. List `.ndjson` and `.ndjson.age` files in `<sync-dir>/changesets/`
2. Skip own machine's files (by machine ID prefix)
3. Skip files older than `last_sync` (timestamp embedded in filename)
4. For each file, decode entries and apply conflict resolution
5. Clear changelog entries created by import triggers (to prevent re-exporting)
6. Update `last_sync` after each successfully processed file (for resumability)

### Encryption layer

Uses `filippo.io/age` with scrypt passphrase encryption. The passphrase is a random 32-byte hex string stored in `sync.key`. Encrypt/decrypt are symmetric — same passphrase for both operations.

### File locking

Uses `flock(2)` via `golang.org/x/sys/unix` for advisory locking on `<sync-dir>/.lock`. Non-blocking (`LOCK_NB`) — fails immediately if another process holds the lock.

---

## Editor integration

The `internal/editor` package uses a temp file approach:

1. Create a temp file (`jot-*.md`)
2. Write initial content (if any)
3. Open the user's editor with the temp file path as argument
4. Wait for the editor to exit
5. Read the temp file contents
6. Clean up the temp file

The editor resolution chain: config `general.editor` → `$VISUAL` → `$EDITOR` → `vi`. The editor string is split on whitespace to support editors with arguments (e.g. `code --wait`).

---

## Render package

Stateless formatters — each function takes an `io.Writer` and data, producing formatted output. No shared state except the global `DateFormat` variable.

**Formatters:**

| Function | Output |
|----------|--------|
| `NoteTable` | Tabular list with ID, title, age, tags |
| `NoteDetail` | Full note with metadata and backlinks |
| `NoteLog` | Git-log style compact view |
| `TagTable` | Two-column tag listing |
| `StatsTable` | Aggregate statistics |
| `HistoryTable` | Version history with diff summaries |
| `VersionDetail` | Single version snapshot |
| `JSON` | Generic JSON encoder to writer |
| `Markdown` | Markdown export format |

**Date formatting** is controlled by the `DateFormat` package variable, set from config at startup. See [configuration.md](configuration.md) for the three modes.

---

## Note linking

The `internal/linking` package extracts `@<prefix>` references from note bodies using a regex:

```
(?:^|[^a-zA-Z0-9])@([a-zA-Z0-9]{4,})
```

This matches `@` followed by 4+ alphanumeric characters, not preceded by another alphanumeric (to avoid matching email addresses). References are resolved to full note IDs using the same prefix-matching logic as CLI commands.

The `cmd/helpers.go` `syncNoteRefs` function orchestrates the flow: extract prefixes → resolve each → call `store.SyncRefs(noteID, resolvedIDs)` which reconciles `ref:*` tags (adds new references, removes stale ones).

---

## TUI architecture

The TUI uses the [Elm architecture](https://guide.elm-lang.org/architecture/) via [Bubbletea](https://github.com/charmbracelet/bubbletea):

### Model

`App` is the top-level model holding the store, dimensions, current view, view stack, and four sub-views (list, detail, compose, help).

### View stack

Views are pushed/popped like a stack. Pressing `Esc` or completing an action pops back to the previous view. The stack enables flows like List → Detail → Compose → (save) → Detail → (back) → List.

### Message types

| Message | Trigger |
|---------|---------|
| `notesLoadedMsg` | Notes fetched from store |
| `noteCreatedMsg` | Note created in compose view |
| `noteUpdatedMsg` | Note updated in compose view |
| `noteArchivedMsg` | Single note archived |
| `bulkArchivedMsg` | Multiple notes archived |
| `bulkPinnedMsg` | Multiple notes pinned/unpinned |
| `notePinnedMsg` | Single note pin toggled |
| `backlinksLoadedMsg` | Backlinks fetched for detail view |
| `searchResultsMsg` | FTS search results returned |
| `SearchTickMsg` | Debounce tick for search |
| `statusMsg` | Status bar message |
| `clearStatusMsg` | Auto-clear status after timeout |

### Search debounce

Search in the TUI uses a tick-based debounce. When the user types, a `SearchTickMsg` is scheduled. When the tick fires, it checks if the query has changed since the tick was emitted — if so, it's stale and ignored. This prevents hammering the FTS index on every keystroke.

### Theme system

All colours and styles live in `internal/tui/theme/theme.go`. View files import the `theme` package rather than defining local styles. The palette is Catppuccin Frappé, exposed as `lipgloss.Color` values and pre-built `lipgloss.Style` objects for each UI element.

---

## Key dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| `spf13/cobra` | v1.10 | CLI framework |
| `charmbracelet/bubbletea` | v1.3 | TUI framework (Elm architecture) |
| `charmbracelet/bubbles` | v1.0 | TUI components (text input, viewport) |
| `charmbracelet/lipgloss` | v1.1 | TUI styling |
| `catppuccin/go` | v0.3 | Colour palette |
| `modernc.org/sqlite` | v1.46 | Pure Go SQLite driver (no CGo) |
| `oklog/ulid/v2` | v2.1 | ULID generation |
| `BurntSushi/toml` | v1.6 | TOML config parsing |
| `filippo.io/age` | v1.3 | Encryption (scrypt) |
| `golang.org/x/sys` | v0.41 | Unix syscalls (flock) |
| `golang.org/x/term` | v0.40 | Terminal detection |
