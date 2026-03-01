# CLAUDE.md

## Project overview

jot-cli is a CLI-first notes app written in Go. It stores notes in SQLite with FTS5 full-text search, auto-tags notes with environment context (folder, git repo, branch), and has both a CLI and Bubbletea TUI.

## Build and test

```bash
# Build (requires Go 1.25+ via mise)
make build          # or: go build -o bin/jot .

# Run all tests
make test           # or: go test ./... -race -count=1

# Run a specific package's tests
go test ./internal/store/... -v
```

## Project structure

- `cmd/` — Cobra CLI commands (one file per command)
- `internal/model/` — Domain types: `Note`, `Tag`, `NoteFilter`
- `internal/store/` — SQLite data layer: CRUD, tags, FTS5 search
- `internal/context/` — Environment detection (git repo/branch, folder) via filesystem reads (no exec)
- `internal/editor/` — `$EDITOR` integration
- `internal/render/` — Output formatters (JSON and table)
- `internal/config/` — XDG path resolution
- `internal/tui/` — Bubbletea TUI app, views, and components

## Key conventions

- **Pure Go SQLite** via `modernc.org/sqlite` — no CGo dependency
- **ULID primary keys** — sortable, no coordination needed
- **FTS5 standalone table** — `notes_fts` with `note_id` column, managed manually (not content-synced)
- **Tags are key:value pairs** — auto-context tags use keys `folder`, `git_repo`, `git_branch`
- Migrations are embedded SQL files in `internal/store/migrations/`
- Tests use temp file databases (not `:memory:`) because Go's `database/sql` connection pool allocates separate in-memory DBs per connection

## Style

- Go standard formatting (`gofmt`)
- Errors wrapped with `fmt.Errorf("context: %w", err)`
- No `any` types except where required by interfaces (e.g., JSON rendering)
- British English in user-facing strings

## Adding a new CLI command

1. Create `cmd/<name>.go` with a `cobra.Command`
2. Register it in `init()` with `rootCmd.AddCommand()`
3. Use `db` (the global `*store.Store`) for data access
4. Support `--json` via `flagJSON` and `render.JSON()`
5. Use `resolveNote()` from `cmd/helpers.go` for ID prefix matching
