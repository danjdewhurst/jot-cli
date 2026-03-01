package store

import (
	"fmt"

	"github.com/danjdewhurst/jot-cli/internal/model"
)

// SyncRefs reconciles ref tags for a note. It removes stale ref tags and
// adds new ones, preserving all non-ref tags. refIDs should be fully
// resolved note IDs.
func (s *Store) SyncRefs(noteID string, refIDs []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback after commit is a no-op

	// Remove all existing ref tags for this note
	_, err = tx.Exec("DELETE FROM tags WHERE note_id = ? AND key = 'ref'", noteID)
	if err != nil {
		return fmt.Errorf("removing old ref tags: %w", err)
	}

	// Insert new ref tags
	for _, refID := range refIDs {
		_, err = tx.Exec(
			"INSERT OR IGNORE INTO tags (note_id, key, value) VALUES (?, 'ref', ?)",
			noteID, refID,
		)
		if err != nil {
			return fmt.Errorf("inserting ref tag: %w", err)
		}
	}

	if err := syncFTS(tx, noteID); err != nil {
		return err
	}

	return tx.Commit()
}

// ReferencesTo returns all notes that have a ref tag pointing to the given
// note ID (i.e. backlinks).
func (s *Store) ReferencesTo(noteID string) ([]model.Note, error) {
	rows, err := s.db.Query(
		"SELECT DISTINCT note_id FROM tags WHERE key = 'ref' AND value = ?",
		noteID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying backlinks: %w", err)
	}
	defer rows.Close() //nolint:errcheck // rows.Close error is checked via rows.Err

	var noteIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning backlink note ID: %w", err)
		}
		noteIDs = append(noteIDs, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Fetch full notes after closing cursor
	var notes []model.Note
	for _, id := range noteIDs {
		note, err := s.GetNote(id)
		if err != nil {
			continue // skip dangling references
		}
		notes = append(notes, note)
	}
	return notes, nil
}
