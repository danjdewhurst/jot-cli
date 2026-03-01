# jot

A fast, context-aware notes tool for the terminal. Supports both interactive TUI and non-interactive CLI modes, automatically captures context (folder, git repo, branch) as tags, and provides full-text search via SQLite FTS5.

## Install

```bash
go install github.com/danjdewhurst/jot-cli@latest
```

Or build from source:

```bash
git clone https://github.com/danjdewhurst/jot-cli.git
cd jot-cli
make build
# Binary at bin/jot
```

## Usage

```
jot                              # No args → TUI (or list if not a TTY)
jot add [-t title] [-m body]     # Create note (opens $EDITOR if no -m/stdin)
jot list [--tag key:val]         # List notes
jot show <id>                    # Display note
jot edit <id> [-t title] [-m body]
jot rm <id> [--purge] [--force]  # Archive (or hard delete with --purge)
jot search <query>               # Full-text search
jot tag list [--key <key>]       # Browse tags
jot tag add <id> <key:value>     # Add tag
jot tag rm <id> <key:value>      # Remove tag
jot tui                          # Explicitly launch TUI
jot version
```

### Context tags

Notes are automatically tagged when created:

- `folder:<name>` — current directory name
- `git_repo:<owner/repo>` — git remote origin
- `git_branch:<branch>` — current git branch

Use `--no-context` to skip, or filter with shortcuts:

```bash
jot list --folder     # Notes from this directory
jot list --repo       # Notes from this git repo
jot list --branch     # Notes from this branch
```

### JSON output

All commands support `--json` for structured output, making jot usable by scripts and AI agents:

```bash
jot add -t "Bug report" -m "Login fails on Safari" --json
jot search "login" --json
```

### Piping

```bash
# Capture command output as a note
kubectl get pods | jot add -t "Pod status"

# Pipe content in
echo "Remember to update deps" | jot add -t "TODO" --json
```

### ID prefix matching

Note IDs are ULIDs (26 characters). You can use any unique prefix:

```bash
jot show 01KJM       # Matches if unambiguous
jot edit 01KJM -t "Updated title"
```

## Storage

```
~/.local/share/jot/jot.db      # Database (XDG_DATA_HOME)
```

Override with `JOT_DB` environment variable or `--db` flag.

## Environment variables

| Variable | Description |
|----------|-------------|
| `JOT_DB` | Database file path |
| `JOT_JSON` | Set to `1` for default JSON output |
| `EDITOR` / `VISUAL` | Editor for composing notes |
| `NO_COLOR` | Disable colour output |

## TUI key bindings

| Key | Action |
|-----|--------|
| `j`/`k` | Navigate up/down |
| `Enter` | Open note |
| `n` | New note |
| `e` | Edit note |
| `d` | Archive note |
| `/` | Search |
| `Tab` | Switch title/body (compose) |
| `Ctrl+S` | Save (compose) |
| `Esc` | Back |
| `q` | Quit |

## Development

Requires Go 1.25+. If using [mise](https://mise.jdx.dev/), the correct version is configured automatically.

```bash
make build    # Build to bin/jot
make test     # Run tests with -race
make install  # Install to $GOPATH/bin
```

## Licence

MIT
