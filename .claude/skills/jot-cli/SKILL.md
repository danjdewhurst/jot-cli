---
name: jot-cli
description: Reference for jot-cli commands and usage. Use when creating, searching, editing, or managing notes with the jot-cli tool.
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

# Pin/unpin a note (pinned notes float to top of lists)
j pin <id> --json                 # Toggle pin
j unpin <id> --json               # Explicitly unpin

# List only pinned notes
j list --pinned --json

# Export notes to JSON
j export --json
j export -o backup.json
j export --tag "project:alpha" -o filtered.json
j export --format md -o notes.md          # Markdown (human-readable, not importable)
j export --since 2026-01-01 --until 2026-02-01 -o january.json
j export --search "deploy" -o matches.json
j export --archived -o all.json           # Include archived notes

# Import notes from JSON
j import backup.json --json
j import backup.json --dry-run            # Preview without writing
j import backup.json --new-ids            # Generate fresh IDs instead of preserving originals
j import backup.json --no-context         # Skip auto-context tags
j import backup.json --tag "source:backup" # Add extra tags to all imported notes
j import -                                # Read from stdin

# Pipe between instances
j export --tag project:alpha | j import --new-ids -

# Sync notes between machines (via shared directory)
j sync --json                                # Full push + pull
j sync status --json                         # Pending changes and last sync time
j sync push --json                           # Push local changes only
j sync pull --json                           # Pull remote changes only
j sync --sync-dir /path/to/shared --json     # Use specific sync directory

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
