package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/danjdewhurst/jot-cli/internal/model"
	"github.com/spf13/cobra"
)

func newLogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "log",
		RunE: runLog,
	}
	cmd.Flags().StringSlice("tag", nil, "Filter by tag (key:value)")
	cmd.Flags().Bool("folder", false, "Filter by current folder")
	cmd.Flags().Bool("repo", false, "Filter by current git repo")
	cmd.Flags().Bool("branch", false, "Filter by current git branch")
	cmd.Flags().Bool("archived", false, "Include archived notes")
	cmd.Flags().Int("limit", 0, "Maximum number of notes (default: 20)")
	cmd.Flags().String("since", "", "Show notes created after this date")
	cmd.Flags().String("until", "", "Show notes created before this date")
	cmd.Flags().Bool("reverse", false, "Show oldest notes first")
	cmd.Flags().Bool("today", false, "Show only today's notes")
	return cmd
}

func TestLogCmd_Default(t *testing.T) {
	s := openTestStore(t, testDBPath(t))
	setDB(t, s)

	for i := range 25 {
		title := "Note " + string(rune('A'+i))
		if _, err := s.CreateNote(title, "body", nil); err != nil {
			t.Fatalf("creating note: %v", err)
		}
	}

	var buf bytes.Buffer
	cmd := newLogCmd()
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("log: %v", err)
	}

	out := buf.String()
	// Default limit is 20, so we should see 20 note IDs (8-char short IDs)
	// Count the number of lines that contain a timestamp pattern
	lines := strings.Split(out, "\n")
	noteLines := 0
	for _, line := range lines {
		if strings.Contains(line, "2026-") {
			noteLines++
		}
	}
	if noteLines != 20 {
		t.Errorf("got %d note lines, want 20 (default limit)", noteLines)
	}
}

func TestLogCmd_Limit(t *testing.T) {
	s := openTestStore(t, testDBPath(t))
	setDB(t, s)

	for i := range 10 {
		if _, err := s.CreateNote("Note "+string(rune('A'+i)), "body", nil); err != nil {
			t.Fatalf("creating note: %v", err)
		}
	}

	var buf bytes.Buffer
	cmd := newLogCmd()
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--limit", "5"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("log: %v", err)
	}

	out := buf.String()
	lines := strings.Split(out, "\n")
	noteLines := 0
	for _, line := range lines {
		if strings.Contains(line, "2026-") {
			noteLines++
		}
	}
	if noteLines != 5 {
		t.Errorf("got %d note lines, want 5", noteLines)
	}
}

func TestLogCmd_Today(t *testing.T) {
	s := openTestStore(t, testDBPath(t))
	setDB(t, s)

	// All notes created now are "today"
	if _, err := s.CreateNote("Today's note", "body", nil); err != nil {
		t.Fatalf("creating note: %v", err)
	}

	var buf bytes.Buffer
	cmd := newLogCmd()
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--today"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("log: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Today's note") {
		t.Errorf("expected today's note in output, got %q", out)
	}
}

func TestLogCmd_Reverse(t *testing.T) {
	s := openTestStore(t, testDBPath(t))
	setDB(t, s)

	first, _ := s.CreateNote("First", "body", nil)
	time.Sleep(1100 * time.Millisecond)
	_, _ = s.CreateNote("Second", "body", nil)

	var buf bytes.Buffer
	cmd := newLogCmd()
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--reverse"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("log: %v", err)
	}

	out := buf.String()
	firstID := first.ID[:8]
	// First should appear before Second in --reverse (oldest first)
	idx := strings.Index(out, firstID)
	if idx < 0 {
		t.Fatalf("first note ID %q not found in output", firstID)
	}
	// The first ID should appear near the start
	if idx > 50 {
		t.Errorf("expected oldest note first with --reverse, first ID at position %d", idx)
	}
}

func TestLogCmd_TagFilter(t *testing.T) {
	s := openTestStore(t, testDBPath(t))
	setDB(t, s)

	if _, err := s.CreateNote("Tagged", "body", []model.Tag{{Key: "project", Value: "alpha"}}); err != nil {
		t.Fatalf("creating note: %v", err)
	}
	if _, err := s.CreateNote("Untagged", "body", nil); err != nil {
		t.Fatalf("creating note: %v", err)
	}

	var buf bytes.Buffer
	cmd := newLogCmd()
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--tag", "project:alpha"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("log: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Tagged") {
		t.Error("expected tagged note in output")
	}
	if strings.Contains(out, "Untagged") {
		t.Error("untagged note should not appear")
	}
}

func TestLogCmd_JSON(t *testing.T) {
	s := openTestStore(t, testDBPath(t))
	setDB(t, s)

	if _, err := s.CreateNote("JSON Note", "body", nil); err != nil {
		t.Fatalf("creating note: %v", err)
	}

	oldJSON := flagJSON
	flagJSON = true
	t.Cleanup(func() { flagJSON = oldJSON })

	var buf bytes.Buffer
	cmd := newLogCmd()
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("log: %v", err)
	}

	var notes []model.Note
	if err := json.Unmarshal(buf.Bytes(), &notes); err != nil {
		t.Fatalf("decoding JSON: %v (output: %q)", err, buf.String())
	}
	if len(notes) != 1 {
		t.Errorf("got %d notes, want 1", len(notes))
	}
}

func TestLogCmd_NoNotes(t *testing.T) {
	s := openTestStore(t, testDBPath(t))
	setDB(t, s)

	var buf bytes.Buffer
	cmd := newLogCmd()
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("log: %v", err)
	}

	if !strings.Contains(buf.String(), "No notes found.") {
		t.Errorf("expected 'No notes found.', got %q", buf.String())
	}
}
