package store

import (
	"fmt"
	"strings"
	"time"

	"github.com/danjdewhurst/jot-cli/internal/model"
)

type SearchResult struct {
	Note    model.Note `json:"note"`
	Snippet string     `json:"snippet,omitempty"`
	Rank    float64    `json:"rank"`
}

func (s *Store) Search(query string, tags []model.Tag) ([]SearchResult, error) {
	q := `SELECT n.id, n.title, n.body, n.created_at, n.updated_at, n.archived, n.pinned,
	             snippet(notes_fts, 2, '<mark>', '</mark>', '…', 32) as snippet,
	             notes_fts.rank
	      FROM notes_fts
	      JOIN notes n ON n.id = notes_fts.note_id`

	var conditions []string
	var args []any

	conditions = append(conditions, "notes_fts MATCH ?")
	args = append(args, query)

	if len(tags) > 0 {
		q += " JOIN tags t ON t.note_id = n.id"
		for _, tag := range tags {
			conditions = append(conditions, "(t.key = ? AND t.value = ?)")
			args = append(args, tag.Key, tag.Value)
		}
	}

	conditions = append(conditions, "n.archived = 0")
	q += " WHERE " + strings.Join(conditions, " AND ")
	q += " ORDER BY notes_fts.rank"

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("searching notes: %w", err)
	}
	defer rows.Close() //nolint:errcheck // rows.Close error is checked via rows.Err

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var createdAt, updatedAt string
		var archived, pinned int
		if err := rows.Scan(
			&r.Note.ID, &r.Note.Title, &r.Note.Body,
			&createdAt, &updatedAt, &archived, &pinned,
			&r.Snippet, &r.Rank,
		); err != nil {
			return nil, fmt.Errorf("scanning search result: %w", err)
		}
		r.Note.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		r.Note.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		r.Note.Archived = archived != 0
		r.Note.Pinned = pinned != 0

		noteTags, err := s.getTagsForNote(r.Note.ID)
		if err != nil {
			return nil, err
		}
		r.Note.Tags = noteTags
		results = append(results, r)
	}

	return results, rows.Err()
}
