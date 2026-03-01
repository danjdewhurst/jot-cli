package model

import "time"

type Note struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Archived  bool      `json:"archived"`
	Tags      []Tag     `json:"tags,omitempty"`
}

type NoteFilter struct {
	Tags     []Tag
	Archived bool
	Limit    int
	Offset   int
}
