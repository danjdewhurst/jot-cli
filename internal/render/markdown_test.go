package render_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/danjdewhurst/jot-cli/internal/model"
	"github.com/danjdewhurst/jot-cli/internal/render"
)

func TestMarkdownSingleNote(t *testing.T) {
	ts := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	notes := []model.Note{{
		ID:        "01JEXAMPLE00000000000000001",
		Title:     "Test Note",
		Body:      "Hello world",
		CreatedAt: ts,
		UpdatedAt: ts,
		Tags:      []model.Tag{{Key: "folder", Value: "work"}},
	}}

	var buf bytes.Buffer
	if err := render.Markdown(&buf, notes); err != nil {
		t.Fatalf("rendering: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "# Test Note") {
		t.Error("missing title heading")
	}
	if !strings.Contains(out, "**ID:** 01JEXAMPLE00000000000000001") {
		t.Error("missing ID line")
	}
	if !strings.Contains(out, "**Created:**") {
		t.Error("missing created line")
	}
	if !strings.Contains(out, "**Tags:** folder:work") {
		t.Error("missing tags line")
	}
	if !strings.Contains(out, "Hello world") {
		t.Error("missing body")
	}
}

func TestMarkdownMultipleNotes(t *testing.T) {
	ts := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	notes := []model.Note{
		{ID: "01J1", Title: "First", Body: "Body 1", CreatedAt: ts, UpdatedAt: ts},
		{ID: "01J2", Title: "Second", Body: "Body 2", CreatedAt: ts, UpdatedAt: ts},
	}

	var buf bytes.Buffer
	if err := render.Markdown(&buf, notes); err != nil {
		t.Fatalf("rendering: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "# First") {
		t.Error("missing first note heading")
	}
	if !strings.Contains(out, "# Second") {
		t.Error("missing second note heading")
	}
	if strings.Count(out, "---") != 1 {
		t.Errorf("expected 1 separator, got %d", strings.Count(out, "---"))
	}
}

func TestMarkdownEmptyTitle(t *testing.T) {
	ts := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	notes := []model.Note{{
		ID:        "01J1",
		Body:      "Body only",
		CreatedAt: ts,
		UpdatedAt: ts,
	}}

	var buf bytes.Buffer
	if err := render.Markdown(&buf, notes); err != nil {
		t.Fatalf("rendering: %v", err)
	}

	out := buf.String()
	if strings.Contains(out, "#") {
		t.Error("should not contain heading when title is empty")
	}
	if !strings.Contains(out, "Body only") {
		t.Error("missing body")
	}
}

func TestMarkdownNoNotes(t *testing.T) {
	var buf bytes.Buffer
	if err := render.Markdown(&buf, nil); err != nil {
		t.Fatalf("rendering: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output, got %q", buf.String())
	}
}
