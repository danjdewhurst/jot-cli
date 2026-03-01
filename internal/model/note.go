package model

import "time"

type Note struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Archived  bool      `json:"archived"`
	Pinned    bool      `json:"pinned"`
	Tags      []Tag     `json:"tags,omitempty"`
}

type NoteFilter struct {
	Tags       []Tag
	Archived   bool
	PinnedOnly bool
	Limit      int
	Offset     int
	Since      *time.Time // only notes created at or after this time
	Until      *time.Time // only notes created before this time
	SortAsc    bool       // if true, order by created_at ASC (oldest first)
}
