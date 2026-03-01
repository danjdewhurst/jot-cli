package importer_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/danjdewhurst/jot-cli/internal/importer"
	"github.com/danjdewhurst/jot-cli/internal/model"
)

func TestParseReader_FrontmatterWithTitleTagsCreated(t *testing.T) {
	input := `---
title: My Note
tags:
  - project:work
  - priority:high
created: 2024-01-15T10:30:00Z
---
Body content here.
`
	got, err := importer.ParseReader(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseReader: %v", err)
	}

	if got.Title != "My Note" {
		t.Errorf("title = %q, want %q", got.Title, "My Note")
	}
	if got.Body != "Body content here.\n" {
		t.Errorf("body = %q, want %q", got.Body, "Body content here.\n")
	}

	wantTags := []model.Tag{
		{Key: "project", Value: "work"},
		{Key: "priority", Value: "high"},
	}
	if len(got.Tags) != len(wantTags) {
		t.Fatalf("got %d tags, want %d", len(got.Tags), len(wantTags))
	}
	for i, tag := range got.Tags {
		if tag != wantTags[i] {
			t.Errorf("tag[%d] = %v, want %v", i, tag, wantTags[i])
		}
	}

	if got.CreatedAt == nil {
		t.Fatal("expected CreatedAt to be set")
	}
	wantTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	if !got.CreatedAt.Equal(wantTime) {
		t.Errorf("created_at = %v, want %v", got.CreatedAt, wantTime)
	}
}

func TestParseReader_FrontmatterDateOnly(t *testing.T) {
	input := `---
title: Date Only
created: 2024-06-01
---
Body.
`
	got, err := importer.ParseReader(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseReader: %v", err)
	}

	if got.CreatedAt == nil {
		t.Fatal("expected CreatedAt to be set")
	}
	wantTime := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	if !got.CreatedAt.Equal(wantTime) {
		t.Errorf("created_at = %v, want %v", got.CreatedAt, wantTime)
	}
}

func TestParseReader_NoFrontmatter_TitleFromHeading(t *testing.T) {
	input := `# My Note

Body content here.
`
	got, err := importer.ParseReader(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseReader: %v", err)
	}

	if got.Title != "My Note" {
		t.Errorf("title = %q, want %q", got.Title, "My Note")
	}
	if got.Body != "Body content here.\n" {
		t.Errorf("body = %q, want %q", got.Body, "Body content here.\n")
	}
	if len(got.Tags) != 0 {
		t.Errorf("got %d tags, want 0", len(got.Tags))
	}
	if got.CreatedAt != nil {
		t.Errorf("expected nil CreatedAt, got %v", got.CreatedAt)
	}
}

func TestParseReader_EmptyBody(t *testing.T) {
	input := `---
title: Empty
---
`
	got, err := importer.ParseReader(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseReader: %v", err)
	}

	if got.Title != "Empty" {
		t.Errorf("title = %q, want %q", got.Title, "Empty")
	}
	if got.Body != "" {
		t.Errorf("body = %q, want empty", got.Body)
	}
}

func TestParseReader_NoTitleNoHeading(t *testing.T) {
	input := `Just some text without a heading.
`
	got, err := importer.ParseReader(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseReader: %v", err)
	}

	if got.Title != "" {
		t.Errorf("title = %q, want empty", got.Title)
	}
	if got.Body != "Just some text without a heading.\n" {
		t.Errorf("body = %q, want %q", got.Body, "Just some text without a heading.\n")
	}
}

func TestParseReader_MalformedFrontmatter(t *testing.T) {
	input := `---
title: [invalid yaml
---
Body after bad frontmatter.
`
	got, err := importer.ParseReader(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseReader: %v (expected graceful fallback)", err)
	}

	if got.Body == "" {
		t.Error("expected non-empty body on malformed frontmatter")
	}
}

func TestParseReader_FrontmatterWithInvalidTag(t *testing.T) {
	input := `---
title: Partial Tags
tags:
  - project:work
  - invalidtag
---
Body.
`
	got, err := importer.ParseReader(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseReader: %v", err)
	}

	if len(got.Tags) != 1 {
		t.Errorf("got %d tags, want 1 (invalid tag should be skipped)", len(got.Tags))
	}
	if len(got.Tags) > 0 && got.Tags[0].Key != "project" {
		t.Errorf("tag key = %q, want %q", got.Tags[0].Key, "project")
	}
}

func TestParseFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/test.md"

	content := `---
title: File Test
---
Content from file.
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	got, err := importer.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	if got.Title != "File Test" {
		t.Errorf("title = %q, want %q", got.Title, "File Test")
	}
	if got.Body != "Content from file.\n" {
		t.Errorf("body = %q, want %q", got.Body, "Content from file.\n")
	}
}
