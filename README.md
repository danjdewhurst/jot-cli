<div align="center">

# jot

**A fast, context-aware notes tool for the terminal.**

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/Licence-MIT-yellow.svg)](LICENSE)
[![SQLite](https://img.shields.io/badge/SQLite-FTS5-003B57?style=flat&logo=sqlite&logoColor=white)](https://www.sqlite.org/fts5.html)

Notes that know where you are. Full-text search. Interactive TUI.<br>
Zero config. Single binary. Works with AI agents out of the box.

</div>

---

## Why jot?

Most note tools live in the browser. **jot** lives where you work — in the terminal.

- **Context-aware** — automatically tags notes with your current folder, git repo, and branch
- **Fast search** — SQLite FTS5 full-text search across all notes and tags
- **Dual interface** — interactive TUI for browsing, CLI for scripting and automation
- **AI-friendly** — `--json` on every command, stdin piping, designed for agent workflows
- **Zero config** — works immediately, stores everything in a single SQLite file
- **Single binary** — pure Go, no CGo, no external dependencies at runtime

---

## Install

**Go:**

```bash
go install github.com/danjdewhurst/jot-cli@latest
```

**From source:**

```bash
git clone https://github.com/danjdewhurst/jot-cli.git
cd jot-cli
make build    # → bin/jot
```

---

## Quick start

```bash
# Jot something down
jot add -t "Fix the auth bug" -m "Token refresh fails after 24h"

# Search your notes
jot search "auth bug"

# List notes from this repo
jot list --repo

# Launch the TUI
jot
```

---

## Commands

| Command | Description |
|---------|-------------|
| `jot` | Launch TUI (or list if not a TTY) |
| `jot add` | Create a note — via flags, `$EDITOR`, or stdin |
| `jot list` | List notes with optional tag filters |
| `jot show <id>` | Display a single note |
| `jot edit <id>` | Edit a note — via flags or `$EDITOR` |
| `jot rm <id>` | Archive a note (or `--purge --force` to delete) |
| `jot search <query>` | Full-text search with FTS5 |
| `jot tag list` | Browse all tags |
| `jot tag add <id> <key:value>` | Tag a note |
| `jot tag rm <id> <key:value>` | Remove a tag |
| `jot version` | Print version |

**Global flags:** `--json` `--db <path>` `--verbose`

> **Tip:** Note IDs are ULIDs — you can use any unique prefix instead of the full 26 characters:
> ```bash
> jot show 01KJM
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
jot list --folder      # Notes from this directory
jot list --repo        # Notes from this git repo
jot list --branch      # Notes from this branch
jot list --tag git_repo:danjdewhurst/jot-cli   # Explicit tag filter
```

Skip auto-tagging with `--no-context`.

---

## Piping & scripting

jot is built for composability:

```bash
# Capture command output
kubectl get pods | jot add -t "Pod status"

# Structured output for scripts
jot search "deploy" --json | jq '.[].note.title'

# Default to JSON everywhere
export JOT_JSON=1
```

---

## TUI

Run `jot` with no arguments to launch the interactive interface.

<table>
<tr><th>Key</th><th>Action</th></tr>
<tr><td><kbd>j</kbd> <kbd>k</kbd></td><td>Navigate up / down</td></tr>
<tr><td><kbd>Enter</kbd></td><td>Open note</td></tr>
<tr><td><kbd>n</kbd></td><td>New note</td></tr>
<tr><td><kbd>e</kbd></td><td>Edit note</td></tr>
<tr><td><kbd>d</kbd></td><td>Archive note</td></tr>
<tr><td><kbd>/</kbd></td><td>Search</td></tr>
<tr><td><kbd>Tab</kbd></td><td>Switch title / body (compose)</td></tr>
<tr><td><kbd>Ctrl+S</kbd></td><td>Save (compose)</td></tr>
<tr><td><kbd>Esc</kbd></td><td>Back</td></tr>
<tr><td><kbd>?</kbd></td><td>Help</td></tr>
<tr><td><kbd>q</kbd></td><td>Quit</td></tr>
</table>

---

## Configuration

**jot works with zero configuration.** Everything below is optional.

### Storage

| Path | Purpose |
|------|---------|
| `~/.local/share/jot/jot.db` | Database (respects `XDG_DATA_HOME`) |

### Environment variables

| Variable | Description |
|----------|-------------|
| `JOT_DB` | Override database path |
| `JOT_JSON` | Set to `1` for JSON output by default |
| `EDITOR` / `VISUAL` | Editor for composing notes |
| `NO_COLOR` | Disable colour output |

---

## AI agent usage

jot is designed to be called by AI agents (Claude Code, etc.) via shell. Every command supports `--json` for structured I/O, and a [SKILL.md](SKILL.md) is included as a Claude Code skill reference.

```bash
# Agent creates a note
jot add -t "Investigation notes" -m "Found the root cause in auth.go:42" --json

# Agent searches context
jot search "root cause" --json

# Agent reads a specific note
jot show 01KJM --json
```

---

## Development

Requires **Go 1.25+**. If using [mise](https://mise.jdx.dev/), the correct version is configured automatically.

```bash
make build      # Build to bin/jot
make test       # Run tests with -race
make install    # Install to $GOPATH/bin
```

### Architecture

```
cmd/              CLI commands (Cobra)
internal/
  model/          Domain types — Note, Tag, NoteFilter
  store/          SQLite data layer — CRUD, tags, FTS5
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
