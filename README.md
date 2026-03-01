<div align="center">

# jot-cli

**Notes that know where you are — a CLI-first notes tool with context-aware tagging, FTS5 search, a TUI, and first-class AI agent support.**

[![CI](https://img.shields.io/github/actions/workflow/status/danjdewhurst/jot-cli/ci.yml?branch=main&style=flat&label=CI)](https://github.com/danjdewhurst/jot-cli/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev)
[![Release](https://img.shields.io/github/v/release/danjdewhurst/jot-cli?style=flat&color=6C63FF)](https://github.com/danjdewhurst/jot-cli/releases/latest)
[![Homebrew](https://img.shields.io/badge/Homebrew-tap-FBB040?style=flat&logo=homebrew&logoColor=white)](https://github.com/danjdewhurst/homebrew-tap)
[![License: MIT](https://img.shields.io/badge/Licence-MIT-yellow.svg)](LICENSE)
[![SQLite](https://img.shields.io/badge/SQLite-FTS5-003B57?style=flat&logo=sqlite&logoColor=white)](https://www.sqlite.org/fts5.html)

</div>

---

## Why jot-cli?

Most note tools live in the browser. **jot-cli** lives where you work — in the terminal.

- **Context-aware** — automatically tags notes with your current folder, git repo, and branch
- **Fast search** — SQLite FTS5 full-text search across all notes and tags
- **Dual interface** — interactive TUI for browsing, CLI for scripting and automation
- **AI-friendly** — `--json` on every command, stdin piping, designed for agent workflows
- **Zero config** — works immediately, stores everything in a single SQLite file
- **Single binary** — pure Go, no CGo, no external dependencies at runtime

---

## Install

**Homebrew:**

```bash
brew tap danjdewhurst/tap
brew install jot-cli
```

This installs `jot-cli` and a `j` shorthand alias.

**Shell script:**

```bash
curl -fsSL https://raw.githubusercontent.com/danjdewhurst/jot-cli/main/install.sh | sh
```

To install a specific version or to a custom directory:

```bash
VERSION=0.3.0 INSTALL_DIR=/usr/local/bin curl -fsSL https://raw.githubusercontent.com/danjdewhurst/jot-cli/main/install.sh | sh
```

**Go:**

```bash
go install github.com/danjdewhurst/jot-cli@latest
```

**From source:**

```bash
git clone https://github.com/danjdewhurst/jot-cli.git
cd jot-cli
make build      # → bin/jot-cli
make install    # installs jot-cli + j alias
```

---

## Quick start

```bash
# Jot something down
j add -t "Fix the auth bug" -m "Token refresh fails after 24h"

# Search your notes
j search "auth bug"

# List notes from this repo
j list --repo

# Launch the TUI
j
```

> `j` and `jot-cli` are interchangeable — use whichever you prefer.

---

## Commands

| Command | Description |
|---------|-------------|
| `j` | Launch TUI (or list if not a TTY) |
| `j add` | Create a note — via flags, `$EDITOR`, or stdin |
| `j list` | List notes with optional tag filters |
| `j log` | Compact, git-log style chronological view of notes |
| `j show <id>` | Display a single note |
| `j edit <id>` | Edit a note — via flags or `$EDITOR` |
| `j rm <id>` | Archive a note (or `--purge --force` to delete) |
| `j dup <id>` | Duplicate a note (copies title, body, and user tags) |
| `j pin <id>` | Toggle pin on a note (pinned notes float to the top) |
| `j unpin <id>` | Explicitly unpin a note |
| `j stats` | Show aggregate note statistics |
| `j search <query>` | Full-text search with FTS5 |
| `j export` | Export notes to JSON or Markdown |
| `j import <file>` | Import notes from a JSON export |
| `j sync` | Synchronise notes via a shared directory |
| `j sync status` | Show pending changes and last sync time |
| `j sync push` | Push local changes only |
| `j sync pull` | Pull remote changes only |
| `j tag list` | Browse all tags |
| `j tag add <id> <key:value>` | Tag a note |
| `j tag rm <id> <key:value>` | Remove a tag |
| `j version` | Print version |

**Global flags:** `--json` `--db <path>` `--verbose`

> **Tip:** Note IDs are ULIDs — you can use any unique prefix instead of the full 26 characters:
> ```bash
> j show 01KJM
> ```

---

## Context tags

Every note is automatically tagged with where it was created:

| Tag | Source | Example |
|-----|--------|---------|
| `folder` | Current directory name | `folder:jot-cli` |
| `git_repo` | Remote origin URL | `git_repo:danjdewhurst/jot-cli` |
| `git_branch` | Current branch | `git_branch:main` |

Then filter by context later:

```bash
j list --folder      # Notes from this directory
j list --repo        # Notes from this git repo
j list --branch      # Notes from this branch
j list --tag git_repo:danjdewhurst/jot-cli   # Explicit tag filter
```

Skip auto-tagging with `--no-context`.

---

## Sync

Synchronise notes between machines using any shared folder — Dropbox, Syncthing, iCloud Drive, a mounted network share, etc.

```bash
# Full sync (push local changes, then pull remote changes)
j sync

# Check what's pending
j sync status

# Push or pull independently
j sync push
j sync pull

# Use a specific sync directory
j sync --sync-dir /mnt/nas/jot

# Machine-readable output
j sync --json
```

**How it works:** Each machine pushes its changes as NDJSON changeset files to the sync directory. On pull, changesets from other machines are ingested. Conflicts are resolved by last-write-wins using `updated_at` timestamps — the most recent edit always wins.

The sync directory defaults to `~/.local/share/jot/sync/` (respects `XDG_DATA_HOME`) and can be overridden with `--sync-dir` or `JOT_SYNC_DIR`.

---

## Piping & scripting

jot-cli is built for composability:

```bash
# Capture command output
kubectl get pods | j add -t "Pod status"

# Structured output for scripts
j search "deploy" --json | jq '.[].note.title'

# Export and re-import notes
j export --tag project:alpha -o backup.json
j import backup.json

# Pipe between instances
j export --tag project:alpha | j import --new-ids -

# Default to JSON everywhere
export JOT_JSON=1
```

---

## TUI

Run `j` with no arguments to launch the interactive interface.

<table>
<tr><th>Key</th><th>Action</th></tr>
<tr><td><kbd>j</kbd> <kbd>k</kbd></td><td>Navigate up / down</td></tr>
<tr><td><kbd>Enter</kbd></td><td>Open note</td></tr>
<tr><td><kbd>n</kbd></td><td>New note</td></tr>
<tr><td><kbd>e</kbd></td><td>Edit note</td></tr>
<tr><td><kbd>d</kbd></td><td>Archive note</td></tr>
<tr><td><kbd>p</kbd></td><td>Toggle pin</td></tr>
<tr><td><kbd>/</kbd></td><td>Search</td></tr>
<tr><td><kbd>Tab</kbd></td><td>Switch title / body (compose)</td></tr>
<tr><td><kbd>Ctrl+S</kbd></td><td>Save (compose)</td></tr>
<tr><td><kbd>Esc</kbd></td><td>Back</td></tr>
<tr><td><kbd>?</kbd></td><td>Help</td></tr>
<tr><td><kbd>q</kbd></td><td>Quit</td></tr>
</table>

---

## Configuration

**jot-cli works with zero configuration.** Everything below is optional.

### Storage

| Path | Purpose |
|------|---------|
| `~/.local/share/jot/jot.db` | Database (respects `XDG_DATA_HOME`) |
| `~/.local/share/jot/sync/` | Sync directory (respects `XDG_DATA_HOME`) |

### Environment variables

| Variable | Description |
|----------|-------------|
| `JOT_DB` | Override database path |
| `JOT_SYNC_DIR` | Override sync directory path |
| `JOT_JSON` | Set to `1` for JSON output by default |
| `EDITOR` / `VISUAL` | Editor for composing notes |
| `NO_COLOR` | Disable colour output |

---

## AI agent usage

jot-cli is designed to be called by AI agents (Claude Code, etc.) via shell. Every command supports `--json` for structured I/O.

```bash
# Agent creates a note
j add -t "Investigation notes" -m "Found the root cause in auth.go:42" --json

# Agent searches context
j search "root cause" --json

# Agent reads a specific note
j show 01KJM --json
```

### Claude Code skill

A [Claude Code skill](.claude/skills/jot-cli/SKILL.md) is included so Claude knows how to use jot-cli automatically. To install it globally (all projects):

```bash
# Copy the skill to your personal skills directory
mkdir -p ~/.claude/skills/jot-cli
cp .claude/skills/jot-cli/SKILL.md ~/.claude/skills/jot-cli/SKILL.md
```

Or with a one-liner if you don't have the repo cloned:

```bash
mkdir -p ~/.claude/skills/jot-cli && curl -fsSL \
  https://raw.githubusercontent.com/danjdewhurst/jot-cli/main/.claude/skills/jot-cli/SKILL.md \
  -o ~/.claude/skills/jot-cli/SKILL.md
```

Once installed, Claude will automatically reference jot-cli commands when relevant — no need to invoke anything manually.

---

## Development

Requires **Go 1.25+**. If using [mise](https://mise.jdx.dev/), the correct version is configured automatically.

```bash
make build      # Build to bin/jot-cli
make test       # Run tests with -race
make install    # Install jot-cli + j alias
```

### Architecture

```
cmd/              CLI commands (Cobra)
internal/
  model/          Domain types — Note, Tag, NoteFilter
  store/          SQLite data layer — CRUD, tags, FTS5
  sync/           File-based sync — push, pull, conflict resolution
  context/        Git & folder detection (filesystem only, no exec)
  editor/         $EDITOR integration
  render/         JSON & table output formatters
  config/         XDG path resolution
  tui/            Bubbletea TUI — views & components
```

### Dependencies

| Package | Purpose |
|---------|---------|
| [cobra](https://github.com/spf13/cobra) | CLI framework |
| [bubbletea](https://github.com/charmbracelet/bubbletea) | TUI framework |
| [bubbles](https://github.com/charmbracelet/bubbles) | TUI components |
| [lipgloss](https://github.com/charmbracelet/lipgloss) | TUI styling |
| [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) | Pure Go SQLite (no CGo) |
| [ulid](https://github.com/oklog/ulid) | Sortable unique IDs |

---

## Licence

[MIT](LICENSE) — Daniel Dewhurst
