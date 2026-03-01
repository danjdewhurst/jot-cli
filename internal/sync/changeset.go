package sync

import "github.com/danjdewhurst/jot-cli/internal/model"

// ChangeEntry represents a single change in a changeset file.
type ChangeEntry struct {
	Action    string      `json:"action"`               // "upsert" or "delete"
	Note      *model.Note `json:"note,omitempty"`        // full note for upserts
	NoteID    string      `json:"note_id,omitempty"`     // just the ID for deletes
	DeletedAt string      `json:"deleted_at,omitempty"`  // timestamp for deletes
}
