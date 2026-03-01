# Plan: `jot stats`

**Priority:** High value, low effort

## Summary

Show aggregate statistics about notes — counts, tag frequency, activity over time, notes per repo/folder.

## Commands

- `jot stats` — Print a summary dashboard
- `jot stats --json` — Machine-readable output

## Output

```
Notes:      142 (3 archived)
Pinned:     7
Tags:       58 unique
Top tags:   git_repo:jot-cli (34), folder:work (21), git_branch:main (18)
This week:  12 notes
This month: 31 notes
Oldest:     2024-11-03
```

## Implementation

### Store layer

- `Stats()` method returning a `NoteStats` struct
- Queries: `COUNT(*)` with filters, `GROUP BY` on tags, date-range counts
- Single method, multiple queries within one call

### Model

- `NoteStats` struct: total, archived, pinned, tag count, top tags, weekly/monthly counts, oldest/newest dates

### CLI

- `cmd/stats.go` — Register under `rootCmd`
- Table output by default, `--json` support
- No filtering flags in v1 — always global stats

### TUI

- Optional: show stats in a footer or dedicated view (defer to later)

## Complexity

~150 lines. One new command, one new store method, one new model type.
