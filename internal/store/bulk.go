package store

import (
	"fmt"
	"strings"
	"time"

	"github.com/danjdewhurst/jot-cli/internal/model"
)

// ArchiveNotes archives multiple notes in a single transaction.
// Returns the number of notes archived.
func (s *Store) ArchiveNotes(ids []string) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	now := time.Now().UTC().Format(time.RFC3339)
	placeholders := makePlaceholders(len(ids))
	args := make([]any, 0, len(ids)+1)
	args = append(args, now)
	for _, id := range ids {
		args = append(args, id)
	}

	query := fmt.Sprintf("UPDATE notes SET archived = 1, updated_at = ? WHERE id IN (%s)", placeholders)
	res, err := s.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("batch archive: %w", err)
	}

	rows, _ := res.RowsAffected()
	return int(rows), nil
}

// DeleteNotes permanently deletes multiple notes in a single transaction,
// cleaning up FTS entries. Tags are cascade-deleted by foreign keys.
// Returns the number of notes deleted.
func (s *Store) DeleteNotes(ids []string) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback after commit is a no-op

	placeholders := makePlaceholders(len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	// Delete FTS entries first
	ftsQuery := fmt.Sprintf("DELETE FROM notes_fts WHERE note_id IN (%s)", placeholders)
	if _, err := tx.Exec(ftsQuery, args...); err != nil {
		return 0, fmt.Errorf("deleting FTS entries: %w", err)
	}

	// Delete notes (cascade deletes tags)
	noteQuery := fmt.Sprintf("DELETE FROM notes WHERE id IN (%s)", placeholders)
	res, err := tx.Exec(noteQuery, args...)
	if err != nil {
		return 0, fmt.Errorf("deleting notes: %w", err)
	}

	rows, _ := res.RowsAffected()

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}

	return int(rows), nil
}

// AddTagToNotes adds a tag to multiple notes in a single transaction,
// syncing FTS for each. Returns the number of notes affected.
func (s *Store) AddTagToNotes(ids []string, tag model.Tag) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback after commit is a no-op

	count := 0
	for _, id := range ids {
		_, err := tx.Exec(
			"INSERT OR IGNORE INTO tags (note_id, key, value) VALUES (?, ?, ?)",
			id, tag.Key, tag.Value,
		)
		if err != nil {
			return 0, fmt.Errorf("inserting tag for note %s: %w", id, err)
		}
		if err := syncFTS(tx, id); err != nil {
			return 0, fmt.Errorf("syncing FTS for note %s: %w", id, err)
		}
		count++
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}

	return count, nil
}

// PinNotes pins multiple notes in a single transaction.
// Returns the number of notes pinned.
func (s *Store) PinNotes(ids []string) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	now := time.Now().UTC().Format(time.RFC3339)
	placeholders := makePlaceholders(len(ids))
	args := make([]any, 0, len(ids)+1)
	args = append(args, now)
	for _, id := range ids {
		args = append(args, id)
	}

	query := fmt.Sprintf("UPDATE notes SET pinned = 1, updated_at = ? WHERE id IN (%s)", placeholders)
	res, err := s.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("batch pin: %w", err)
	}

	rows, _ := res.RowsAffected()
	return int(rows), nil
}

// UnpinNotes unpins multiple notes in a single transaction.
// Returns the number of notes unpinned.
func (s *Store) UnpinNotes(ids []string) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	now := time.Now().UTC().Format(time.RFC3339)
	placeholders := makePlaceholders(len(ids))
	args := make([]any, 0, len(ids)+1)
	args = append(args, now)
	for _, id := range ids {
		args = append(args, id)
	}

	query := fmt.Sprintf("UPDATE notes SET pinned = 0, updated_at = ? WHERE id IN (%s)", placeholders)
	res, err := s.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("batch unpin: %w", err)
	}

	rows, _ := res.RowsAffected()
	return int(rows), nil
}

// makePlaceholders returns a comma-separated string of n SQL placeholders.
func makePlaceholders(n int) string {
	p := make([]string, n)
	for i := range p {
		p[i] = "?"
	}
	return strings.Join(p, ", ")
}
