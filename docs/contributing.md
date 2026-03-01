# Contributing Guide

Thanks for your interest in contributing to jot-cli. This guide covers development setup, conventions, and how to add new features.

---

## Development setup

**Prerequisites:**

- **Go 1.25+** — if using [mise](https://mise.jdx.dev/), the correct version is configured automatically via `.tool-versions`
- **golangci-lint** — for linting (`make lint`)

**Clone and build:**

```bash
git clone https://github.com/danjdewhurst/jot-cli.git
cd jot-cli
make build      # → bin/jot-cli
make install    # installs jot-cli + j symlink to ~/.local/bin
```

**Make targets:**

| Target | Command | Description |
|--------|---------|-------------|
| `build` | `go build -ldflags "..." -o bin/jot-cli .` | Build binary with version info |
| `test` | `go test ./... -race -count=1` | Run all tests with race detector |
| `lint` | `golangci-lint run ./...` | Run linter |
| `install` | Build + copy to `~/.local/bin` + symlink `j` | Install locally |
| `clean` | `rm -rf bin/` | Remove build artefacts |

---

## Running tests

```bash
make test                           # all tests
go test ./internal/store/... -v     # specific package
go test ./... -run TestSearch -v    # specific test
```

**Important:** Tests use **temp file databases**, not `:memory:`. This is because Go's `database/sql` connection pool can allocate separate in-memory databases per connection, which breaks assumptions about shared state. Temp files ensure a single consistent database.

The `-race` flag is always used to catch data races. `-count=1` disables test caching.

---

## Code style

- **Formatting:** `gofmt` (standard Go formatting)
- **Error wrapping:** Always wrap with context: `fmt.Errorf("doing thing: %w", err)`
- **Language:** British English in user-facing strings, comments, and documentation (e.g. "synchronise", "colour", "initialise")
- **No `any` types** except where required by interfaces (e.g. JSON rendering with `map[string]any`)
- **No unnecessary abstractions** — keep things simple and direct

---

## Adding a new CLI command

Follow this pattern:

### 1. Create `cmd/<name>.go`

```go
package cmd

import (
    "fmt"
    "os"

    "github.com/danjdewhurst/jot-cli/internal/render"
    "github.com/spf13/cobra"
)

var myCmd = &cobra.Command{
    Use:   "mycommand <arg>",
    Short: "One-line description",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        // Use resolveNote() for ID prefix matching
        note, err := resolveNote(args[0])
        if err != nil {
            return err
        }

        // Support --json output
        if flagJSON {
            return render.JSON(os.Stdout, note)
        }

        // Human output to stderr
        fmt.Fprintf(os.Stderr, "Done: %s\n", note.Title)
        return nil
    },
}

func init() {
    // Register with root command
    rootCmd.AddCommand(myCmd)
}
```

### 2. Key conventions

- **Register in `init()`** with `rootCmd.AddCommand()`
- **Use `db`** (the global `*store.Store`) for data access
- **Support `--json`** via the global `flagJSON` and `render.JSON()`
- **Use `resolveNote()`** from `cmd/helpers.go` for ID prefix matching
- **Human output to stderr**, data to stdout — this keeps `--json` output clean for piping
- **Use `buildNoteFilter()`** for commands that support tag/context/limit filtering
- **Use `addBulkFlags()`** for commands that support bulk operations
- **Use `confirmBulk()`** for destructive bulk operations

---

## Key invariants

These must be maintained across all changes:

### FTS sync

Every note mutation (create, update, delete) must update the `notes_fts` table in the same transaction. The store handles this internally — never write to `notes` without going through the Store API.

### Sync triggers

The `sync_changelog` table is populated by database triggers on `notes` and `tags`. These fire automatically. After importing notes via sync pull, the triggered changelog entries are cleared to prevent re-exporting them.

### Prefix ID resolution

`resolveNote()` tries an exact match first, then falls back to prefix matching across all notes. If a prefix is ambiguous (matches multiple notes), it returns an error. This is used consistently across all commands.

### TUI I/O

All I/O in the TUI happens through `tea.Cmd` functions — never block the main update loop. Database calls, file operations, and anything that could block must be wrapped in a `tea.Cmd` that returns a message.

### Styles in theme only

All TUI colours and `lipgloss.Style` definitions live in `internal/tui/theme/`. View files import the theme package — they never define local colour constants.

### Tags are additive and idempotent

Adding a tag that already exists is a no-op (enforced by the `UNIQUE(note_id, key, value)` constraint with `INSERT OR IGNORE`). Tags are never silently overwritten.

---

## macOS install gotcha

On Apple Silicon, copying a binary over an existing one (`cp new old`) invalidates the ad-hoc code signature, causing the kernel to `SIGKILL` the process on launch. The Makefile handles this by removing the old binary before copying:

```makefile
install: build
    mkdir -p $(INSTALL_DIR)
    rm -f $(INSTALL_DIR)/jot-cli
    cp $(BIN) $(INSTALL_DIR)/jot-cli
    ln -sf jot-cli $(INSTALL_DIR)/j
```

If you write custom install scripts, always `rm` before `cp`.

---

## Linting

```bash
make lint
```

Uses `golangci-lint`. Fix any warnings before submitting.

---

## Commit conventions

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add bulk unpin command
fix: handle empty body in FTS sync
refactor: extract date parsing to helpers
test: add search edge cases for special characters
chore: update dependencies
```

Keep commits atomic and focused — one logical change per commit.
