package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/danjdewhurst/jot-cli/internal/model"
)

// ErrVersionNotFound is returned when a requested version does not exist.
var ErrVersionNotFound = errors.New("version not found")

// saveVersion snapshots the current note state into note_versions.
// Call within a transaction, before overwriting the note.
func saveVersion(tx *sql.Tx, noteID string) error {
	var title, body string
	err := tx.QueryRow("SELECT title, body FROM notes WHERE id = ?", noteID).Scan(&title, &body)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("note %q: %w", noteID, ErrNoteNotFound)
	}
	if err != nil {
		return fmt.Errorf("reading note for version snapshot: %w", err)
	}

	var nextVersion int
	err = tx.QueryRow(
		"SELECT COALESCE(MAX(version), 0) + 1 FROM note_versions WHERE note_id = ?", noteID,
	).Scan(&nextVersion)
	if err != nil {
		return fmt.Errorf("calculating next version: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = tx.Exec(
		"INSERT INTO note_versions (note_id, title, body, created_at, version) VALUES (?, ?, ?, ?, ?)",
		noteID, title, body, now, nextVersion,
	)
	if err != nil {
		return fmt.Errorf("inserting version: %w", err)
	}

	return nil
}

// ListVersions returns all versions for a note, newest first.
func (s *Store) ListVersions(noteID string) ([]model.NoteVersion, error) {
	rows, err := s.db.Query(
		"SELECT id, note_id, title, body, created_at, version FROM note_versions WHERE note_id = ? ORDER BY version DESC",
		noteID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing versions: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var versions []model.NoteVersion
	for rows.Next() {
		var v model.NoteVersion
		var createdAt string
		if err := rows.Scan(&v.ID, &v.NoteID, &v.Title, &v.Body, &createdAt, &v.Version); err != nil {
			return nil, fmt.Errorf("scanning version: %w", err)
		}
		v.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("parsing version created_at: %w", err)
		}
		versions = append(versions, v)
	}
	return versions, rows.Err()
}

// GetVersion returns a specific version of a note.
func (s *Store) GetVersion(noteID string, version int) (model.NoteVersion, error) {
	var v model.NoteVersion
	var createdAt string

	err := s.db.QueryRow(
		"SELECT id, note_id, title, body, created_at, version FROM note_versions WHERE note_id = ? AND version = ?",
		noteID, version,
	).Scan(&v.ID, &v.NoteID, &v.Title, &v.Body, &createdAt, &v.Version)
	if errors.Is(err, sql.ErrNoRows) {
		return model.NoteVersion{}, fmt.Errorf("note %q version %d: %w", noteID, version, ErrVersionNotFound)
	}
	if err != nil {
		return model.NoteVersion{}, fmt.Errorf("querying version: %w", err)
	}

	v.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return model.NoteVersion{}, fmt.Errorf("parsing version created_at: %w", err)
	}

	return v, nil
}
