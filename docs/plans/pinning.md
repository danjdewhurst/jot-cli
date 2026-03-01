# Note Pinning / Favourites — Implementation Plan

## Overview

Add the ability to pin/favourite notes so they float to the top of `jot list` and TUI views. Provides `jot pin <id>` (toggle) and `jot unpin <id>` (explicit) commands, a `--pinned` filter flag on `jot list`, and a pin indicator in both CLI and TUI output.

Requires a new database migration to add a `pinned` column to the `notes` table.

---

## 1. Database Migration

### New file: `internal/store/migrations/002_add_pinned.sql`

```sql
ALTER TABLE notes ADD COLUMN pinned INTEGER NOT NULL DEFAULT 0;
```

### Migration runner update — `internal/store/store.go`

The current `migrate()` method only reads `001_initial.sql`. Update it to iterate over all migration files in order, tolerating `ALTER TABLE` re-runs by catching "duplicate column" errors:

```go
func (s *Store) migrate() error {
    conn, err := s.db.Conn(context.Background())
    if err != nil {
        return fmt.Errorf("getting connection: %w", err)
    }
    defer conn.Close()

    migrationFiles := []string{
        "migrations/001_initial.sql",
        "migrations/002_add_pinned.sql",
    }

    for _, file := range migrationFiles {
        data, err := migrations.ReadFile(file)
        if err != nil {
            return fmt.Errorf("reading migration %s: %w", file, err)
        }
        for _, stmt := range splitSQL(string(data)) {
            stmt = strings.TrimSpace(stmt)
            if stmt == "" {
                continue
            }
            if _, err := conn.ExecContext(context.Background(), stmt); err != nil {
                if strings.Contains(err.Error(), "duplicate column") {
                    continue
                }
                return fmt.Errorf("executing migration statement: %w\nSQL: %s", err, stmt)
            }
        }
    }
    return nil
}
```

**Note:** The sync plan proposes a more robust `PRAGMA user_version` approach. If sync lands first, this migration should use that system instead.

---

## 2. Model Changes

### Modify: `internal/model/note.go`

Add `Pinned` field to `Note` and `PinnedOnly` filter to `NoteFilter`:

```go
type Note struct {
    ID        string    `json:"id"`
    Title     string    `json:"title"`
    Body      string    `json:"body"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
    Archived  bool      `json:"archived"`
    Pinned    bool      `json:"pinned"`
    Tags      []Tag     `json:"tags,omitempty"`
}

type NoteFilter struct {
    Tags       []Tag
    Archived   bool
    PinnedOnly bool
    Limit      int
    Offset     int
}
```

---

## 3. Store Layer Changes

### Modify: `internal/store/notes.go`

**3a. Update all scan calls** to include `pinned`:

In `GetNote`:
```go
err := s.db.QueryRow(
    "SELECT id, title, body, created_at, updated_at, archived, pinned FROM notes WHERE id = ?", id,
).Scan(&n.ID, &n.Title, &n.Body, &createdAt, &updatedAt, &archived, &pinned)
n.Pinned = pinned != 0
```

In `ListNotes`:
```go
query := "SELECT DISTINCT n.id, n.title, n.body, n.created_at, n.updated_at, n.archived, n.pinned FROM notes n"
// ...
var pinned int
rows.Scan(&n.ID, &n.Title, &n.Body, &createdAt, &updatedAt, &archived, &pinned)
n.Pinned = pinned != 0
```

**3b. Update `ListNotes` ORDER BY** to float pinned notes to the top:

```go
query += " ORDER BY n.pinned DESC, n.created_at DESC"
```

**3c. Add `PinnedOnly` filter condition:**

```go
if filter.PinnedOnly {
    conditions = append(conditions, "n.pinned = 1")
}
```

**3d. Add pin/unpin/toggle methods:**

```go
func (s *Store) PinNote(id string) error {
    res, err := s.db.Exec("UPDATE notes SET pinned = 1, updated_at = ? WHERE id = ?",
        time.Now().UTC().Format(time.RFC3339), id)
    if err != nil {
        return fmt.Errorf("pinning note: %w", err)
    }
    if rows, _ := res.RowsAffected(); rows == 0 {
        return fmt.Errorf("note %q not found", id)
    }
    return nil
}

func (s *Store) UnpinNote(id string) error {
    res, err := s.db.Exec("UPDATE notes SET pinned = 0, updated_at = ? WHERE id = ?",
        time.Now().UTC().Format(time.RFC3339), id)
    if err != nil {
        return fmt.Errorf("unpinning note: %w", err)
    }
    if rows, _ := res.RowsAffected(); rows == 0 {
        return fmt.Errorf("note %q not found", id)
    }
    return nil
}

func (s *Store) TogglePin(id string) (pinned bool, err error) {
    note, err := s.GetNote(id)
    if err != nil {
        return false, err
    }
    if note.Pinned {
        return false, s.UnpinNote(id)
    }
    return true, s.PinNote(id)
}
```

### Modify: `internal/store/search.go`

Update the SELECT and Scan to include `pinned`:

```go
q := `SELECT n.id, n.title, n.body, n.created_at, n.updated_at, n.archived, n.pinned,
             snippet(notes_fts, 2, '<mark>', '</mark>', '…', 32) as snippet,
             notes_fts.rank
      FROM notes_fts
      JOIN notes n ON n.id = notes_fts.note_id`
```

---

## 4. CLI Commands

### New file: `cmd/pin.go`

```go
var pinCmd = &cobra.Command{
    Use:   "pin <id>",
    Short: "Toggle pin on a note",
    Long:  "Pin a note so it appears at the top of lists. Run again to unpin.",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        note, err := resolveNote(args[0])
        if err != nil {
            return err
        }
        pinned, err := db.TogglePin(note.ID)
        if err != nil {
            return err
        }
        if flagJSON {
            return render.JSON(os.Stdout, map[string]any{
                "id":     note.ID,
                "pinned": pinned,
            })
        }
        if pinned {
            fmt.Fprintf(os.Stderr, "Pinned note %s\n", note.ID[:8])
        } else {
            fmt.Fprintf(os.Stderr, "Unpinned note %s\n", note.ID[:8])
        }
        return nil
    },
}

var unpinCmd = &cobra.Command{
    Use:   "unpin <id>",
    Short: "Unpin a note",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        note, err := resolveNote(args[0])
        if err != nil {
            return err
        }
        if err := db.UnpinNote(note.ID); err != nil {
            return err
        }
        if flagJSON {
            return render.JSON(os.Stdout, map[string]any{
                "id":     note.ID,
                "pinned": false,
            })
        }
        fmt.Fprintf(os.Stderr, "Unpinned note %s\n", note.ID[:8])
        return nil
    },
}

func init() {
    rootCmd.AddCommand(pinCmd)
    rootCmd.AddCommand(unpinCmd)
}
```

### Modify: `cmd/list.go`

Add `--pinned` flag:

```go
listCmd.Flags().Bool("pinned", false, "Show only pinned notes")
```

In the `RunE`, set the filter:

```go
if pinned, _ := cmd.Flags().GetBool("pinned"); pinned {
    filter.PinnedOnly = true
}
```

---

## 5. Render Changes

### Modify: `internal/render/table.go`

Add a pin indicator before the title in `NoteTable`:

```go
title := n.Title
if n.Pinned {
    title = "* " + title
}
if len(title) > 40 {
    title = title[:37] + "..."
}
```

In `NoteDetail`, add a pinned line:

```go
if n.Pinned {
    fmt.Fprintf(w, "Pinned:  yes\n")
}
```

Use `*` rather than emoji (`📌`) to avoid width calculation issues in terminals that render emoji as double-width.

The JSON output requires no changes — the `Pinned` field is included automatically via the `json:"pinned"` struct tag.

---

## 6. TUI Changes

### Modify: `internal/tui/keys.go`

Add a Pin key binding:

```go
type keyMap struct {
    // ... existing ...
    Pin key.Binding
}

var keys = keyMap{
    // ... existing ...
    Pin: key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "toggle pin")),
}
```

### Modify: `internal/tui/app.go`

Add a message type:

```go
type notePinnedMsg struct {
    id     string
    pinned bool
}
```

Handle the pin key in `updateList`:

```go
case key.Matches(kmsg, keys.Pin):
    if note, ok := a.list.SelectedNote(); ok {
        s := a.store
        noteID := note.ID
        return a, func() tea.Msg {
            pinned, err := s.TogglePin(noteID)
            if err != nil {
                return statusMsg(fmt.Sprintf("Error: %v", err))
            }
            return notePinnedMsg{id: noteID, pinned: pinned}
        }
    }
```

Handle the message in `Update`:

```go
case notePinnedMsg:
    if msg.pinned {
        a.statusMsg = "Note pinned"
    } else {
        a.statusMsg = "Note unpinned"
    }
    return a, loadNotes(a.store, a.contextFilter)
```

Update the status bar hint:

```go
right = "n:new  p:pin  /:search  c:context  ?:help  q:quit"
```

### Modify: `internal/tui/views/list.go`

Add a pin indicator in the list rendering:

```go
pinIndicator := "  "
if n.Pinned {
    pinIndicator = "* "
}
```

### Modify: `internal/tui/views/detail.go`

Show pin status in the detail view:

```go
if d.note.Pinned {
    title = "* " + title
}
```

### Modify: `internal/tui/views/help.go`

Add pin to help text after the archive line:

```go
b.WriteString("  p       Toggle pin\n")
```

---

## 7. Testing Strategy

### Store tests — `internal/store/store_test.go`

| Test | Description |
|------|-------------|
| `TestPinNote` | Pin a note, retrieve it, verify `Pinned == true`. |
| `TestUnpinNote` | Pin then unpin a note, verify `Pinned == false`. |
| `TestTogglePin` | Toggle twice, verify state flips each time. |
| `TestPinnedNotesFloatToTop` | Create 3 notes, pin the oldest. Verify it appears first in `ListNotes`. |
| `TestPinnedOnlyFilter` | Create pinned and unpinned notes. Filter with `PinnedOnly: true`. Verify only pinned notes returned. |
| `TestPinNonExistentNote` | Attempt to pin a non-existent ID. Verify error. |
| `TestArchivedAndPinned` | Pin a note, archive it. Verify it does not appear in default list but does with `Archived: true`. |

---

## 8. Implementation Sequence

1. **Migration file** (`002_add_pinned.sql`) and migration runner update
2. **Model** (`note.go`) — add `Pinned` field and `PinnedOnly` filter
3. **Store** (`notes.go`, `search.go`) — update scans, ORDER BY, add pin methods
4. **Store tests** — verify all store changes
5. **CLI** (`cmd/pin.go`) — new `pin` and `unpin` commands
6. **CLI** (`cmd/list.go`) — add `--pinned` flag
7. **Render** (`table.go`) — pin indicator in table and detail output
8. **TUI** (`keys.go`, `app.go`, views) — key binding, toggle, indicator

---

## 9. Edge Cases

- **Archived + pinned:** A note can be both. The `archived = 0` filter takes precedence in default list views. Pin status is preserved through archive/unarchive.
- **Migration idempotency:** `ALTER TABLE ADD COLUMN` fails on re-run. The "duplicate column" error handling in the updated migrate function handles this.
- **Search results ordering:** Search results are ranked by FTS relevance, not pin status. The `pinned` field is still present in output for client-side use.
- **Terminal compatibility:** Use `*` not emoji for the pin indicator to avoid terminal width issues.

---

## 10. Future Considerations (Out of Scope)

- **Pin ordering** — Allow reordering pinned notes relative to each other (a `pin_position` column).
- **Pin limit** — Warn if too many notes are pinned (defeating the purpose).
- **Pin in search results** — Optionally boost pinned notes in search ranking.
