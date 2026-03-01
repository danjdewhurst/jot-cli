# Plan: `jot context`

**Priority:** High value, low effort

## Summary

Print the auto-detected environment context tags without creating a note. Useful for debugging why notes get tagged a certain way.

## Commands

- `jot context` — Print detected context as key:value pairs
- `jot context --json` — JSON output

## Output

```
folder:     jot-cli
git_repo:   github.com/user/jot-cli
git_branch: feat/stats
```

Or when not in a git repo:

```
folder:     documents
git_repo:   (none)
git_branch: (none)
```

## Implementation

### CLI

- `cmd/context.go` — New command, ~30 lines
- Calls `context.AutoTags()` directly
- Table output by default, `--json` support
- No store access needed

## Complexity

~30 lines. Trivial — just surfaces existing `internal/context` functionality.
