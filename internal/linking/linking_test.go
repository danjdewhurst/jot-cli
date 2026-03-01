package linking

import (
	"testing"
)

func TestExtractRefs_NoRefs(t *testing.T) {
	refs := ExtractRefs("Just a plain note body")
	if len(refs) != 0 {
		t.Errorf("expected no refs, got %v", refs)
	}
}

func TestExtractRefs_SingleRef(t *testing.T) {
	refs := ExtractRefs("Related to @abc123")
	if len(refs) != 1 || refs[0] != "abc123" {
		t.Errorf("expected [abc123], got %v", refs)
	}
}

func TestExtractRefs_MultipleRefs(t *testing.T) {
	refs := ExtractRefs("See @abc123 and @def456")
	if len(refs) != 2 {
		t.Fatalf("expected 2 refs, got %d", len(refs))
	}
	if refs[0] != "abc123" || refs[1] != "def456" {
		t.Errorf("expected [abc123 def456], got %v", refs)
	}
}

func TestExtractRefs_ShortRefIgnored(t *testing.T) {
	refs := ExtractRefs("Too short @abc")
	if len(refs) != 0 {
		t.Errorf("expected no refs for 3-char prefix, got %v", refs)
	}
}

func TestExtractRefs_MinLength(t *testing.T) {
	refs := ExtractRefs("Minimum @abcd")
	if len(refs) != 1 || refs[0] != "abcd" {
		t.Errorf("expected [abcd], got %v", refs)
	}
}

func TestExtractRefs_AtStartOfLine(t *testing.T) {
	refs := ExtractRefs("@abc123 is at the start")
	if len(refs) != 1 || refs[0] != "abc123" {
		t.Errorf("expected [abc123], got %v", refs)
	}
}

func TestExtractRefs_AtEndOfLine(t *testing.T) {
	refs := ExtractRefs("Ends with @abc123")
	if len(refs) != 1 || refs[0] != "abc123" {
		t.Errorf("expected [abc123], got %v", refs)
	}
}

func TestExtractRefs_InMultiline(t *testing.T) {
	body := "First line\nSee @abc123\nAnother @def456 here"
	refs := ExtractRefs(body)
	if len(refs) != 2 {
		t.Fatalf("expected 2 refs, got %d", len(refs))
	}
	if refs[0] != "abc123" || refs[1] != "def456" {
		t.Errorf("expected [abc123 def456], got %v", refs)
	}
}

func TestExtractRefs_DuplicatesRemoved(t *testing.T) {
	refs := ExtractRefs("See @abc123 and again @abc123")
	if len(refs) != 1 {
		t.Errorf("expected 1 unique ref, got %v", refs)
	}
}

func TestExtractRefs_EmailNotMatched(t *testing.T) {
	refs := ExtractRefs("Email user@example.com should not match")
	if len(refs) != 0 {
		t.Errorf("expected no refs from email, got %v", refs)
	}
}

func TestExtractRefs_EmptyBody(t *testing.T) {
	refs := ExtractRefs("")
	if len(refs) != 0 {
		t.Errorf("expected no refs from empty body, got %v", refs)
	}
}

func TestExtractRefs_FullULID(t *testing.T) {
	refs := ExtractRefs("See @01JMXYZ1234567890ABCDEFGH")
	if len(refs) != 1 || refs[0] != "01JMXYZ1234567890ABCDEFGH" {
		t.Errorf("expected full ULID, got %v", refs)
	}
}

func TestExtractRefs_WithPunctuation(t *testing.T) {
	refs := ExtractRefs("Check @abc123, and @def456.")
	if len(refs) != 2 {
		t.Fatalf("expected 2 refs, got %d", len(refs))
	}
	if refs[0] != "abc123" || refs[1] != "def456" {
		t.Errorf("expected [abc123 def456], got %v", refs)
	}
}
