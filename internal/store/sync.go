package store

import (
	"fmt"
	"strings"
	"time"

	"github.com/danjdewhurst/jot-cli/internal/model"
)

// SyncChangelogEntry represents a row in the sync_changelog table.
type SyncChangelogEntry struct {
	ID        int64
	NoteID    string
	Action    string // "upsert" or "delete"
	ChangedAt string
}

// UnsyncedChanges returns changelog entries that haven't been synced yet.
func (s *Store) UnsyncedChanges() ([]SyncChangelogEntry, error) {
	rows, err := s.db.Query(
		"SELECT id, note_id, action, changed_at FROM sync_changelog WHERE synced = 0 ORDER BY id",
	)
	if err != nil {
		return nil, fmt.Errorf("querying unsynced changes: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var entries []SyncChangelogEntry
	for rows.Next() {
		var e SyncChangelogEntry
		if err := rows.Scan(&e.ID, &e.NoteID, &e.Action, &e.ChangedAt); err != nil {
			return nil, fmt.Errorf("scanning changelog entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// MarkChangesSynced marks changelog entries up to a given ID as synced.
func (s *Store) MarkChangesSynced(upToID int64) error {
	_, err := s.db.Exec("UPDATE sync_changelog SET synced = 1 WHERE synced = 0 AND id <= ?", upToID)
	if err != nil {
		return fmt.Errorf("marking changes synced: %w", err)
	}
	return nil
}

// ClearChangelogForNotes removes unsynced changelog entries for specific note IDs.
// Used after pulling to avoid re-exporting imported changes.
func (s *Store) ClearChangelogForNotes(noteIDs []string) error {
	if len(noteIDs) == 0 {
		return nil
	}
	placeholders := make([]string, len(noteIDs))
	args := make([]any, len(noteIDs))
	for i, id := range noteIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	query := fmt.Sprintf(
		"DELETE FROM sync_changelog WHERE synced = 0 AND note_id IN (%s)",
		strings.Join(placeholders, ","),
	)
	_, err := s.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("clearing changelog for imported notes: %w", err)
	}
	return nil
}

// GetSyncMeta reads a value from the sync_meta table.
func (s *Store) GetSyncMeta(key string) (string, error) {
	var value string
	err := s.db.QueryRow("SELECT value FROM sync_meta WHERE key = ?", key).Scan(&value)
	if err != nil {
		return "", fmt.Errorf("getting sync meta %q: %w", key, err)
	}
	return value, nil
}

// SetSyncMeta writes a value to the sync_meta table.
func (s *Store) SetSyncMeta(key, value string) error {
	_, err := s.db.Exec(
		"INSERT INTO sync_meta (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value",
		key, value,
	)
	if err != nil {
		return fmt.Errorf("setting sync meta %q: %w", key, err)
	}
	return nil
}

// UpsertNote inserts or replaces a note with a specific ID, timestamps, and tags.
// Used by sync to import notes from other machines where conflict resolution
// has already determined the incoming version should win.
func (s *Store) UpsertNote(n model.Note) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	archivedInt := 0
	if n.Archived {
		archivedInt = 1
	}
	pinnedInt := 0
	if n.Pinned {
		pinnedInt = 1
	}

	_, err = tx.Exec(`
		INSERT INTO notes (id, title, body, created_at, updated_at, archived, pinned)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			body = excluded.body,
			created_at = excluded.created_at,
			updated_at = excluded.updated_at,
			archived = excluded.archived,
			pinned = excluded.pinned`,
		n.ID, n.Title, n.Body,
		n.CreatedAt.Format(time.RFC3339), n.UpdatedAt.Format(time.RFC3339),
		archivedInt, pinnedInt,
	)
	if err != nil {
		return fmt.Errorf("upserting note: %w", err)
	}

	// Replace all tags
	if _, err := tx.Exec("DELETE FROM tags WHERE note_id = ?", n.ID); err != nil {
		return fmt.Errorf("clearing tags: %w", err)
	}
	if err := insertTags(tx, n.ID, n.Tags); err != nil {
		return err
	}

	if err := syncFTS(tx, n.ID); err != nil {
		return err
	}

	return tx.Commit()
}
