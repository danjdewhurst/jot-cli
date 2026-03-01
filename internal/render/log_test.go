package render_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/danjdewhurst/jot-cli/internal/model"
	"github.com/danjdewhurst/jot-cli/internal/render"
)

func TestNoteLog_Empty(t *testing.T) {
	var buf bytes.Buffer
	render.NoteLog(&buf, nil)

	if !strings.Contains(buf.String(), "No notes found.") {
		t.Errorf("expected 'No notes found.', got %q", buf.String())
	}
}

func TestNoteLog_SingleNote(t *testing.T) {
	ts := time.Date(2026, 3, 1, 14, 30, 0, 0, time.UTC)
	notes := []model.Note{{
		ID:        "01HXK3M2ABCDEF0123456789",
		Title:     "Fix deployment script for staging",
		CreatedAt: ts,
	}}

	var buf bytes.Buffer
	render.NoteLog(&buf, notes)
	out := buf.String()

	if !strings.Contains(out, "01HXK3M2") {
		t.Error("missing short ID")
	}
	if !strings.Contains(out, "2026-03-01 14:30") {
		t.Error("missing timestamp")
	}
	if !strings.Contains(out, "Fix deployment script for staging") {
		t.Error("missing title")
	}
}

func TestNoteLog_NoTitle(t *testing.T) {
	ts := time.Date(2026, 3, 1, 14, 30, 0, 0, time.UTC)
	notes := []model.Note{{
		ID:        "01HXK3M2ABCDEF0123456789",
		Body:      "Quick thought about caching\nand some more detail",
		CreatedAt: ts,
	}}

	var buf bytes.Buffer
	render.NoteLog(&buf, notes)
	out := buf.String()

	// Body fallback should replace newlines with spaces
	if !strings.Contains(out, "Quick thought about caching and some more detail") {
		t.Errorf("expected body fallback, got %q", out)
	}
}

func TestNoteLog_NoTitleNoBody(t *testing.T) {
	ts := time.Date(2026, 3, 1, 14, 30, 0, 0, time.UTC)
	notes := []model.Note{{
		ID:        "01HXK3M2ABCDEF0123456789",
		CreatedAt: ts,
	}}

	var buf bytes.Buffer
	render.NoteLog(&buf, notes)
	out := buf.String()

	if !strings.Contains(out, "(empty)") {
		t.Errorf("expected '(empty)', got %q", out)
	}
}

func TestNoteLog_TagFormatting(t *testing.T) {
	ts := time.Date(2026, 3, 1, 14, 30, 0, 0, time.UTC)
	notes := []model.Note{{
		ID:        "01HXK3M2ABCDEF0123456789",
		Title:     "Tagged note",
		CreatedAt: ts,
		Tags: []model.Tag{
			{Key: "folder", Value: "/home/dan/projects"},
			{Key: "git_branch", Value: "main"},
		},
	}}

	var buf bytes.Buffer
	render.NoteLog(&buf, notes)
	out := buf.String()

	if !strings.Contains(out, "folder:") {
		t.Error("missing tag key 'folder:'")
	}
	if !strings.Contains(out, "/home/dan/projects") {
		t.Error("missing tag value")
	}
	if !strings.Contains(out, "git_branch:") {
		t.Error("missing tag key 'git_branch:'")
	}
}

func TestNoteLog_LongBody(t *testing.T) {
	ts := time.Date(2026, 3, 1, 14, 30, 0, 0, time.UTC)
	longBody := strings.Repeat("a", 100)
	notes := []model.Note{{
		ID:        "01HXK3M2ABCDEF0123456789",
		Body:      longBody,
		CreatedAt: ts,
	}}

	var buf bytes.Buffer
	render.NoteLog(&buf, notes)
	out := buf.String()

	if !strings.Contains(out, "...") {
		t.Error("expected truncation ellipsis for long body")
	}
}
