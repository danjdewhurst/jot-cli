# `jot log` Command — Implementation Plan

## Overview

Add a `jot log` command that displays a compact, git-log style chronological view of notes. Unlike the tabular `jot list`, the log format uses vertical stacking with colour-coded output — each note rendered as a tight block with timestamp, ID, title/body preview, and tags. Optimised for scanning a day or week of notes at a glance.

No database migration is required. The feature builds on existing store methods with a small extension to `NoteFilter` for date-range filtering and sort direction.

---

## 1. Model Changes

### Extend `NoteFilter` — `internal/model/note.go`

Add `Since`, `Until`, and `SortAsc` fields:

```go
type NoteFilter struct {
    Tags     []Tag
    Archived bool
    Limit    int
    Offset   int
    Since    *time.Time // only notes created at or after this time
    Until    *time.Time // only notes created before this time
    SortAsc  bool       // if true, order by created_at ASC (oldest first)
}
```

Using `*time.Time` means zero-value is `nil`, so existing callers are unaffected.

---

## 2. Store Layer Changes

### Date-range conditions in `ListNotes` — `internal/store/notes.go`

After the existing filter conditions, add:

```go
if filter.Since != nil {
    conditions = append(conditions, "n.created_at >= ?")
    args = append(args, filter.Since.Format(time.RFC3339))
}
if filter.Until != nil {
    conditions = append(conditions, "n.created_at < ?")
    args = append(args, filter.Until.Format(time.RFC3339))
}
```

Uses strict `<` for `Until` so `--until 2026-03-01` means "before midnight on that day".

### Sort direction

Change the static ORDER BY:

```go
if filter.SortAsc {
    query += " ORDER BY n.created_at ASC"
} else {
    query += " ORDER BY n.created_at DESC"
}
```

**Why push date filtering to SQL rather than filter in-memory:** The `LIMIT` clause interacts with filtering. In-memory filtering after `LIMIT` would return fewer results than requested.

---

## 3. Render Layer — `internal/render/log.go`

### 3.1 Colour palette

Use `lipgloss` (already a dependency via the TUI layer):

```go
package render

import "github.com/charmbracelet/lipgloss"

var (
    logHashStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("214")) // amber
    logTimestampStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // dim grey
    logTitleStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")) // bright white
    logTagKeyStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("243")) // mid grey
    logTagValueStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("110")) // soft blue
)
```

### 3.2 Output format

Each note is a compact block mirroring `git log --oneline` with an optional second line for tags:

```
01HXK3M2  2026-03-01 14:30  Fix deployment script for staging
                             folder:/home/dan/projects  git_branch:main

01HXK2P7  2026-03-01 13:15  Meeting notes — Q1 review with team
                             project:alpha

01HXK1A9  2026-03-01 09:00  Quick thought about caching strategy
```

When a note has no title, the first line of the body is used (truncated to 60 chars, newlines replaced with spaces). When neither exists, `(empty)` is shown.

### 3.3 Function signature

```go
// NoteLog renders notes in a compact, git-log style chronological format
// with colour-coded output.
func NoteLog(w io.Writer, notes []model.Note)
```

### 3.4 Implementation

```go
func NoteLog(w io.Writer, notes []model.Note) {
    if len(notes) == 0 {
        fmt.Fprintln(w, "No notes found.")
        return
    }

    for i, n := range notes {
        id := logHashStyle.Render(shortID(n.ID))
        ts := logTimestampStyle.Render(n.CreatedAt.Format("2006-01-02 15:04"))

        title := n.Title
        if title == "" {
            title = truncateLog(n.Body, 60)
        }
        if title == "" {
            title = "(empty)"
        }
        title = logTitleStyle.Render(title)

        fmt.Fprintf(w, "%s  %s  %s\n", id, ts, title)

        if len(n.Tags) > 0 {
            var parts []string
            for _, t := range n.Tags {
                parts = append(parts, logTagKeyStyle.Render(t.Key+":")+logTagValueStyle.Render(t.Value))
            }
            indent := strings.Repeat(" ", 28)
            fmt.Fprintf(w, "%s%s\n", indent, strings.Join(parts, "  "))
        }

        if i < len(notes)-1 {
            fmt.Fprintln(w)
        }
    }
}

func truncateLog(s string, max int) string {
    s = strings.ReplaceAll(s, "\n", " ")
    s = strings.TrimSpace(s)
    if len(s) > max {
        return s[:max-1] + "..."
    }
    return s
}
```

`lipgloss` automatically strips ANSI codes when output is piped to a non-terminal, so no `--no-colour` flag is needed.

---

## 4. Date Parsing Helper — `cmd/helpers.go`

Shared helper for parsing user-supplied date strings (also used by the export command):

```go
// parseDate parses a date string in RFC 3339 or YYYY-MM-DD format.
// Date-only strings are interpreted as midnight UTC.
func parseDate(s string) (time.Time, error) {
    if t, err := time.Parse(time.RFC3339, s); err == nil {
        return t, nil
    }
    if t, err := time.Parse("2006-01-02", s); err == nil {
        return t, nil
    }
    return time.Time{}, fmt.Errorf("invalid date %q (expected YYYY-MM-DD or RFC 3339)", s)
}
```

---

## 5. CLI Command — `cmd/log.go`

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--tag` | `[]string` | `nil` | Filter by tag (key:value), repeatable |
| `--folder` | `bool` | `false` | Filter by current folder |
| `--repo` | `bool` | `false` | Filter by current git repo |
| `--branch` | `bool` | `false` | Filter by current git branch |
| `--archived` | `bool` | `false` | Include archived notes |
| `--limit` | `int` | `20` | Maximum number of notes (0 for unlimited) |
| `--since` | `string` | `""` | Only notes created at or after this date |
| `--until` | `string` | `""` | Only notes created before this date |
| `--reverse` | `bool` | `false` | Oldest notes first |
| `--today` | `bool` | `false` | Shorthand: today's notes, no limit |
| `--json` | `bool` | `false` | (inherited from root) JSON output |

### Command definition

```go
var logCmd = &cobra.Command{
    Use:   "log",
    Short: "Show a chronological log of notes",
    Long:  "Display notes in a compact, git-log style chronological view.",
    RunE:  runLog,
}

func runLog(cmd *cobra.Command, args []string) error {
    filter := model.NoteFilter{}

    // Parse tag, context, date, and limit flags into filter...

    // Default limit of 20 for log (more compact than list but still bounded)
    limit, _ := cmd.Flags().GetInt("limit")
    if limit == 0 {
        limit = 20
    }
    filter.Limit = limit

    // --today shorthand
    if today, _ := cmd.Flags().GetBool("today"); today {
        now := time.Now()
        startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
        filter.Since = &startOfDay
        filter.Limit = 0 // no limit for today
    }

    reverse, _ := cmd.Flags().GetBool("reverse")
    filter.SortAsc = reverse

    notes, err := db.ListNotes(filter)
    if err != nil {
        return err
    }

    if flagJSON {
        return render.JSON(os.Stdout, notes)
    }

    render.NoteLog(os.Stdout, notes)
    return nil
}

func init() {
    logCmd.Flags().StringSlice("tag", nil, "Filter by tag (key:value)")
    logCmd.Flags().Bool("folder", false, "Filter by current folder")
    logCmd.Flags().Bool("repo", false, "Filter by current git repo")
    logCmd.Flags().Bool("branch", false, "Filter by current git branch")
    logCmd.Flags().Bool("archived", false, "Include archived notes")
    logCmd.Flags().Int("limit", 0, "Maximum number of notes (default: 20)")
    logCmd.Flags().String("since", "", "Show notes created after this date (YYYY-MM-DD or RFC 3339)")
    logCmd.Flags().String("until", "", "Show notes created before this date (YYYY-MM-DD or RFC 3339)")
    logCmd.Flags().Bool("reverse", false, "Show oldest notes first")
    logCmd.Flags().Bool("today", false, "Show only today's notes")
    rootCmd.AddCommand(logCmd)
}
```

---

## 6. Edge Cases and Design Decisions

### 6.1 Default limit of 20

Unlike `jot list` (unlimited), `jot log` defaults to 20. The log format uses 2-3 lines per entry, so showing too many defeats the purpose. Override with `--limit 0`.

### 6.2 `--today` flag

Convenience shorthand. Sets `--since` to midnight UTC today and removes the limit. Covers the most common use case: "what did I note down today?"

### 6.3 Empty title fallback

When a note has no title, the first line of the body is used (truncated, newlines replaced with spaces). When both are empty, `(empty)` is shown. Matches the TUI list view behaviour.

### 6.4 Colour suppression in pipes

`lipgloss` uses `muesli/termenv` which automatically detects non-terminal output and strips ANSI codes. No manual `--no-colour` flag needed.

### 6.5 `--reverse` with `--limit`

`--reverse --limit 10` returns the 10 oldest notes (SQL `ORDER BY ASC LIMIT 10`). This is correct and expected.

---

## 7. Testing Strategy

### Store tests — `internal/store/store_test.go`

| Test | Description |
|------|-------------|
| `TestListNotes_Since` | Create notes at known times. Filter with `Since`. Verify only matching notes returned. |
| `TestListNotes_Until` | Filter with `Until`. Verify only notes before the time are returned. |
| `TestListNotes_SinceAndUntil` | Combine both. Verify the intersection. |
| `TestListNotes_SortAsc` | Filter with `SortAsc: true`. Verify ascending order. |

### Render tests — `internal/render/log_test.go`

| Test | Description |
|------|-------------|
| `TestNoteLog_Empty` | Empty slice. Verify "No notes found." output. |
| `TestNoteLog_SingleNote` | Render one note. Verify ID, timestamp, title present. |
| `TestNoteLog_NoTitle` | Note with no title but a body. Verify body fallback. |
| `TestNoteLog_NoTitleNoBody` | Empty note. Verify `(empty)` appears. |
| `TestNoteLog_TagFormatting` | Verify tags render as `key:value` pairs. |
| `TestNoteLog_LongBody` | Verify truncation with ellipsis. |

### Helper tests — `cmd/helpers_test.go`

| Test | Description |
|------|-------------|
| `TestParseDate_RFC3339` | Parse full RFC 3339 string. |
| `TestParseDate_DateOnly` | Parse `YYYY-MM-DD`, verify midnight UTC. |
| `TestParseDate_Invalid` | Parse garbage, verify error. |

### Integration tests — `cmd/log_test.go`

| Test | Description |
|------|-------------|
| `TestLogCmd_Default` | Create 25 notes. Verify only 20 shown. |
| `TestLogCmd_Limit` | Create 10 notes, `--limit 5`. Verify 5 entries. |
| `TestLogCmd_Today` | Notes today and yesterday. `--today` shows only today's. |
| `TestLogCmd_Reverse` | Create 3 notes, `--reverse`. Verify oldest first. |
| `TestLogCmd_TagFilter` | Different tags, filter with `--tag`. |
| `TestLogCmd_JSON` | `--json` outputs valid JSON array. |
| `TestLogCmd_NoNotes` | Empty DB. Verify "No notes found." |

---

## 8. Implementation Sequence

1. **`internal/model/note.go`** — Add `Since`, `Until`, `SortAsc` to `NoteFilter`.
2. **`internal/store/notes.go`** — Add date-range conditions and sort direction to `ListNotes`.
3. **Store tests** — Verify new filter fields.
4. **`internal/render/log.go`** — Implement `NoteLog` renderer.
5. **Render tests** — Test the renderer.
6. **`cmd/helpers.go`** — Add `parseDate` helper.
7. **`cmd/log.go`** — Implement the command with all flags.
8. **Integration tests**.

Steps 1-2 are sequential. Steps 4-6 are independent of each other and of 2-3. Steps 7-8 depend on all prior steps.

---

## 9. Future Considerations (Out of Scope)

- **`--format` flag** — Support `oneline`, `full`, `short` formats like `git log`.
- **`--body` flag** — Show truncated body preview below the title.
- **`--no-tags` flag** — Hide tags for an even more compact view.
- **Pager integration** — Pipe through `$PAGER` when output exceeds terminal height.
- **Day-group headers** — Separator lines between days (e.g. `── 2026-03-01 ──`) with time-only timestamps within each group.
- **Context grouping** — Group notes by git repo or folder.
