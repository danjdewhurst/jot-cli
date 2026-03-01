# Export/Import Feature — Implementation Plan

## Overview

Add `jot export` and `jot import` commands to allow users to back up, migrate, and share notes as portable files. The primary interchange format is JSON (matching the existing `model.Note` serialisation). A secondary Markdown export is provided for human-readable output.

No database migration is required — the feature operates entirely through existing store methods and the current schema.

---

## 1. Export File Format Specification

### 1.1 JSON (primary)

A JSON export file is a self-describing envelope wrapping an array of notes. The note objects reuse the exact `json` struct tags already on `model.Note` and `model.Tag`.

```json
{
  "version": 1,
  "exported_at": "2026-03-01T14:30:00Z",
  "count": 2,
  "notes": [
    {
      "id": "01JEXAMPLE00000000000000001",
      "title": "Example Note",
      "body": "Some content here.",
      "created_at": "2026-02-28T10:00:00Z",
      "updated_at": "2026-02-28T12:00:00Z",
      "archived": false,
      "tags": [
        {"key": "folder", "value": "/home/dan/projects"},
        {"key": "project", "value": "alpha"}
      ]
    }
  ]
}
```

**Rationale for the envelope:** The `version` field enables future schema evolution without breaking older exports. `count` enables quick validation and progress reporting on import.

### 1.2 Markdown (export only)

Each note is rendered as a Markdown document separated by `---`. This format is for human consumption and is not importable.

```markdown
# Example Note

- **ID:** 01JEXAMPLE00000000000000001
- **Created:** 2026-02-28T10:00:00Z
- **Updated:** 2026-02-28T12:00:00Z
- **Tags:** folder:/home/dan/projects, project:alpha

Some content here.

---
```

---

## 2. New Types

### 2.1 `internal/model/export.go`

```go
package model

import "time"

// ExportEnvelope wraps exported notes with metadata for format versioning.
type ExportEnvelope struct {
	Version    int       `json:"version"`
	ExportedAt time.Time `json:"exported_at"`
	Count      int       `json:"count"`
	Notes      []Note    `json:"notes"`
}

const ExportVersion = 1
```

### 2.2 `internal/model/import.go`

```go
package model

// ImportResult summarises the outcome of an import operation.
type ImportResult struct {
	Created  int      `json:"created"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors,omitempty"`
}
```

---

## 3. Store Layer Additions

### 3.1 `internal/store/import.go`

A new method on `*Store` to import a single note, preserving its original ID and timestamps.

```go
// ImportNote inserts a note with an explicit ID and timestamps.
// If a note with the same ID already exists, it is skipped (no error).
// Returns true if the note was created, false if it was skipped.
func (s *Store) ImportNote(n model.Note) (created bool, err error)
```

**Implementation details:**

1. Begin a transaction.
2. Attempt `INSERT OR IGNORE INTO notes (id, title, body, created_at, updated_at, archived) VALUES (?, ?, ?, ?, ?, ?)`.
3. Check `RowsAffected()`. If 0, the ID already exists — return `false, nil` (skip).
4. Call `insertTags(tx, n.ID, n.Tags)` to insert all tags.
5. Call `syncFTS(tx, n.ID)` to populate the FTS index.
6. Commit the transaction.

This approach uses `INSERT OR IGNORE` to handle duplicate IDs gracefully without requiring a separate existence check. The existing `insertTags` and `syncFTS` helper functions (both accept `*sql.Tx`) are already suitable for reuse.

### 3.2 Why no new migration is needed

The `notes`, `tags`, and `notes_fts` tables already support everything the import needs. The `notes.id` column is `TEXT PRIMARY KEY` so any valid ULID string can be inserted. No schema changes are required.

---

## 4. CLI Commands

### 4.1 `cmd/export.go` — `jot export`

```go
var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export notes to a file",
	RunE:  runExport,
}

func runExport(cmd *cobra.Command, args []string) error
```

**Flags:**

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--output` | `-o` | `string` | `""` (stdout) | Output file path |
| `--format` | `-f` | `string` | `"json"` | Format: `json` or `md` |
| `--tag` | | `[]string` | `nil` | Filter by tag (key:value), repeatable |
| `--archived` | | `bool` | `false` | Include archived notes |
| `--search` | `-s` | `string` | `""` | Filter by FTS search query |
| `--since` | | `string` | `""` | Only notes created after this date (RFC 3339 or YYYY-MM-DD) |
| `--until` | | `string` | `""` | Only notes created before this date (RFC 3339 or YYYY-MM-DD) |

**Logic:**

1. Parse filter flags. Build a `model.NoteFilter` from `--tag` and `--archived`.
2. If `--search` is set, use `db.Search()` and extract the notes from the results.
3. Otherwise, use `db.ListNotes()`.
4. Apply date-range filtering in-memory on `CreatedAt` (the store layer does not currently support date-range queries, and adding that complexity is not warranted for a first iteration).
5. Build an `ExportEnvelope` with `Version: 1`, `ExportedAt: time.Now().UTC()`, `Count: len(notes)`, and the notes slice.
6. If `--format` is `"md"`, call a new `render.Markdown(w, notes)` function.
7. If `--format` is `"json"`, use `render.JSON(w, envelope)`.
8. Write to `--output` file if specified, otherwise to `os.Stdout`.
9. Print a summary to `os.Stderr`: `Exported N notes to <path|stdout>`.

**Registration:**

```go
func init() {
	exportCmd.Flags().StringP("output", "o", "", "Output file path (default: stdout)")
	exportCmd.Flags().StringP("format", "f", "json", "Export format: json, md")
	exportCmd.Flags().StringSlice("tag", nil, "Filter by tag (key:value)")
	exportCmd.Flags().Bool("archived", false, "Include archived notes")
	exportCmd.Flags().StringP("search", "s", "", "Filter by search query")
	exportCmd.Flags().String("since", "", "Only notes created after this date")
	exportCmd.Flags().String("until", "", "Only notes created before this date")
	rootCmd.AddCommand(exportCmd)
}
```

### 4.2 `cmd/import.go` — `jot import`

```go
var importCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import notes from a JSON file",
	Args:  cobra.ExactArgs(1),
	RunE:  runImport,
}

func runImport(cmd *cobra.Command, args []string) error
```

**Flags:**

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--dry-run` | | `bool` | `false` | Preview what would be imported without writing |
| `--new-ids` | | `bool` | `false` | Generate fresh ULIDs instead of preserving originals |
| `--no-context` | | `bool` | `false` | Skip adding auto-context tags to imported notes |
| `--tag` | | `[]string` | `nil` | Additional tags to apply to all imported notes |

**Logic:**

1. Open and read the file at `args[0]`. If the path is `-`, read from `os.Stdin`.
2. Decode JSON into `ExportEnvelope`. Validate `Version == 1`; reject unknown versions with a clear error.
3. Validate `Count` matches `len(Notes)`. If not, warn to stderr but continue.
4. For each note in the envelope:
   a. If `--new-ids` is set, replace `n.ID` with `ulid.Make().String()` and set `CreatedAt`/`UpdatedAt` to `time.Now().UTC()`.
   b. If `!--no-context`, append auto-context tags (from `context.AutoTags()`).
   c. Append any extra tags from `--tag`.
   d. If `--dry-run`, print a summary line to stderr and continue.
   e. Call `db.ImportNote(n)`. Track created/skipped counts.
5. Build and return an `ImportResult`.
6. If `flagJSON`, render the result as JSON. Otherwise, print a human-readable summary to stderr.

**Registration:**

```go
func init() {
	importCmd.Flags().Bool("dry-run", false, "Preview import without writing")
	importCmd.Flags().Bool("new-ids", false, "Generate new IDs instead of preserving originals")
	importCmd.Flags().Bool("no-context", false, "Skip auto-context tags")
	importCmd.Flags().StringSlice("tag", nil, "Additional tags for all imported notes (key:value)")
	rootCmd.AddCommand(importCmd)
}
```

---

## 5. Render Additions

### 5.1 `internal/render/markdown.go`

```go
package render

import (
	"io"
	"github.com/danjdewhurst/jot-cli/internal/model"
)

// Markdown renders notes as a human-readable Markdown document.
func Markdown(w io.Writer, notes []model.Note) error
```

Each note is rendered as:

```
# <Title>

- **ID:** <full ULID>
- **Created:** <RFC 3339>
- **Updated:** <RFC 3339>
- **Archived:** true (only if true)
- **Tags:** key:value, key:value

<body>

---
```

If the title is empty, the heading line is omitted. Notes are separated by `---\n\n`.

---

## 6. Edge Cases and Design Decisions

### 6.1 Duplicate IDs on import

Handled by `INSERT OR IGNORE`. If a note with the same ULID already exists, the import skips it silently and increments the `Skipped` counter. This is the safest default — it allows re-running an import idempotently.

### 6.2 Tag deduplication

The `tags` table has a `UNIQUE(note_id, key, value)` constraint, and `insertTags` uses `INSERT OR IGNORE`. Importing a note with duplicate tags (or tags that overlap with auto-context) will not cause errors.

### 6.3 FTS consistency

`syncFTS` is called within the import transaction, so the FTS index is always kept in sync. No separate rebuild step is needed.

### 6.4 Large files

Notes are decoded from JSON in one pass (`json.Decoder` reads the full envelope). For extremely large exports (tens of thousands of notes), this is acceptable — each note is small. A streaming JSON approach would add significant complexity for marginal benefit and is not included in this iteration.

### 6.5 Importing from stdin

When the file argument is `-`, the command reads from `os.Stdin`. This enables piping: `jot export --tag project:alpha | jot import -`.

### 6.6 Date parsing for `--since`/`--until`

Accept both RFC 3339 (`2026-03-01T00:00:00Z`) and date-only (`2026-03-01`, interpreted as midnight UTC). Use `time.Parse` with both layouts, trying RFC 3339 first.

### 6.7 Markdown import (not supported)

Markdown export is lossy — it is designed for reading, not round-tripping. The import command only supports JSON. This is stated clearly in the `--help` text. If users request Markdown import in future, it can be added as a separate parser.

### 6.8 Version mismatch

If `envelope.Version > ExportVersion`, return an error: `"unsupported export version %d (this build supports up to %d)"`. This gives a clear upgrade path.

---

## 7. Testing Strategy

### 7.1 Unit tests — `internal/store/import_test.go`

| Test | Description |
|------|-------------|
| `TestImportNote_New` | Import a note with a known ULID. Verify it can be retrieved with correct title, body, timestamps, and tags. |
| `TestImportNote_Duplicate` | Import the same note twice. Verify the second call returns `created=false` and no error. Verify the original note is unchanged. |
| `TestImportNote_WithTags` | Import a note with multiple tags. Verify all tags are stored and FTS is searchable. |
| `TestImportNote_Archived` | Import a note with `archived=true`. Verify it only appears with `NoteFilter{Archived: true}`. |

### 7.2 Unit tests — `internal/model/export_test.go`

| Test | Description |
|------|-------------|
| `TestExportEnvelopeJSON` | Marshal and unmarshal an `ExportEnvelope`. Verify round-trip fidelity. |
| `TestExportVersionConstant` | Verify `ExportVersion == 1`. |

### 7.3 Unit tests — `internal/render/markdown_test.go`

| Test | Description |
|------|-------------|
| `TestMarkdownSingleNote` | Render one note. Verify title heading, metadata lines, and body are present. |
| `TestMarkdownMultipleNotes` | Render two notes. Verify they are separated by `---`. |
| `TestMarkdownEmptyTitle` | Render a note with no title. Verify no `#` heading is emitted. |
| `TestMarkdownNoNotes` | Render an empty slice. Verify output is empty. |

### 7.4 Integration tests — `cmd/export_test.go` and `cmd/import_test.go`

These test the full round-trip via the CLI layer, using `newTestStore` and invoking the command functions directly.

| Test | Description |
|------|-------------|
| `TestExportJSON_AllNotes` | Create 3 notes, export as JSON, decode the envelope, verify count and content. |
| `TestExportJSON_FilterByTag` | Create notes with different tags, export with `--tag`, verify only matching notes appear. |
| `TestExportJSON_DateRange` | Create notes, export with `--since`/`--until`, verify filtering. |
| `TestExportMarkdown` | Export as Markdown, verify output contains expected headings and separators. |
| `TestImportJSON_NewNotes` | Export notes, import into a fresh store, verify all notes exist with correct data. |
| `TestImportJSON_DuplicateSkip` | Import the same file twice, verify second run reports all skipped. |
| `TestImportJSON_NewIDs` | Import with `--new-ids`, verify imported notes have different IDs from the originals. |
| `TestImportJSON_DryRun` | Import with `--dry-run`, verify no notes are created. |
| `TestImportJSON_Stdin` | Pipe export output to import via `-`, verify round-trip works. |
| `TestImportJSON_BadVersion` | Attempt to import a file with `version: 99`, verify error message. |
| `TestImportJSON_MalformedJSON` | Attempt to import invalid JSON, verify a clear error. |

---

## 8. Implementation Sequence

1. **`internal/model/export.go`** — Define `ExportEnvelope`, `ImportResult`, and `ExportVersion` constant.
2. **`internal/store/import.go`** — Implement `ImportNote` method.
3. **`internal/store/import_test.go`** — Test the store-level import logic.
4. **`internal/render/markdown.go`** — Implement `Markdown` renderer.
5. **`internal/render/markdown_test.go`** — Test the Markdown renderer.
6. **`cmd/export.go`** — Implement the export command with all flags.
7. **`cmd/import.go`** — Implement the import command with all flags.
8. **Integration tests** — Full round-trip tests for both commands.

Steps 1-3 have no dependencies on each other beyond 1 preceding 2. Steps 4-5 are independent of steps 2-3. Steps 6-7 depend on all prior steps. Step 8 depends on steps 6-7.

---

## 9. Future Considerations (Out of Scope)

These are explicitly deferred to keep the initial implementation focused:

- **CSV export** — Could be added as another `--format` option.
- **Markdown import** — Would require a parser to extract metadata from the rendered format.
- **Selective conflict resolution** — e.g., `--on-conflict=overwrite|skip|error`. The current default (skip) is sufficient.
- **Streaming JSON** — For very large datasets, a line-delimited JSON format could be considered.
- **Export encryption** — Encrypting export files for secure transfer.
