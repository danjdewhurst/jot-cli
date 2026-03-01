package model

import "time"

// NoteVersion represents a historical snapshot of a note's content.
type NoteVersion struct {
	ID        int       `json:"id"`
	NoteID    string    `json:"note_id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	Version   int       `json:"version"`
}
