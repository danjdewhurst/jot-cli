package cmd

import (
	"fmt"
	"strings"

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
