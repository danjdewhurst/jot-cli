# User Guide

This guide covers jot-cli in depth. If you haven't installed it yet, see the [README](../README.md) for installation and a quick start.

Throughout this guide, `j` and `jot-cli` are interchangeable.

---

## Working with notes

### Creating notes

There are three ways to create a note:

**Inline with flags:**

```bash
j add -t "Meeting notes" -m "Discussed the migration plan"
```

**Via your editor:**

```bash
j add                         # opens $EDITOR
j add -t "Bug report"         # opens editor pre-filled with "# Bug report"
```

When the editor opens, write a `# heading` on the first line to set the title. Everything below becomes the body.

**From stdin:**

```bash
kubectl get pods | j add -t "Pod snapshot"
echo "Quick thought" | j add
```

Stdin is capped at 1 MiB. If both `--message` and stdin are absent, the editor opens.

An empty note (no title, no body) is discarded with an "empty note, aborting" error.

| Flag | Short | Description |
|------|-------|-------------|
| `--title` | `-t` | Note title |
| `--message` | `-m` | Note body |
| `--tag` | | Additional tags (`key:value`, repeatable) |
| `--no-context` | | Skip auto-context tags |

### Editing notes

```bash
j edit 01KJM                  # opens editor with current content
j edit 01KJM -t "New title"   # update title only
j edit 01KJM -m "New body"    # update body only
```

Without `-t` or `-m`, the editor opens pre-filled as `# <title>\n\n<body>`. The `# heading` convention applies: the first line becomes the title.

### Showing notes

```bash
j show 01KJM
j show 01KJM --json
```

The detail view includes the note's ID, title, timestamps, tags, body, and a "Referenced by" section listing any notes that link to it (backlinks).

### Archiving and deleting

**Archive** (soft delete — notes are hidden from default listings):

```bash
j archive 01KJM
j rm 01KJM                    # also archives by default
```

**Permanently delete:**

```bash
j rm 01KJM --purge --force
```

`--purge` requires `--force` as a safety measure. Purged notes cannot be recovered.

**View archived notes:**

```bash
j list --archived
j log --archived
```

### Duplicating notes

```bash
j dup 01KJM
```

Copies the title, body, and user tags into a new note with a fresh ID and timestamp.

---

## Note IDs

Notes use [ULIDs](https://github.com/ulid/spec) — 26-character, time-sorted, case-insensitive identifiers like `01JKXYZ1A2B3C4D5E6F7G8H9JK`.

You can use any unique prefix instead of the full ID:

```bash
j show 01KJM         # matches if only one note starts with "01KJM"
j edit 01            # fails if ambiguous (multiple matches)
```

If a prefix matches more than one note, jot returns an "ambiguous prefix" error listing the count. Add more characters until the prefix is unique.

The `list` and `log` commands display 8-character short IDs.

---

## Context tags

Every note is automatically tagged with your current environment:

| Tag | Source | Example |
|-----|--------|---------|
| `folder` | Current directory name | `folder:jot-cli` |
| `git_repo` | Remote origin URL (owner/repo) | `git_repo:danjdewhurst/jot-cli` |
| `git_branch` | Current branch | `git_branch:main` |

Context detection is **pure filesystem** — jot reads `.git/HEAD` and `.git/config` directly. It never shells out to `git`. This means it works in environments without git installed and supports worktrees (where `.git` is a file pointing to the real git dir).

If not inside a git repository, only the `folder` tag is added. If any detection fails silently, that tag is omitted.

**Skip auto-tagging:**

```bash
j add -t "General note" --no-context
```

**Filter by context:**

```bash
j list --folder       # notes from this directory
j list --repo         # notes from this git repo
j list --branch       # notes from this branch
```

These convenience flags are equivalent to `--tag folder:<current>`, etc.

---

## Note linking

Reference other notes in a note's body using `@<id-prefix>`:

```bash
j add -t "Follow-up" -m "See @01JMXY for context and @01JNAB for the fix"
```

**How it works:**

1. On save (create or edit), jot scans the body for `@<prefix>` references — 4+ alphanumeric characters, not preceded by another alphanumeric (so email addresses aren't matched)
2. Each prefix is resolved to a full note ID via the same prefix-matching logic as CLI commands
3. Resolved references are stored as `ref:<target-note-id>` tags
4. `j show <id>` displays a "Referenced by" section listing backlinks
5. The TUI detail view highlights `@` references and shows backlinks
6. Self-references are ignored
7. Unresolvable references are silently skipped (use `--verbose` to see warnings)
8. When a note is edited, references are reconciled — stale links removed, new ones added

---

## Searching

```bash
j search "auth bug"
j search "auth bug" --tag git_repo:myproject
j search "auth bug" --json
```

Search uses SQLite FTS5 with `unicode61` tokenisation and diacritics removal. The query is passed directly to FTS5's `MATCH` syntax, so you can use:

- Simple terms: `auth bug` (matches notes containing both words)
- Phrases: `"auth bug"` (matches the exact phrase)
- Prefix queries: `auth*` (matches words starting with "auth")
- Boolean operators: `auth AND bug`, `auth OR bug`, `auth NOT fix`

Results are ranked by FTS5 relevance. The `--tag` flag narrows results by tag after the text search.

JSON output includes a `snippet` field (with `<mark>` markers) and a `rank` score for each result.

---

## Browsing with log

The `log` command shows a compact, git-log style view:

```bash
j log                         # last 20 notes (default)
j log --today                 # today's notes only
j log --since 2025-01-01      # notes created after a date
j log --until 2025-06-30      # notes created before a date
j log --since 2025-01-01 --until 2025-06-30
j log --reverse               # oldest first
j log --limit 50              # override default limit
j log --tag folder:work       # filter by tag
j log --repo                  # filter by current repo
```

The default limit is 20 (overridable via `default_limit` in config). `--today` overrides `--since`/`--until` and removes the limit.

Date flags accept `YYYY-MM-DD` or full RFC 3339 timestamps.

---

## Version history

Every edit creates a version snapshot automatically.

```bash
j history 01KJM               # list all versions with diff summaries
j history 01KJM --version 2   # show a specific version
j history 01KJM --version 2 --diff   # show full diff
```

**Revert to a previous version:**

```bash
j revert 01KJM --version 3
```

Reverting creates a new version (it doesn't discard history).

---

## Bulk operations

Several commands support operating on multiple notes at once. You can target notes by IDs or by tag filter:

```bash
# By IDs
j archive abc123 def456 ghi789

# By tag filter
j archive --tag folder:old-project
j rm --tag git_repo:deleted-repo --purge --force

# Bulk tag
j tag add --tag folder:work project:active

# Bulk pin/unpin
j pin --tag priority:high
j unpin abc123 def456
```

**Safety:**

- Bulk operations require `--force` or interactive confirmation
- Use `--dry-run` to preview what would be affected without executing
- `--purge` (permanent delete) always requires `--force`

All bulk commands share these flags:

| Flag | Description |
|------|-------------|
| `--dry-run` | Preview without executing |
| `--force` | Skip confirmation |
| `--tag` | Filter by tag (`key:value`, repeatable) |
| `--folder` | Filter by current folder |
| `--repo` | Filter by current git repo |
| `--branch` | Filter by current git branch |
| `--archived` | Include archived notes |
| `--limit` | Limit number of results |

---

## Pinning

Pinned notes float to the top of listings and are marked with `*` in the table view.

```bash
j pin 01KJM                   # toggle pin (single note)
j pin abc123 def456           # bulk pin
j unpin 01KJM                 # unpin
j list --pinned               # show only pinned notes
```

With a single ID, `j pin` toggles the pin state. With multiple IDs or a filter, it always pins.

---

## Tagging

Tags are `key:value` pairs. Context tags (`folder`, `git_repo`, `git_branch`) are added automatically; you can also add your own.

```bash
# Add a tag
j tag add 01KJM project:alpha

# Bulk tag
j tag add --tag folder:work project:alpha

# Remove a tag
j tag rm 01KJM project:alpha

# List all tags
j tag list
j tag list --key folder       # filter by key
```

Tags are additive and idempotent — adding the same tag twice has no effect.

---

## Export and import

### Export

```bash
j export                              # all notes as JSON to stdout
j export -o backup.json               # write to file
j export -f md -o notes.md            # export as Markdown
j export --tag project:alpha          # filter by tag
j export --search "deploy"            # filter by FTS search
j export --since 2025-01-01           # filter by date
j export --archived                   # include archived notes
```

**JSON format** (`ExportEnvelope`):

```json
{
  "version": 1,
  "exported_at": "2025-03-01T12:00:00Z",
  "count": 42,
  "notes": [...]
}
```

**Markdown format:** Notes separated by `---`, each with a `# title`, metadata list, and body.

### Import

```bash
j import backup.json                  # import from JSON
j import notes.md                     # import from Markdown
j import ./notes-dir/                 # import all .md files from a directory
cat export.json | j import -          # import from stdin (JSON)
```

| Flag | Description |
|------|-------------|
| `--new-ids` | Generate new IDs instead of preserving originals |
| `--no-context` | Skip auto-context tags |
| `--dry-run` | Preview without writing |
| `--force` | Skip deduplication check (Markdown only) |
| `--tag` | Additional tags for all imported notes |

**Deduplication:**

- JSON import uses `INSERT OR IGNORE` on the ID — duplicate IDs are skipped
- Markdown import checks for matching title + body content — duplicates are skipped unless `--force` is set

**Markdown frontmatter** (optional YAML between `---`):

```yaml
---
title: My Note
tags:
  - project:alpha
created: 2025-01-01T00:00:00Z
---
```

Without frontmatter, the first `# heading` becomes the title. Timestamps fall back to file modification time.

**Pipe between instances:**

```bash
j export --tag project:alpha | j import --new-ids -
```

---

## Piping and scripting

jot-cli is designed for composability:

```bash
# Capture command output as a note
kubectl get pods | j add -t "Pod status"

# Structured output for scripts
j search "deploy" --json | jq '.[].note.title'

# Default to JSON everywhere
export JOT_JSON=1
```

Every command supports `--json` for structured output. Human-readable output goes to stderr; data goes to stdout. This means `--json` output is always clean for piping.

---

## TUI

Run `j` with no arguments (in a TTY) to launch the interactive interface.

### Views

| View | Description |
|------|-------------|
| **List** | Browse all notes, search, select, and act |
| **Detail** | Read a note with full metadata and backlinks |
| **Compose** | Create or edit a note with title and body fields |
| **Help** | Key binding reference |

### Key bindings

**List view:**

| Key | Action |
|-----|--------|
| `j` / `k` (or `↓` / `↑`) | Navigate down / up |
| `PgDn` / `PgUp` (or `Ctrl+D` / `Ctrl+U`) | Page down / up |
| `Enter` | Open note |
| `n` | New note |
| `e` | Edit note |
| `d` | Archive note (or selected notes) |
| `p` | Toggle pin (or pin selected) |
| `Space` | Toggle selection |
| `Ctrl+A` | Select all |
| `a` | Archive selected |
| `/` | Search |
| `c` | Toggle context filter |
| `?` | Help |
| `q` or `Ctrl+C` | Quit |
| `Esc` | Clear selection / back |

**Search mode** (within list view):

| Key | Action |
|-----|--------|
| Type | Filter notes by FTS query |
| `Enter` | Open selected result |
| `Esc` | Exit search |

Search uses a debounce — results update after you stop typing, not on every keystroke.

**Detail view:**

| Key | Action |
|-----|--------|
| `e` | Edit note |
| `Esc` | Back to list |

**Compose view:**

| Key | Action |
|-----|--------|
| `Tab` | Switch between title and body fields |
| `Ctrl+S` | Save |
| `Esc` | Cancel |

### Context filter

Press `c` in the list view to toggle the context filter. When enabled, only notes matching your current folder, git repo, and branch are shown. A status message confirms "Context filter: on/off".

---

## AI agent usage

jot-cli is designed for AI agent workflows:

```bash
# Create a note (returns JSON with the new note)
j add -t "Investigation" -m "Found issue in auth.go" --json

# Search (returns ranked results with snippets)
j search "root cause" --json

# Read a note
j show 01KJM --json

# Default to JSON for all commands
export JOT_JSON=1
```

A [Claude Code skill](../.claude/skills/jot-cli/SKILL.md) is included. See the README for installation instructions.

---

## Statistics

```bash
j stats
j stats --json
```

Shows total notes, archived count, pinned count, unique tags, top tags, weekly/monthly counts, and oldest/newest dates.
