package render_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/danjdewhurst/jot-cli/internal/render"
)

func TestTruncate_ShortString(t *testing.T) {
	got := render.Truncate("hello", 10)
	if got != "hello" {
		t.Errorf("expected %q, got %q", "hello", got)
	}
}

func TestTruncate_ExactLength(t *testing.T) {
	got := render.Truncate("hello", 5)
	if got != "hello" {
		t.Errorf("expected %q, got %q", "hello", got)
	}
}

func TestTruncate_LongString(t *testing.T) {
	got := render.Truncate("hello world", 8)
	// Should be 7 chars + ellipsis
	if got != "hello w…" {
		t.Errorf("expected %q, got %q", "hello w…", got)
	}
}

func TestTruncate_NewlinesReplacedWithSpaces(t *testing.T) {
	got := render.Truncate("line1\nline2", 20)
	if got != "line1 line2" {
		t.Errorf("expected %q, got %q", "line1 line2", got)
	}
}

func TestTruncate_MultibyteRunes(t *testing.T) {
	// 6 runes: each is multi-byte in UTF-8
	input := "日本語テスト"
	got := render.Truncate(input, 4)
	// Should be 3 runes + ellipsis = 4 runes total
	expected := "日本語…"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestTruncate_EmptyString(t *testing.T) {
	got := render.Truncate("", 10)
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestTruncate_MaxOne(t *testing.T) {
	got := render.Truncate("hello", 1)
	if got != "…" {
		t.Errorf("expected %q, got %q", "…", got)
	}
}

func TestRelativeTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		t        time.Time
		expected string
	}{
		{"just now", now.Add(-30 * time.Second), "now"},
		{"minutes", now.Add(-5 * time.Minute), "5m ago"},
		{"hours", now.Add(-3 * time.Hour), "3h ago"},
		{"days", now.Add(-7 * 24 * time.Hour), "7d ago"},
		{"months", now.Add(-90 * 24 * time.Hour), "3mo ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := render.RelativeTime(tt.t)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestRelativeTimeShort(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		t        time.Time
		expected string
	}{
		{"just now", now.Add(-30 * time.Second), "now"},
		{"minutes", now.Add(-5 * time.Minute), "5m"},
		{"hours", now.Add(-3 * time.Hour), "3h"},
		{"days", now.Add(-7 * 24 * time.Hour), "7d"},
		{"months", now.Add(-90 * 24 * time.Hour), "3mo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := render.RelativeTimeShort(tt.t)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestTruncateLog_TrimsAndUsesEllipsis(t *testing.T) {
	input := "  hello world  "
	got := render.TruncateLog(input, 8)
	// After trim: "hello world" (11 chars > 8), truncate to 7 + "…"
	expected := "hello w…"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestTruncateLog_ShortString(t *testing.T) {
	got := render.TruncateLog("hi", 10)
	if got != "hi" {
		t.Errorf("expected %q, got %q", "hi", got)
	}
}

func TestTruncateLog_MultibyteRunes(t *testing.T) {
	input := "日本語テスト"
	got := render.TruncateLog(input, 4)
	expected := "日本語…"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestTruncateLog_WithNewlines(t *testing.T) {
	input := "line1\nline2\nline3"
	got := render.TruncateLog(input, 50)
	expected := "line1 line2 line3"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestFormatTime(t *testing.T) {
	ts := time.Date(2025, 6, 15, 14, 30, 0, 0, time.UTC)

	t.Run("relative (default)", func(t *testing.T) {
		render.DateFormat = ""
		got := render.FormatTime(ts)
		// Should contain "ago" or "now" (relative format)
		if got == "" {
			t.Error("FormatTime returned empty string")
		}
	})

	t.Run("relative explicit", func(t *testing.T) {
		render.DateFormat = "relative"
		got := render.FormatTime(ts)
		if got == "" {
			t.Error("FormatTime returned empty string")
		}
	})

	t.Run("absolute", func(t *testing.T) {
		render.DateFormat = "absolute"
		got := render.FormatTime(ts)
		want := "2025-06-15 14:30"
		if got != want {
			t.Errorf("FormatTime = %q, want %q", got, want)
		}
	})

	t.Run("iso", func(t *testing.T) {
		render.DateFormat = "iso"
		got := render.FormatTime(ts)
		want := ts.Format(time.RFC3339)
		if got != want {
			t.Errorf("FormatTime = %q, want %q", got, want)
		}
	})

	// Reset
	render.DateFormat = ""
}

func ExampleTruncate() {
	fmt.Println(render.Truncate("hello world", 8))
	// Output: hello w…
}
