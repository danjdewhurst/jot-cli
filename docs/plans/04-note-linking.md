# Plan: Note linking

**Priority:** Medium effort, high value

## Summary

Allow notes to reference other notes using `@<id-prefix>` syntax in note bodies. Store references as tags and render them as navigable links in the TUI.

## Syntax

In a note body:

```
Related to @abc123 and @def456
See also @789
```

## Behaviour

1. **On save** (create/edit): scan body for `@<hex-or-alnum>{4,}` patterns
2. **Store references** as tags with key `ref` and value being the full resolved note ID
3. **Render in TUI**: highlight `@` references, allow jumping to referenced note
4. **Backlinks**: `jot show <id>` includes "Referenced by" section listing notes that link to this one

## Implementation

### Model

- Add `ref` to the set of known system tag keys (alongside `folder`, `git_repo`, `git_branch`)

### Store layer

- `ReferencesTo(noteID)` — Query tags where key=`ref` and value=noteID → returns referencing notes
- No schema changes — uses existing `tags` table

### Parsing

- `internal/linking/` package — extract `@` references from body text
- Resolve prefix to full ID using existing prefix-match logic
- Unresolvable references logged as warnings, not stored

### CLI

- `cmd/show.go` — Add "Referenced by" section to detail output
- `cmd/add.go` / `cmd/edit.go` — Call linking parser after save, upsert `ref` tags

### TUI

- `DetailView` — Highlight `@` references, `Enter` on reference navigates to that note
- `ListView` — Optional: show link count indicator

## Edge cases

- Circular references: allowed, no special handling needed
- Deleted referenced note: `ref` tag becomes dangling; show "(deleted)" in output
- Updated body: re-scan and reconcile `ref` tags (remove stale, add new)

## Complexity

~300 lines. New package, store method, updates to add/edit/show commands, TUI enhancements.
