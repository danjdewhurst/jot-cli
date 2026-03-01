# Sync Guide

jot-cli can synchronise notes between machines using any shared folder — Dropbox, Syncthing, iCloud Drive, a mounted network share, or anything that syncs files.

---

## Overview

Sync is file-based and append-only. Each machine writes its changes as NDJSON changeset files to a shared directory. On pull, changesets from other machines are read and applied locally. There is no central server.

---

## Setup

### Initialise

```bash
j sync init
```

This validates the sync directory path and optionally sets up encryption. The `changesets/` subdirectory is created automatically on first push or pull. Run this once on each machine.

### Configure the sync directory

The sync directory defaults to `~/.local/share/jot/sync/` (respects `XDG_DATA_HOME`). Point it at a shared folder using any of these methods (in precedence order):

1. **CLI flag:** `j sync --sync-dir ~/Dropbox/jot-sync`
2. **Environment variable:** `export JOT_SYNC_DIR=~/Dropbox/jot-sync`
3. **Config file:**
   ```toml
   [sync]
   dir = "~/Dropbox/jot-sync"
   ```

The directory must be the same shared folder on all machines.

---

## Usage

### Full sync (push then pull)

```bash
j sync
```

Pushes local changes, then pulls remote changes. This is the most common operation.

### Push only

```bash
j sync push
```

Writes a changeset file containing all local changes since the last sync.

### Pull only

```bash
j sync pull
```

Reads and applies changeset files from other machines.

### Check status

```bash
j sync status
```

Shows the number of pending (unsynced) local changes and the last sync timestamp.

### JSON output

All sync commands support `--json`:

```bash
j sync --json          # {"pushed": 3, "pulled": 5, "conflicts": 0, "synced_at": "..."}
j sync push --json     # {"pushed": 3}
j sync pull --json     # {"pulled": 5, "conflicts": 0}
j sync status --json   # {"pending": 2, "last_sync": "..."}
```

---

## Encryption

Sync supports optional encryption using [age](https://age-encryption.org/) with scrypt passphrase-based encryption.

### Enable encryption

```bash
j sync init --encrypt
```

This generates 32 bytes of random data, stored as a 64-character hex string in `~/.local/share/jot/sync.key` (respects `XDG_DATA_HOME`). The identity file is created with `0600` permissions.

After enabling, all new changeset files are written with `.ndjson.age` extension instead of `.ndjson`. Existing unencrypted changesets remain readable.

### Share between machines

Copy the identity file (`sync.key`) to each machine that participates in sync. It must be at the same relative path — `~/.local/share/jot/sync.key`.

Then on each additional machine:

```bash
j sync init --encrypt
```

Replace the generated `sync.key` with the one from your first machine (or copy it before running init).

### Migrate existing changesets

If you enabled encryption after already having unencrypted changesets:

```bash
j sync migrate --encrypt
```

This reads each plain `.ndjson` file, encrypts it, writes a `.ndjson.age` replacement, and removes the original. The command requires encryption to already be enabled (`sync init --encrypt` first).

### How it works internally

- **Key file:** A random 64-character hex string stored in `sync.key`
- **Algorithm:** age scrypt encryption (passphrase-based)
- **Encrypted files:** `.ndjson.age` extension
- **Plain files:** `.ndjson` extension (still readable even when encryption is enabled)
- **Encryption scope:** Only changeset files in the sync directory are encrypted. The local database is not encrypted.

---

## Conflict resolution

### Last-write-wins

When the same note has been edited on multiple machines, conflicts are resolved by comparing `updated_at` timestamps:

- **Remote is newer** → remote version overwrites local
- **Local is newer** → local version is kept
- **Timestamps are equal** → deterministic tiebreaker using SHA-256 hash of the note body. The lower hash wins, ensuring both machines converge to the same version regardless of pull order.

### Delete conflicts

When one machine deletes a note and another edits it:

- If the delete timestamp is after the local note's `updated_at`, the note is deleted
- If the local note was edited after the delete, it survives

### Conflict reporting

Conflicts are counted and reported:

```
Synced: pushed 3, pulled 5 (2 conflicts — local version kept)
```

In JSON output, the `conflicts` field shows the count.

---

## Changeset format

Changesets are NDJSON (newline-delimited JSON) files stored in `<sync-dir>/changesets/`.

### File naming

```
<machine_id>_<RFC3339_timestamp>.ndjson
<machine_id>_<RFC3339_timestamp>.ndjson.age   # encrypted
```

- **Machine ID:** A ULID generated on first sync and stored in the database (`sync_meta` table). Each machine has a unique, stable ID.
- **Timestamp:** RFC 3339 format, used to skip already-processed files on pull.

### Entry format

Each line is a JSON object:

**Upsert (create or update):**

```json
{"action":"upsert","note":{"id":"...","title":"...","body":"...","created_at":"...","updated_at":"...","archived":false,"pinned":false,"tags":[...]}}
```

**Delete:**

```json
{"action":"delete","note_id":"...","deleted_at":"..."}
```

### Deduplication

On push, if the same note has multiple changelog entries (e.g. created then immediately edited), only the latest entry is included in the changeset.

### Atomicity

Changeset files are written atomically: content is written to a `.tmp` file, fsynced, then renamed to the final path. This prevents corrupt changesets if the process crashes mid-write.

---

## Troubleshooting

### "another jot sync is already running"

Sync uses an advisory file lock (`<sync-dir>/.lock`) via `flock(2)` to prevent concurrent sync operations. This error means another `jot sync` process holds the lock.

If the process crashed without releasing the lock, the lock is automatically released when the file descriptor is closed — which happens on process exit. Simply retry.

### Stale changesets

Old changeset files in the sync directory are harmless — they are skipped based on the `last_sync` timestamp stored in each machine's database. You can safely delete old `.ndjson` / `.ndjson.age` files if disk space is a concern.

### Encryption key issues

- **"loading sync identity" error:** The `sync.key` file is missing or empty. Copy it from another machine or re-run `j sync init --encrypt`.
- **"decrypting changeset" error:** The `sync.key` doesn't match the one used to encrypt. Ensure all machines use the same identity file.
- **Mixed encrypted/plain files:** This is fine. jot reads both `.ndjson` and `.ndjson.age` files. Enabling encryption only affects new writes.

### Changes not appearing on pull

1. Check `j sync status` — are there pending changes to push on the source machine?
2. Verify both machines point to the same sync directory
3. Ensure the file sync service (Dropbox, Syncthing, etc.) has finished syncing the changeset files

---

## Platform notes

- **File locking** uses `flock(2)` via `golang.org/x/sys/unix` — this is Unix-only (macOS, Linux). Windows is not currently supported for sync.
- **Maximum line size** for NDJSON parsing is 1 MB per entry.
