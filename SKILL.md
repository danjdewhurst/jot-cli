# jot — CLI Notes Tool

Use `jot` to create, search, and manage notes from the terminal. All commands support `--json` for structured output.

## Quick Reference

```bash
# Create a note
jot add -t "Title" -m "Body text" --json

# Create with stdin
echo "note content" | jot add -t "From pipe" --json

# List all notes
jot list --json

# List notes for current folder/repo/branch
jot list --folder --json
jot list --repo --json
jot list --branch --json

# Filter by tag
jot list --tag "key:value" --json

# Full-text search
jot search "query terms" --json

# Show a note (prefix ID match supported)
jot show <id> --json

# Edit a note
jot edit <id> -t "New Title" -m "New Body" --json

# Archive a note
jot rm <id> --json

# Permanently delete
jot rm <id> --purge --force --json

# Manage tags
jot tag list --json
jot tag add <id> "key:value" --json
jot tag rm <id> "key:value" --json
```

## Context Tags

Notes are automatically tagged with context when created:
- `folder:<name>` — current directory name
- `git_repo:<owner/repo>` — git remote origin
- `git_branch:<branch>` — current git branch

Use `--no-context` with `jot add` to skip auto-tagging.

## Tips

- IDs are ULIDs. Use the first 8 characters as a prefix for convenience.
- Use `--json` for all machine-readable output.
- Pipe content into `jot add` for capturing command output as notes.
- Search supports FTS5 syntax: quotes for phrases, OR for alternatives.
