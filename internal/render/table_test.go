package render_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/danjdewhurst/jot-cli/internal/model"
	"github.com/danjdewhurst/jot-cli/internal/render"
)

func TestNoteDetail_WithBacklinks(t *testing.T) {
	now := time.Now()
	note := model.Note{
		ID:        "01ABC123",
		Title:     "Test Note",
		Body:      "Hello world",
		CreatedAt: now,
		UpdatedAt: now,
	}

	backlinks := []model.Note{
		{ID: "01DEF456", Title: "Linking Note"},
		{ID: "01GHI789", Title: "Another Linker"},
	}

	var buf bytes.Buffer
	render.NoteDetail(&buf, note, backlinks)
	output := buf.String()

	if !strings.Contains(output, "Referenced by") {
		t.Error("expected 'Referenced by' section in output")
	}
	if !strings.Contains(output, "01DEF456") {
		t.Error("expected backlink note ID in output")
	}
	if !strings.Contains(output, "Linking Note") {
		t.Error("expected backlink note title in output")
	}
	if !strings.Contains(output, "01GHI789") {
		t.Error("expected second backlink note ID in output")
	}
}

func TestNoteDetail_NoBacklinks(t *testing.T) {
	now := time.Now()
	note := model.Note{
		ID:        "01ABC123",
		Title:     "Test Note",
		Body:      "Hello world",
		CreatedAt: now,
		UpdatedAt: now,
	}

	var buf bytes.Buffer
	render.NoteDetail(&buf, note, nil)
	output := buf.String()

	if strings.Contains(output, "Referenced by") {
		t.Error("should not show 'Referenced by' section when there are no backlinks")
	}
}
