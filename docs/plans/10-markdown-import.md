# Plan: Markdown import

**Priority:** Bigger bet

## Summary

Import notes from markdown files, complementing the existing markdown export. Useful for migrating from other note-taking tools.

## Commands

```bash
jot import notes.md              # Single file
jot import ./notes-dir/          # Directory of .md files
jot import notes.md --dry-run    # Preview without creating
jot import notes.md --tag source:obsidian  # Add extra tags
```

## Markdown format

### With frontmatter (preferred)

```markdown
---
title: My Note
tags:
  - project:work
  - status:active
created: 2024-01-15T10:30:00Z
---

Note body content here.
```

### Without frontmatter

```markdown
# My Note

Note body content here.
```

- Title extracted from first `#` heading (existing behaviour in `add`)
- No tags unless `--tag` flags provided
- Created timestamp defaults to file modification time

## Implementation

### Parser

- `internal/importer/markdown.go` — Parse frontmatter + body
- Use `gopkg.in/yaml.v3` (or similar) for frontmatter parsing
- Handle both formats gracefully

### Deduplication

- Match by title + body hash to avoid duplicates on re-import
- `--force` to skip dedup check
- Report: "Created: 5, Skipped (duplicate): 3"

### Directory import

- Walk `.md` files recursively
- Skip non-markdown files
- Progress output: "Importing 42 files..."

### Integration

- Extend existing `cmd/import.go` to detect file type (JSON vs markdown)
- JSON import path unchanged
- Markdown import creates notes via `db.CreateNote()`
- Auto-context tags applied unless `--no-context`

## Complexity

~350 lines. New parser package, extend import command, frontmatter parsing.
