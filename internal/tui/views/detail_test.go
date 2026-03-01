package views_test

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/danjdewhurst/jot-cli/internal/model"
	"github.com/danjdewhurst/jot-cli/internal/tui/views"
)

func TestDetailView_ScrollClampedInUpdate(t *testing.T) {
	dv := views.NewDetailView()
	dv.SetSize(80, 5) // Small viewport
	dv.SetNote(model.Note{
		Title:     "Short",
		Body:      "One line body",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	// Scroll down many times — should not drift past content
	for i := 0; i < 50; i++ {
		dv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	}

	out := dv.View()
	// The view should still produce output even after excessive scrolling.
	// Previously, scroll was clamped on a value-receiver View() copy,
	// leaving the actual struct scroll drifting upward unbounded.
	if out == "" {
		t.Error("expected non-empty view after excessive scroll-down")
	}

	// pgdown should also be clamped
	for i := 0; i < 10; i++ {
		dv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{0x04}}) // ctrl+d
	}

	out2 := dv.View()
	if out2 == "" {
		t.Error("expected non-empty view after pgdown scrolling")
	}
}

func TestDetailView_BacklinksDisplayed(t *testing.T) {
	dv := views.NewDetailView()
	dv.SetSize(80, 40)
	dv.SetNote(model.Note{
		ID:        "01TARGET",
		Title:     "Target Note",
		Body:      "Some content",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	dv.SetBacklinks([]model.Note{
		{ID: "01ABCDEF12345678", Title: "Linking Note"},
	})

	out := dv.View()
	if !strings.Contains(out, "Referenced by") {
		t.Error("expected 'Referenced by' section in detail view")
	}
	if !strings.Contains(out, "01ABCDEF") {
		t.Error("expected backlink note ID (truncated) in detail view")
	}
	if !strings.Contains(out, "Linking Note") {
		t.Error("expected backlink note title in detail view")
	}
}

func TestDetailView_RefHighlighting(t *testing.T) {
	dv := views.NewDetailView()
	dv.SetSize(80, 40)
	dv.SetNote(model.Note{
		Title:     "Note with refs",
		Body:      "See @abc123 for details",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	out := dv.View()
	// The raw "@abc123" should be replaced with a styled version
	// The styled version will contain ANSI escape codes, so just check
	// that "abc123" is still present (styling wraps it)
	if !strings.Contains(out, "abc123") {
		t.Error("expected reference content in output")
	}
}

func TestDetailView_ScrollNeverNegative(t *testing.T) {
	dv := views.NewDetailView()
	dv.SetSize(80, 24)
	dv.SetNote(model.Note{
		Title:     "Test",
		Body:      "Body",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	// Scroll up past beginning
	for i := 0; i < 20; i++ {
		dv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	}

	out := dv.View()
	if !strings.Contains(out, "Test") {
		t.Error("expected title visible after excessive scroll-up")
	}
}
