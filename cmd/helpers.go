package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/danjdewhurst/jot-cli/internal/model"
)

// resolveNote finds a note by full or prefix ID.
func resolveNote(idOrPrefix string) (model.Note, error) {
	// Try exact match first
	note, err := db.GetNote(idOrPrefix)
	if err == nil {
		return note, nil
	}

	// Try prefix match
	notes, err := db.ListNotes(model.NoteFilter{Archived: true})
	if err != nil {
		return model.Note{}, fmt.Errorf("listing notes: %w", err)
	}

	var matches []model.Note
	for _, n := range notes {
		if strings.HasPrefix(n.ID, idOrPrefix) {
			matches = append(matches, n)
		}
	}

	switch len(matches) {
	case 0:
		return model.Note{}, fmt.Errorf("no note matching %q", idOrPrefix)
	case 1:
		return matches[0], nil
	default:
		return model.Note{}, fmt.Errorf("ambiguous prefix %q matches %d notes", idOrPrefix, len(matches))
	}
}

// parseDate parses a date string in RFC 3339 or YYYY-MM-DD format.
// Returns zero time for empty input.
func parseDate(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("cannot parse %q as RFC 3339 or YYYY-MM-DD", s)
}

func filterByDateRange(notes []model.Note, since, until time.Time) []model.Note {
	if since.IsZero() && until.IsZero() {
		return notes
	}
	var filtered []model.Note
	for _, n := range notes {
		if !since.IsZero() && n.CreatedAt.Before(since) {
			continue
		}
		if !until.IsZero() && n.CreatedAt.After(until) {
			continue
		}
		filtered = append(filtered, n)
	}
	return filtered
}
