package store

import (
	"fmt"
	"time"

	"github.com/danjdewhurst/jot-cli/internal/model"
)

// ImportNote inserts a note with an explicit ID and timestamps.
// If a note with the same ID already exists, it is skipped (no error).
// Returns true if the note was created, false if it was skipped.
func (s *Store) ImportNote(n model.Note) (bool, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return false, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback after commit is a no-op

	archivedInt := 0
	if n.Archived {
		archivedInt = 1
	}
	pinnedInt := 0
	if n.Pinned {
		pinnedInt = 1
	}

	res, err := tx.Exec(
		"INSERT OR IGNORE INTO notes (id, title, body, created_at, updated_at, archived, pinned) VALUES (?, ?, ?, ?, ?, ?, ?)",
		n.ID, n.Title, n.Body,
		n.CreatedAt.Format(time.RFC3339), n.UpdatedAt.Format(time.RFC3339),
		archivedInt, pinnedInt,
	)
	if err != nil {
		return false, fmt.Errorf("inserting note: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("checking rows affected: %w", err)
	}
	if rows == 0 {
		return false, nil // duplicate ID — skip
	}

	if err := insertTags(tx, n.ID, n.Tags); err != nil {
		return false, err
	}

	if err := syncFTS(tx, n.ID); err != nil {
		return false, err
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit: %w", err)
	}

	return true, nil
}
