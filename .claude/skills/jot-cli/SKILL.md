---
name: jot-cli
description: Reference for jot-cli commands and usage. Use when creating, searching, editing, or managing notes with the jot-cli tool.
user-invocable: false
---

# jot-cli — CLI Notes Tool

Use `jot-cli` (or `j` for short) to create, search, and manage notes from the terminal. All commands support `--json` for structured output.

## Quick Reference

```bash
# Create a note
j add -t "Title" -m "Body text" --json

# Create with stdin
echo "note content" | j add -t "From pipe" --json

# List all notes
j list --json

# List notes for current folder/repo/branch
j list --folder --json
j list --repo --json
j list --branch --json

# Filter by tag
j list --tag "key:value" --json

# Full-text search
j search "query terms" --json

# Show a note (prefix ID match supported)
j show <id> --json

# Edit a note
j edit <id> -t "New Title" -m "New Body" --json

# Archive a note
j rm <id> --json

# Permanently delete
j rm <id> --purge --force --json

# Manage tags
j tag list --json
j tag add <id> "key:value" --json
j tag rm <id> "key:value" --json
```

## Context Tags

Notes are automatically tagged with context when created:
- `folder:<name>` — current directory name
- `git_repo:<owner/repo>` — git remote origin
- `git_branch:<branch>` — current git branch

Use `--no-context` with `j add` to skip auto-tagging.

## Tips

- IDs are ULIDs. Use the first 8 characters as a prefix for convenience.
- Use `--json` for all machine-readable output.
- Pipe content into `j add` for capturing command output as notes.
- Search supports FTS5 syntax: quotes for phrases, OR for alternatives.
- `j` and `jot-cli` are the same binary — use whichever you prefer.
