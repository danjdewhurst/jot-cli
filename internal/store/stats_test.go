package store_test

import (
	"testing"
	"time"

	"github.com/danjdewhurst/jot-cli/internal/model"
)

func TestStats_Empty(t *testing.T) {
	s := newTestStore(t)

	stats, err := s.Stats()
	if err != nil {
		t.Fatalf("stats: %v", err)
	}

	if stats.TotalNotes != 0 {
		t.Errorf("TotalNotes = %d, want 0", stats.TotalNotes)
	}
	if stats.ArchivedNotes != 0 {
		t.Errorf("ArchivedNotes = %d, want 0", stats.ArchivedNotes)
	}
	if stats.PinnedNotes != 0 {
		t.Errorf("PinnedNotes = %d, want 0", stats.PinnedNotes)
	}
	if stats.UniqueTags != 0 {
		t.Errorf("UniqueTags = %d, want 0", stats.UniqueTags)
	}
	if len(stats.TopTags) != 0 {
		t.Errorf("TopTags = %v, want empty", stats.TopTags)
	}
	if stats.WeeklyCount != 0 {
		t.Errorf("WeeklyCount = %d, want 0", stats.WeeklyCount)
	}
	if stats.MonthlyCount != 0 {
		t.Errorf("MonthlyCount = %d, want 0", stats.MonthlyCount)
	}
	if !stats.OldestDate.IsZero() {
		t.Errorf("OldestDate = %v, want zero", stats.OldestDate)
	}
	if !stats.NewestDate.IsZero() {
		t.Errorf("NewestDate = %v, want zero", stats.NewestDate)
	}
}

func TestStats_Counts(t *testing.T) {
	s := newTestStore(t)

	_, _ = s.CreateNote("Active 1", "body", nil)
	_, _ = s.CreateNote("Active 2", "body", nil)
	pinned, _ := s.CreateNote("Pinned", "body", nil)
	_ = s.PinNote(pinned.ID)
	archived, _ := s.CreateNote("Archived", "body", nil)
	_ = s.ArchiveNote(archived.ID)

	stats, err := s.Stats()
	if err != nil {
		t.Fatalf("stats: %v", err)
	}

	if stats.TotalNotes != 4 {
		t.Errorf("TotalNotes = %d, want 4", stats.TotalNotes)
	}
	if stats.ArchivedNotes != 1 {
		t.Errorf("ArchivedNotes = %d, want 1", stats.ArchivedNotes)
	}
	if stats.PinnedNotes != 1 {
		t.Errorf("PinnedNotes = %d, want 1", stats.PinnedNotes)
	}
}

func TestStats_Tags(t *testing.T) {
	s := newTestStore(t)

	_, _ = s.CreateNote("A", "", []model.Tag{
		{Key: "folder", Value: "work"},
		{Key: "git_repo", Value: "jot-cli"},
	})
	_, _ = s.CreateNote("B", "", []model.Tag{
		{Key: "folder", Value: "work"},
		{Key: "git_branch", Value: "main"},
	})
	_, _ = s.CreateNote("C", "", []model.Tag{
		{Key: "folder", Value: "home"},
	})

	stats, err := s.Stats()
	if err != nil {
		t.Fatalf("stats: %v", err)
	}

	if stats.UniqueTags != 4 {
		t.Errorf("UniqueTags = %d, want 4", stats.UniqueTags)
	}

	// Top tag should be folder:work (2 notes)
	if len(stats.TopTags) == 0 {
		t.Fatal("expected at least one top tag")
	}
	top := stats.TopTags[0]
	if top.Key != "folder" || top.Value != "work" || top.Count != 2 {
		t.Errorf("top tag = %s:%s (%d), want folder:work (2)", top.Key, top.Value, top.Count)
	}
}

func TestStats_WeeklyAndMonthly(t *testing.T) {
	s := newTestStore(t)

	// Notes created now count for both weekly and monthly
	_, _ = s.CreateNote("Recent", "body", nil)
	_, _ = s.CreateNote("Also Recent", "body", nil)

	stats, err := s.Stats()
	if err != nil {
		t.Fatalf("stats: %v", err)
	}

	if stats.WeeklyCount != 2 {
		t.Errorf("WeeklyCount = %d, want 2", stats.WeeklyCount)
	}
	if stats.MonthlyCount != 2 {
		t.Errorf("MonthlyCount = %d, want 2", stats.MonthlyCount)
	}
}

func TestStats_Dates(t *testing.T) {
	s := newTestStore(t)

	before := time.Now().UTC()
	_, _ = s.CreateNote("First", "body", nil)
	_, _ = s.CreateNote("Second", "body", nil)
	after := time.Now().UTC()

	stats, err := s.Stats()
	if err != nil {
		t.Fatalf("stats: %v", err)
	}

	if stats.OldestDate.Before(before.Add(-time.Second)) || stats.OldestDate.After(after.Add(time.Second)) {
		t.Errorf("OldestDate = %v, expected between %v and %v", stats.OldestDate, before, after)
	}
	if stats.NewestDate.Before(before.Add(-time.Second)) || stats.NewestDate.After(after.Add(time.Second)) {
		t.Errorf("NewestDate = %v, expected between %v and %v", stats.NewestDate, before, after)
	}
}
