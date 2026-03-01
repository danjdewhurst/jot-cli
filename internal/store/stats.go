package store

import (
	"fmt"
	"time"
)

// TagCount represents a tag and how many notes use it.
type TagCount struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Count int    `json:"count"`
}

// NoteStats holds aggregate statistics about notes.
type NoteStats struct {
	TotalNotes    int       `json:"total_notes"`
	ArchivedNotes int       `json:"archived_notes"`
	PinnedNotes   int       `json:"pinned_notes"`
	UniqueTags    int       `json:"unique_tags"`
	TopTags       []TagCount `json:"top_tags"`
	WeeklyCount   int       `json:"weekly_count"`
	MonthlyCount  int       `json:"monthly_count"`
	OldestDate    time.Time `json:"oldest_date"`
	NewestDate    time.Time `json:"newest_date"`
}

// Stats returns aggregate statistics about all notes.
func (s *Store) Stats() (NoteStats, error) {
	var st NoteStats

	// Note counts
	err := s.db.QueryRow(`
		SELECT
			COUNT(*),
			COUNT(CASE WHEN archived = 1 THEN 1 END),
			COUNT(CASE WHEN pinned = 1 AND archived = 0 THEN 1 END)
		FROM notes
	`).Scan(&st.TotalNotes, &st.ArchivedNotes, &st.PinnedNotes)
	if err != nil {
		return NoteStats{}, fmt.Errorf("querying note counts: %w", err)
	}

	if st.TotalNotes == 0 {
		return st, nil
	}

	// Unique tags
	err = s.db.QueryRow(`
		SELECT COUNT(DISTINCT key || ':' || value) FROM tags
	`).Scan(&st.UniqueTags)
	if err != nil {
		return NoteStats{}, fmt.Errorf("querying unique tags: %w", err)
	}

	// Top tags (up to 5)
	rows, err := s.db.Query(`
		SELECT key, value, COUNT(*) as cnt
		FROM tags
		GROUP BY key, value
		ORDER BY cnt DESC, key, value
		LIMIT 5
	`)
	if err != nil {
		return NoteStats{}, fmt.Errorf("querying top tags: %w", err)
	}
	defer rows.Close() //nolint:errcheck // rows.Close error is checked via rows.Err

	for rows.Next() {
		var tc TagCount
		if err := rows.Scan(&tc.Key, &tc.Value, &tc.Count); err != nil {
			return NoteStats{}, fmt.Errorf("scanning top tag: %w", err)
		}
		st.TopTags = append(st.TopTags, tc)
	}
	if err := rows.Err(); err != nil {
		return NoteStats{}, fmt.Errorf("iterating top tags: %w", err)
	}

	// Weekly and monthly counts
	now := time.Now().UTC()
	weekAgo := now.AddDate(0, 0, -7).Format(time.RFC3339)
	monthAgo := now.AddDate(0, -1, 0).Format(time.RFC3339)

	err = s.db.QueryRow(`
		SELECT
			COUNT(CASE WHEN created_at >= ? THEN 1 END),
			COUNT(CASE WHEN created_at >= ? THEN 1 END)
		FROM notes
	`, weekAgo, monthAgo).Scan(&st.WeeklyCount, &st.MonthlyCount)
	if err != nil {
		return NoteStats{}, fmt.Errorf("querying activity counts: %w", err)
	}

	// Oldest and newest dates
	var oldest, newest string
	err = s.db.QueryRow(`
		SELECT MIN(created_at), MAX(created_at) FROM notes
	`).Scan(&oldest, &newest)
	if err != nil {
		return NoteStats{}, fmt.Errorf("querying date range: %w", err)
	}

	st.OldestDate, err = time.Parse(time.RFC3339, oldest)
	if err != nil {
		return NoteStats{}, fmt.Errorf("parsing oldest date: %w", err)
	}
	st.NewestDate, err = time.Parse(time.RFC3339, newest)
	if err != nil {
		return NoteStats{}, fmt.Errorf("parsing newest date: %w", err)
	}

	return st, nil
}
