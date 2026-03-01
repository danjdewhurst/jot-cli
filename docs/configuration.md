# Configuration Reference

jot-cli works with zero configuration. Everything below is optional.

---

## Config file

The config file lives at `~/.config/jot/config.toml` (or `$XDG_CONFIG_HOME/jot/config.toml`).

```bash
j config init     # create a default config file with comments
j config          # print resolved config (all sources merged)
j config --path   # print config file path
```

### Full TOML reference

```toml
[general]
# Editor override — used instead of $VISUAL/$EDITOR for jot only.
# Default: "" (uses $VISUAL, then $EDITOR, then vi)
editor = "nvim"

# Default --limit for list and log commands. 0 means unlimited.
# Default: 0
default_limit = 20

[display]
# Timestamp display format: "relative", "absolute", or "iso".
# Default: "relative"
date_format = "relative"

# Default to JSON output (same as JOT_JSON=1 or --json).
# Default: false
json = false

[sync]
# Sync directory path. Supports ~ expansion.
# Default: ~/.local/share/jot/sync/ (respects XDG_DATA_HOME)
dir = "~/Dropbox/jot-sync"
```

---

## Environment variables

| Variable | Description | Example |
|----------|-------------|---------|
| `JOT_DB` | Override database path | `JOT_DB=/tmp/test.db` |
| `JOT_SYNC_DIR` | Override sync directory | `JOT_SYNC_DIR=~/Dropbox/jot` |
| `JOT_JSON` | Default to JSON output (`1` or `true`) | `JOT_JSON=1` |
| `VISUAL` | Preferred editor (checked before `EDITOR`) | `VISUAL=code --wait` |
| `EDITOR` | Fallback editor | `EDITOR=nvim` |
| `NO_COLOR` | Disable colour output (any value) | `NO_COLOR=1` |
| `XDG_CONFIG_HOME` | Override config directory base | `XDG_CONFIG_HOME=~/.config` |
| `XDG_DATA_HOME` | Override data directory base | `XDG_DATA_HOME=~/.local/share` |

---

## Storage paths

| Path | Purpose |
|------|---------|
| `~/.config/jot/config.toml` | Config file (respects `XDG_CONFIG_HOME`) |
| `~/.local/share/jot/jot.db` | SQLite database (respects `XDG_DATA_HOME`) |
| `~/.local/share/jot/sync/` | Sync directory (respects `XDG_DATA_HOME`) |
| `~/.local/share/jot/sync.key` | Encryption identity file (respects `XDG_DATA_HOME`) |

All paths create parent directories automatically on first use.

---

## Precedence

Configuration values are resolved in this order (highest wins):

1. **CLI flags** — `--json`, `--db`, `--sync-dir`, `--limit`, etc.
2. **Environment variables** — `JOT_DB`, `JOT_SYNC_DIR`, `JOT_JSON`
3. **Config file** — `~/.config/jot/config.toml`
4. **Defaults** — built-in values

For example, if `config.toml` sets `json = true` but you run `j list` without `--json`, you'll get JSON output. But if you set `--json=false` explicitly on the command line, that wins.

The editor resolution chain is: config `general.editor` → `$VISUAL` → `$EDITOR` → `vi`.

---

## Date formats

The `date_format` setting (and `render.DateFormat` internally) controls how timestamps appear in human-readable output:

| Value | Format | Example |
|-------|--------|---------|
| `relative` (default) | Human-readable relative time | `5m ago`, `3h ago`, `2d ago`, `1mo ago` |
| `absolute` | `YYYY-MM-DD HH:MM` | `2025-03-01 14:30` |
| `iso` | RFC 3339 | `2025-03-01T14:30:00Z` |

Date flags (`--since`, `--until`) accept two input formats:

- **ISO / RFC 3339:** `2025-03-01T14:30:00Z`
- **Date only:** `2025-03-01` (interpreted as midnight UTC)

---

## Global CLI flags

These flags are available on every command:

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON |
| `--db <path>` | Override database path |
| `--verbose` | Verbose output (shows unresolvable `@ref` warnings) |
