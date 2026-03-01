package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/duncanjbrown/jot-cli/internal/model"
	"github.com/oklog/ulid/v2"
)

func (s *Store) CreateNote(title, body string, tags []model.Tag) (model.Note, error) {
	now := time.Now().UTC()
	id := ulid.Make().String()

	tx, err := s.db.Begin()
	if err != nil {
		return model.Note{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		"INSERT INTO notes (id, title, body, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		id, title, body, now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	if err != nil {
		return model.Note{}, fmt.Errorf("inserting note: %w", err)
	}

	if err := insertTags(tx, id, tags); err != nil {
		return model.Note{}, err
	}

	if err := syncFTS(tx, id); err != nil {
		return model.Note{}, err
	}

	if err := tx.Commit(); err != nil {
		return model.Note{}, fmt.Errorf("commit: %w", err)
	}

	return model.Note{
		ID:        id,
		Title:     title,
		Body:      body,
		CreatedAt: now,
		UpdatedAt: now,
		Tags:      tags,
	}, nil
}

func (s *Store) GetNote(id string) (model.Note, error) {
	var n model.Note
	var createdAt, updatedAt string
	var archived int

	err := s.db.QueryRow(
		"SELECT id, title, body, created_at, updated_at, archived FROM notes WHERE id = ?", id,
	).Scan(&n.ID, &n.Title, &n.Body, &createdAt, &updatedAt, &archived)
	if err == sql.ErrNoRows {
		return model.Note{}, fmt.Errorf("note %q not found", id)
	}
	if err != nil {
		return model.Note{}, fmt.Errorf("querying note: %w", err)
	}

	n.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	n.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	n.Archived = archived != 0

	tags, err := s.getTagsForNote(n.ID)
	if err != nil {
		return model.Note{}, err
	}
	n.Tags = tags

	return n, nil
}

func (s *Store) UpdateNote(id, title, body string) (model.Note, error) {
	now := time.Now().UTC()

	tx, err := s.db.Begin()
	if err != nil {
		return model.Note{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		"UPDATE notes SET title = ?, body = ?, updated_at = ? WHERE id = ?",
		title, body, now.Format(time.RFC3339), id,
	)
	if err != nil {
		return model.Note{}, fmt.Errorf("updating note: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return model.Note{}, fmt.Errorf("note %q not found", id)
	}

	if err := syncFTS(tx, id); err != nil {
		return model.Note{}, err
	}

	if err := tx.Commit(); err != nil {
		return model.Note{}, fmt.Errorf("commit: %w", err)
	}

	return s.GetNote(id)
}

func (s *Store) ArchiveNote(id string) error {
	res, err := s.db.Exec("UPDATE notes SET archived = 1, updated_at = ? WHERE id = ?",
		time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("archiving note: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return fmt.Errorf("note %q not found", id)
	}
	return nil
}

func (s *Store) DeleteNote(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Delete FTS entry first
	if _, err := tx.Exec("DELETE FROM notes_fts WHERE note_id = ?", id); err != nil {
		return fmt.Errorf("deleting FTS entry: %w", err)
	}

	res, err := tx.Exec("DELETE FROM notes WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting note: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return fmt.Errorf("note %q not found", id)
	}

	return tx.Commit()
}

func (s *Store) ListNotes(filter model.NoteFilter) ([]model.Note, error) {
	query := "SELECT DISTINCT n.id, n.title, n.body, n.created_at, n.updated_at, n.archived FROM notes n"
	var args []any
	var conditions []string

	if len(filter.Tags) > 0 {
		query += " JOIN tags t ON t.note_id = n.id"
		for _, tag := range filter.Tags {
			conditions = append(conditions, "(t.key = ? AND t.value = ?)")
			args = append(args, tag.Key, tag.Value)
		}
	}

	if !filter.Archived {
		conditions = append(conditions, "n.archived = 0")
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY n.created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing notes: %w", err)
	}
	defer rows.Close()

	var notes []model.Note
	for rows.Next() {
		var n model.Note
		var createdAt, updatedAt string
		var archived int
		if err := rows.Scan(&n.ID, &n.Title, &n.Body, &createdAt, &updatedAt, &archived); err != nil {
			return nil, fmt.Errorf("scanning note: %w", err)
		}
		n.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		n.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		n.Archived = archived != 0

		tags, err := s.getTagsForNote(n.ID)
		if err != nil {
			return nil, err
		}
		n.Tags = tags
		notes = append(notes, n)
	}

	return notes, rows.Err()
}

func (s *Store) getTagsForNote(noteID string) ([]model.Tag, error) {
	rows, err := s.db.Query("SELECT key, value FROM tags WHERE note_id = ? ORDER BY key, value", noteID)
	if err != nil {
		return nil, fmt.Errorf("querying tags: %w", err)
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

func insertTags(tx *sql.Tx, noteID string, tags []model.Tag) error {
	for _, t := range tags {
		_, err := tx.Exec(
			"INSERT OR IGNORE INTO tags (note_id, key, value) VALUES (?, ?, ?)",
			noteID, t.Key, t.Value,
		)
		if err != nil {
			return fmt.Errorf("inserting tag: %w", err)
		}
	}
	return nil
}

// syncFTS upserts the FTS entry for a note. Call within a transaction.
func syncFTS(tx *sql.Tx, noteID string) error {
	// Gather tag strings
	rows, err := tx.Query("SELECT key, value FROM tags WHERE note_id = ?", noteID)
	if err != nil {
		return fmt.Errorf("querying tags for FTS sync: %w", err)
	}
	defer rows.Close()

	var parts []string
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return fmt.Errorf("scanning tag for FTS: %w", err)
		}
		parts = append(parts, k+":"+v)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	tagStr := strings.Join(parts, " ")

	// Get current title/body
	var title, body string
	err = tx.QueryRow("SELECT title, body FROM notes WHERE id = ?", noteID).Scan(&title, &body)
	if err != nil {
		return fmt.Errorf("getting note for FTS: %w", err)
	}

	// Delete existing FTS row, then insert fresh
	_, err = tx.Exec("DELETE FROM notes_fts WHERE note_id = ?", noteID)
	if err != nil {
		return fmt.Errorf("deleting FTS entry: %w", err)
	}

	_, err = tx.Exec("INSERT INTO notes_fts(note_id, title, body, tags) VALUES (?, ?, ?, ?)",
		noteID, title, body, tagStr)
	if err != nil {
		return fmt.Errorf("inserting FTS entry: %w", err)
	}

	return nil
}
