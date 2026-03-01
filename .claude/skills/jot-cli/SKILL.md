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

# Archive multiple notes (by ID or tag filter)
j archive <id1> <id2> --json
j archive --tag folder:old-project --force --json

# Permanently delete
j rm <id> --purge --force --json
j rm --tag git_repo:stale --purge --force --json  # Bulk delete

# Pin/unpin a note (pinned notes float to top of lists)
j pin <id> --json                 # Toggle pin (single)
j pin <id1> <id2> --json          # Bulk pin
j pin --tag priority:high --force --json  # Bulk pin by filter
j unpin <id> --json               # Explicitly unpin
j unpin --tag priority:high --force --json  # Bulk unpin

# List only pinned notes
j list --pinned --json

# Chronological log (compact, git-log style)
j log --json                                 # Default: 20 most recent notes
j log --today --json                         # Today's notes (no limit)
j log --since 2026-01-01 --until 2026-02-01 --json
j log --reverse --json                       # Oldest first
j log --limit 50 --json                      # Custom limit (0 for unlimited)
j log --tag "project:alpha" --json           # Filter by tag
j log --repo --json                          # Filter by current git repo

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

# Note statistics
j stats                                      # Human-readable summary
j stats --json                               # Machine-readable output

# Show detected environment context
j context                                    # Table output: folder, git_repo, git_branch
j context --json                             # Machine-readable output

# Duplicate a note (copies title, body, user tags; regenerates context tags)
j dup <id> --json

# Manage tags
j tag list --json
j tag add <id> "key:value" --json
j tag add <id1> <id2> "key:value" --json          # Bulk tag by IDs
j tag add --tag folder:work "project:active" --force --json  # Bulk tag by filter
j tag rm <id> "key:value" --json

# Bulk operations — safety flags
# --dry-run: preview what would be affected
j archive --tag folder:old --dry-run
# --force: skip confirmation for bulk operations
j archive --tag folder:old --force

# Note history — view and revert to previous versions
j history <id> --json                        # List all versions
j history <id> --version 1 --json            # Show specific version
j history <id> --version 2 --diff            # Show version with diff
j revert <id> --version 1 --json             # Revert to a previous version

# Configuration
j config                                     # Print resolved config (all sources merged)
j config --path                              # Print config file path
j config init                                # Create default config file at ~/.config/jot/config.toml

# Note linking — reference other notes with @<id-prefix> in the body
j add -t "Follow-up" -m "Related to @01JMXY and @01JNAB" --json

# Show backlinks (notes that reference this one)
j show <id> --json    # JSON output includes "backlinks" array
```

## Context Tags

Notes are automatically tagged with context when created:
- `folder:<name>` — current directory name
- `git_repo:<owner/repo>` — git remote origin
- `git_branch:<branch>` — current git branch

Use `--no-context` with `j add` to skip auto-tagging.

## Note Linking

- Reference other notes in body text using `@<id-prefix>` (4+ characters)
- References are auto-resolved to full note IDs and stored as `ref` tags
- `j show <id>` displays a "Referenced by" section (backlinks)
- References are reconciled on every save (create/edit) — stale links removed, new ones added

## Tips

- IDs are ULIDs. Use the first 8 characters as a prefix for convenience.
- Use `--json` for all machine-readable output.
- Pipe content into `j add` for capturing command output as notes.
- Search supports FTS5 syntax: quotes for phrases, OR for alternatives.
- `j` and `jot-cli` are the same binary — use whichever you prefer.
