# Plan: `jot dup`

**Priority:** High value, low effort

## Summary

Duplicate an existing note, creating a new note with the same title, body, and user tags. Auto-context tags are regenerated from the current environment.

## Commands

- `jot dup <id>` — Duplicate note, print new ID
- `jot dup <id> --json` — JSON output

## Behaviour

1. Resolve note by ID prefix (existing `resolveNote()` helper)
2. Create a new note with:
   - Title: original title (optionally prefixed with "Copy of " — decide during implementation)
   - Body: original body
   - Tags: copy user tags only (not auto-context tags `folder`, `git_repo`, `git_branch`)
   - New auto-context tags from current environment
   - Fresh ULID and timestamps
3. Print new note ID

## Implementation

### CLI

- `cmd/dup.go` — New command, ~40 lines
- Uses `resolveNote()` + `db.GetNote()` + `db.CreateNote()`
- Filter out context tag keys before copying tags

### Store layer

No changes needed — `CreateNote()` already handles everything.

## Complexity

~50 lines. Purely a CLI command composing existing store methods.
