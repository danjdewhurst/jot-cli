# Plan: Bulk operations

**Priority:** Bigger bet

## Summary

Support multi-note operations in both CLI and TUI — batch archive, tag, delete, and pin.

## CLI

### Batch by filter

```bash
jot archive --tag folder:old-project    # Archive all notes matching filter
jot tag add --tag folder:work project:active  # Add tag to filtered notes
jot rm --tag git_repo:deleted-repo --purge --force  # Bulk delete
jot pin --tag priority:high             # Bulk pin
```

### Batch by IDs

```bash
jot archive abc123 def456 ghi789
jot tag add abc123 def456 status:done
```

### Safety

- All destructive bulk operations require `--force` or interactive confirmation
- Show count and preview before executing: "Archive 23 notes? [y/N]"
- `--dry-run` flag shows what would be affected

## TUI

### Multi-select mode

1. Press `Space` to toggle selection on current note
2. Visual indicator (checkbox or highlight) for selected notes
3. Press `a` to archive selected, `t` to tag selected, `d` to delete selected
4. Confirmation prompt before executing
5. `Ctrl+A` to select all visible, `Escape` to clear selection

### Implementation

- Add `selected map[string]bool` to `ListView` state
- Render selected notes with distinct style
- Action commands check for selections before single-note operation

## Store layer

- `ArchiveNotes(ids []string)` — Batch archive in single transaction
- `DeleteNotes(ids []string)` — Batch delete
- `AddTagToNotes(ids []string, tag Tag)` — Batch tag
- `PinNotes(ids []string)` / `UnpinNotes(ids []string)`
- All wrap operations in a transaction for atomicity

## Complexity

~500 lines. Store batch methods, CLI flag handling, TUI multi-select mode.
