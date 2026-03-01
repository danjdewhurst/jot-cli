package render_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/danjdewhurst/jot-cli/internal/render"
	"github.com/danjdewhurst/jot-cli/internal/store"
)

func TestStatsTable_Empty(t *testing.T) {
	var buf bytes.Buffer
	render.StatsTable(&buf, store.NoteStats{})
	out := buf.String()

	if !strings.Contains(out, "0") {
		t.Errorf("expected zeroes for empty stats, got %q", out)
	}
}

func TestStatsTable_WithData(t *testing.T) {
	stats := store.NoteStats{
		TotalNotes:    142,
		ArchivedNotes: 3,
		PinnedNotes:   7,
		UniqueTags:    58,
		TopTags: []store.TagCount{
			{Key: "git_repo", Value: "jot-cli", Count: 34},
			{Key: "folder", Value: "work", Count: 21},
		},
		WeeklyCount:  12,
		MonthlyCount: 31,
		OldestDate:   time.Date(2024, 11, 3, 0, 0, 0, 0, time.UTC),
		NewestDate:   time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
	}

	var buf bytes.Buffer
	render.StatsTable(&buf, stats)
	out := buf.String()

	checks := []string{
		"142",
		"3 archived",
		"7",
		"58",
		"git_repo:jot-cli (34)",
		"folder:work (21)",
		"12",
		"31",
		"2024-11-03",
	}
	for _, check := range checks {
		if !strings.Contains(out, check) {
			t.Errorf("missing %q in output:\n%s", check, out)
		}
	}
}
