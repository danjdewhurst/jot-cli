# Plan: Encrypted sync

**Priority:** Bigger bet

## Summary

Encrypt changeset files before writing them to the sync directory, so notes remain private when syncing via cloud storage (Dropbox, iCloud, etc.).

## Approach

Use **age** (filippo.io/age) for encryption — modern, simple, no config. Each changeset file is encrypted with a passphrase or age identity.

## Behaviour

- `jot sync init --encrypt` — Set up encryption, generate or import an age identity
- Encrypted sync is opt-in; unencrypted sync remains the default
- Push: encrypt changeset JSON → write `.age` files to sync dir
- Pull: decrypt `.age` files → process as normal
- Key stored in `$XDG_DATA_HOME/jot/identity.age` (never synced)
- Passphrase-based alternative for simpler setup

## Config

```toml
[sync]
encrypt = true
identity = "~/.local/share/jot/identity.age"
```

## Implementation

### Dependencies

- `filippo.io/age` — Pure Go, no CGo, well-maintained

### Sync layer

- `internal/sync/crypto.go` — Encrypt/decrypt wrappers
- Modify `Push()` to encrypt before writing
- Modify `Pull()` to detect and decrypt `.age` files
- Fall back to reading unencrypted files (migration path)

### Key management

- `internal/sync/identity.go` — Generate, load, and store age identities
- First-time setup wizard via `jot sync init --encrypt`
- Support both identity file and passphrase (scrypt-based)

### Migration

- Existing unencrypted sync dirs continue to work
- `jot sync migrate --encrypt` — Re-encrypt existing changesets

## Security considerations

- Identity file permissions: `0600`
- Never log or display key material
- Passphrase prompt uses `golang.org/x/term` (already a dependency)
- No key escrow — user is responsible for backup

## Complexity

~500 lines. New dependency, crypto wrappers, key management, sync layer modifications.
