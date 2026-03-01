package diff_test

import (
	"testing"

	"github.com/danjdewhurst/jot-cli/internal/diff"
)

func TestSummary_NoChanges(t *testing.T) {
	s := diff.Summary("hello\nworld", "hello\nworld")
	if s != "no changes" {
		t.Errorf("got %q, want %q", s, "no changes")
	}
}

func TestSummary_AddedLines(t *testing.T) {
	s := diff.Summary("hello", "hello\nworld")
	if s != "+1 / -0" {
		t.Errorf("got %q, want %q", s, "+1 / -0")
	}
}

func TestSummary_RemovedLines(t *testing.T) {
	s := diff.Summary("hello\nworld", "hello")
	if s != "+0 / -1" {
		t.Errorf("got %q, want %q", s, "+0 / -1")
	}
}

func TestSummary_MixedChanges(t *testing.T) {
	s := diff.Summary("line1\nline2\nline3", "line1\nchanged\nline3\nline4")
	// line2 -> changed = 1 add + 1 remove, line4 = 1 add
	if s != "+2 / -1" {
		t.Errorf("got %q, want %q", s, "+2 / -1")
	}
}

func TestSummary_EmptyOld(t *testing.T) {
	s := diff.Summary("", "hello\nworld")
	if s != "+2 / -0" {
		t.Errorf("got %q, want %q", s, "+2 / -0")
	}
}

func TestSummary_EmptyNew(t *testing.T) {
	s := diff.Summary("hello\nworld", "")
	if s != "+0 / -2" {
		t.Errorf("got %q, want %q", s, "+0 / -2")
	}
}

func TestLines_NoChanges(t *testing.T) {
	result := diff.Lines("hello", "hello")
	if len(result) != 1 {
		t.Fatalf("got %d lines, want 1", len(result))
	}
	if result[0].Op != diff.OpEqual {
		t.Errorf("op = %v, want Equal", result[0].Op)
	}
}

func TestLines_Added(t *testing.T) {
	result := diff.Lines("hello", "hello\nworld")
	var added int
	for _, l := range result {
		if l.Op == diff.OpAdd {
			added++
		}
	}
	if added != 1 {
		t.Errorf("got %d added lines, want 1", added)
	}
}

func TestLines_Removed(t *testing.T) {
	result := diff.Lines("hello\nworld", "hello")
	var removed int
	for _, l := range result {
		if l.Op == diff.OpRemove {
			removed++
		}
	}
	if removed != 1 {
		t.Errorf("got %d removed lines, want 1", removed)
	}
}

func TestFormat(t *testing.T) {
	output := diff.Format("hello\nworld", "hello\nchanged")
	if output == "" {
		t.Error("expected non-empty formatted diff")
	}
}
