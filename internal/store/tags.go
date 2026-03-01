package store

import (
	"fmt"

	"github.com/duncanjbrown/jot-cli/internal/model"
)

func (s *Store) AddTag(noteID string, tag model.Tag) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		"INSERT OR IGNORE INTO tags (note_id, key, value) VALUES (?, ?, ?)",
		noteID, tag.Key, tag.Value,
	)
	if err != nil {
		return fmt.Errorf("inserting tag: %w", err)
	}

	if err := syncFTS(tx, noteID); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) RemoveTag(noteID string, tag model.Tag) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		"DELETE FROM tags WHERE note_id = ? AND key = ? AND value = ?",
		noteID, tag.Key, tag.Value,
	)
	if err != nil {
		return fmt.Errorf("removing tag: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return fmt.Errorf("tag %s not found on note %q", tag, noteID)
	}

	if err := syncFTS(tx, noteID); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) ListTags(key string) ([]model.Tag, error) {
	query := "SELECT DISTINCT key, value FROM tags"
	var args []any
	if key != "" {
		query += " WHERE key = ?"
		args = append(args, key)
	}
	query += " ORDER BY key, value"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing tags: %w", err)
	}
	defer rows.Close()

	var tags []model.Tag
	for rows.Next() {
		var t model.Tag
		if err := rows.Scan(&t.Key, &t.Value); err != nil {
			return nil, fmt.Errorf("scanning tag: %w", err)
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}
