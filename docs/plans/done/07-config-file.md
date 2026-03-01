# Plan: Configuration file

**Priority:** Medium effort, high value

## Summary

Add a TOML config file at `~/.config/jot/config.toml` (XDG-compliant) for persistent settings. Env vars override config file values.

## Config options (v1)

```toml
[general]
editor = "nvim"           # Override $EDITOR for jot only
default_limit = 20        # Default --limit for list/log

[display]
theme = "frappe"          # Future: support multiple Catppuccin flavours
date_format = "relative"  # "relative" | "absolute" | "iso"
json = false              # Default to JSON output

[sync]
dir = "~/Dropbox/jot-sync"
auto = false              # Future: auto-sync on add/edit
```

## Precedence

1. CLI flags (highest)
2. Environment variables (`JOT_DB`, `JOT_SYNC_DIR`, `JOT_JSON`)
3. Config file
4. Defaults (lowest)

## Implementation

### Config layer

- Extend `internal/config/` to read TOML file
- Use `BurntSushi/toml` or similar lightweight TOML parser
- `Load()` reads file → merges with env vars → returns `Config` struct
- Add new fields to `Config` struct

### XDG paths

- Config: `$XDG_CONFIG_HOME/jot/config.toml` (default `~/.config/jot/config.toml`)
- Data: unchanged (`$XDG_DATA_HOME/jot/`)

### CLI

- `jot config` — Print resolved config (all sources merged)
- `jot config --path` — Print config file path
- `jot config init` — Create a default config file with comments

### Integration

- `cmd/root.go` — Load config early, pass to subcommands
- Thread config values through to editor, render, and store layers

## Complexity

~300 lines. New dependency (TOML parser), config layer refactor, new command.
