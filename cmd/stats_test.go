package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/danjdewhurst/jot-cli/internal/model"
	"github.com/danjdewhurst/jot-cli/internal/store"
	"github.com/spf13/cobra"
)

func newStatsCmd() *cobra.Command {
	return &cobra.Command{
		Use:  "stats",
		RunE: runStats,
	}
}

func TestStatsCmd_Empty(t *testing.T) {
	s := openTestStore(t, testDBPath(t))
	setDB(t, s)

	var buf bytes.Buffer
	cmd := newStatsCmd()
	cmd.SetOut(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("stats: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Notes:") {
		t.Errorf("expected 'Notes:' in output, got %q", out)
	}
}

func TestStatsCmd_WithData(t *testing.T) {
	s := openTestStore(t, testDBPath(t))
	setDB(t, s)

	_, _ = s.CreateNote("A", "body", []model.Tag{{Key: "folder", Value: "work"}})
	_, _ = s.CreateNote("B", "body", nil)
	pinned, _ := s.CreateNote("C", "body", nil)
	_ = s.PinNote(pinned.ID)

	var buf bytes.Buffer
	cmd := newStatsCmd()
	cmd.SetOut(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("stats: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Pinned:") {
		t.Errorf("expected pinned info, got %q", out)
	}
	if !strings.Contains(out, "folder:work") {
		t.Errorf("expected top tag info, got %q", out)
	}
}

func TestStatsCmd_JSON(t *testing.T) {
	s := openTestStore(t, testDBPath(t))
	setDB(t, s)

	_, _ = s.CreateNote("Note", "body", nil)

	oldJSON := flagJSON
	flagJSON = true
	t.Cleanup(func() { flagJSON = oldJSON })

	var buf bytes.Buffer
	cmd := newStatsCmd()
	cmd.SetOut(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("stats: %v", err)
	}

	var stats store.NoteStats
	if err := json.Unmarshal(buf.Bytes(), &stats); err != nil {
		t.Fatalf("decoding JSON: %v (output: %q)", err, buf.String())
	}
	if stats.TotalNotes != 1 {
		t.Errorf("TotalNotes = %d, want 1", stats.TotalNotes)
	}
}
