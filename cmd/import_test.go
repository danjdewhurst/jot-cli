package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/danjdewhurst/jot-cli/internal/model"
	"github.com/spf13/cobra"
)

func writeEnvelope(t *testing.T, env model.ExportEnvelope) string {
	t.Helper()
	data, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		t.Fatalf("marshalling envelope: %v", err)
	}
	path := filepath.Join(t.TempDir(), "import.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("writing file: %v", err)
	}
	return path
}

// newImportCmd creates a fresh import cobra command for testing.
func newImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "import <file|directory>",
		Args: cobra.ExactArgs(1),
		RunE: runImport,
	}
	cmd.Flags().Bool("dry-run", false, "Preview import without writing")
	cmd.Flags().Bool("new-ids", false, "Generate new IDs instead of preserving originals")
	cmd.Flags().Bool("no-context", false, "Skip auto-context tags")
	cmd.Flags().Bool("force", false, "Skip deduplication check")
	cmd.Flags().StringSlice("tag", nil, "Additional tags for all imported notes (key:value)")
	return cmd
}

func TestImportJSON_NewNotes(t *testing.T) {
	s := openTestStore(t, testDBPath(t))
	setDB(t, s)

	now := time.Now().UTC().Truncate(time.Second)
	envelope := model.ExportEnvelope{
		Version:    1,
		ExportedAt: now,
		Count:      2,
		Notes: []model.Note{
			{
				ID:        "01JTESTIMPORT000000000001",
				Title:     "Import A",
				Body:      "Body A",
				CreatedAt: now,
				UpdatedAt: now,
				Tags:      []model.Tag{{Key: "source", Value: "test"}},
			},
			{
				ID:        "01JTESTIMPORT000000000002",
				Title:     "Import B",
				Body:      "Body B",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	importFile := writeEnvelope(t, envelope)

	cmd := newImportCmd()
	cmd.SetArgs([]string{"--no-context", importFile})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("import: %v", err)
	}

	n, err := db.GetNote("01JTESTIMPORT000000000001")
	if err != nil {
		t.Fatalf("getting note: %v", err)
	}
	if n.Title != "Import A" {
		t.Errorf("title = %q, want %q", n.Title, "Import A")
	}
	if len(n.Tags) < 1 || n.Tags[0].Key != "source" {
		t.Errorf("tags = %v, want at least source:test", n.Tags)
	}

	n2, err := db.GetNote("01JTESTIMPORT000000000002")
	if err != nil {
		t.Fatalf("getting note 2: %v", err)
	}
	if n2.Title != "Import B" {
		t.Errorf("title = %q, want %q", n2.Title, "Import B")
	}
}

func TestImportJSON_DuplicateSkip(t *testing.T) {
	s := openTestStore(t, testDBPath(t))
	setDB(t, s)

	now := time.Now().UTC().Truncate(time.Second)
	envelope := model.ExportEnvelope{
		Version:    1,
		ExportedAt: now,
		Count:      1,
		Notes: []model.Note{{
			ID:        "01JTESTIMPORTDUP00000001",
			Title:     "Dup Note",
			Body:      "body",
			CreatedAt: now,
			UpdatedAt: now,
		}},
	}

	importFile := writeEnvelope(t, envelope)

	// First import
	cmd1 := newImportCmd()
	cmd1.SetArgs([]string{"--no-context", importFile})
	if err := cmd1.Execute(); err != nil {
		t.Fatalf("first import: %v", err)
	}

	// Second import — should skip
	cmd2 := newImportCmd()
	cmd2.SetArgs([]string{"--no-context", importFile})
	if err := cmd2.Execute(); err != nil {
		t.Fatalf("second import: %v", err)
	}

	// Verify only one note exists
	notes, err := db.ListNotes(model.NoteFilter{})
	if err != nil {
		t.Fatalf("listing notes: %v", err)
	}
	if len(notes) != 1 {
		t.Errorf("got %d notes, want 1", len(notes))
	}
}

func TestImportJSON_NewIDs(t *testing.T) {
	s := openTestStore(t, testDBPath(t))
	setDB(t, s)

	now := time.Now().UTC().Truncate(time.Second)
	originalID := "01JTESTIMPORTNEWID000001"
	envelope := model.ExportEnvelope{
		Version:    1,
		ExportedAt: now,
		Count:      1,
		Notes: []model.Note{{
			ID:        originalID,
			Title:     "New ID Note",
			Body:      "body",
			CreatedAt: now,
			UpdatedAt: now,
		}},
	}

	importFile := writeEnvelope(t, envelope)

	cmd := newImportCmd()
	cmd.SetArgs([]string{"--new-ids", "--no-context", importFile})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("import: %v", err)
	}

	// Original ID should not exist
	_, err := db.GetNote(originalID)
	if err == nil {
		t.Error("expected original ID to not exist")
	}

	// Should have one note with a different ID
	notes, err := db.ListNotes(model.NoteFilter{})
	if err != nil {
		t.Fatalf("listing notes: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("got %d notes, want 1", len(notes))
	}
	if notes[0].ID == originalID {
		t.Error("imported note should have a new ID")
	}
	if notes[0].Title != "New ID Note" {
		t.Errorf("title = %q, want %q", notes[0].Title, "New ID Note")
	}
}

func TestImportJSON_DryRun(t *testing.T) {
	s := openTestStore(t, testDBPath(t))
	setDB(t, s)

	now := time.Now().UTC().Truncate(time.Second)
	envelope := model.ExportEnvelope{
		Version:    1,
		ExportedAt: now,
		Count:      1,
		Notes: []model.Note{{
			ID:        "01JTESTIMPORTDRY0000001",
			Title:     "Dry Run Note",
			Body:      "body",
			CreatedAt: now,
			UpdatedAt: now,
		}},
	}

	importFile := writeEnvelope(t, envelope)

	cmd := newImportCmd()
	cmd.SetArgs([]string{"--dry-run", "--no-context", importFile})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("import: %v", err)
	}

	// Note should NOT exist
	notes, err := db.ListNotes(model.NoteFilter{Archived: true})
	if err != nil {
		t.Fatalf("listing notes: %v", err)
	}
	if len(notes) != 0 {
		t.Errorf("got %d notes, want 0 after dry run", len(notes))
	}
}

func TestImportJSON_BadVersion(t *testing.T) {
	s := openTestStore(t, testDBPath(t))
	setDB(t, s)

	envelope := model.ExportEnvelope{
		Version:    99,
		ExportedAt: time.Now().UTC(),
		Count:      0,
		Notes:      nil,
	}

	importFile := writeEnvelope(t, envelope)

	cmd := newImportCmd()
	cmd.SetArgs([]string{"--no-context", importFile})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for bad version")
	}
	if !strings.Contains(err.Error(), "unsupported export version") {
		t.Errorf("error = %q, want to contain 'unsupported export version'", err.Error())
	}
}

func TestImportJSON_MalformedJSON(t *testing.T) {
	s := openTestStore(t, testDBPath(t))
	setDB(t, s)

	badFile := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(badFile, []byte("{not valid json"), 0644); err != nil {
		t.Fatalf("writing file: %v", err)
	}

	cmd := newImportCmd()
	cmd.SetArgs([]string{"--no-context", badFile})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func writeMDFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing file: %v", err)
	}
	return path
}

func TestImportMarkdown_SingleFile(t *testing.T) {
	s := openTestStore(t, testDBPath(t))
	setDB(t, s)

	dir := t.TempDir()
	mdFile := writeMDFile(t, dir, "note.md", `---
title: Markdown Note
tags:
  - project:work
created: 2024-03-15T12:00:00Z
---
This is the body.
`)

	cmd := newImportCmd()
	cmd.SetArgs([]string{"--no-context", mdFile})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("import: %v", err)
	}

	notes, err := db.ListNotes(model.NoteFilter{})
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("got %d notes, want 1", len(notes))
	}
	if notes[0].Title != "Markdown Note" {
		t.Errorf("title = %q, want %q", notes[0].Title, "Markdown Note")
	}
	if notes[0].Body != "This is the body.\n" {
		t.Errorf("body = %q, want %q", notes[0].Body, "This is the body.\n")
	}
	wantTime := time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC)
	if !notes[0].CreatedAt.Equal(wantTime) {
		t.Errorf("created_at = %v, want %v", notes[0].CreatedAt, wantTime)
	}
	// Should have the project:work tag
	found := false
	for _, tag := range notes[0].Tags {
		if tag.Key == "project" && tag.Value == "work" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected project:work tag, got %v", notes[0].Tags)
	}
}

func TestImportMarkdown_Directory(t *testing.T) {
	s := openTestStore(t, testDBPath(t))
	setDB(t, s)

	dir := t.TempDir()
	writeMDFile(t, dir, "one.md", "# First\n\nBody one.\n")
	writeMDFile(t, dir, "two.md", "# Second\n\nBody two.\n")
	// Non-md file should be ignored
	writeMDFile(t, dir, "readme.txt", "not a note")

	cmd := newImportCmd()
	cmd.SetArgs([]string{"--no-context", dir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("import: %v", err)
	}

	notes, err := db.ListNotes(model.NoteFilter{})
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if len(notes) != 2 {
		t.Errorf("got %d notes, want 2", len(notes))
	}
}

func TestImportMarkdown_DryRun(t *testing.T) {
	s := openTestStore(t, testDBPath(t))
	setDB(t, s)

	dir := t.TempDir()
	mdFile := writeMDFile(t, dir, "dry.md", "# Dry Run\n\nBody.\n")

	cmd := newImportCmd()
	cmd.SetArgs([]string{"--dry-run", "--no-context", mdFile})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("import: %v", err)
	}

	notes, err := db.ListNotes(model.NoteFilter{Archived: true})
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if len(notes) != 0 {
		t.Errorf("got %d notes, want 0 after dry run", len(notes))
	}
}

func TestImportMarkdown_DedupSkips(t *testing.T) {
	s := openTestStore(t, testDBPath(t))
	setDB(t, s)

	// Pre-create a note with the same title and body
	_, err := s.CreateNote("Existing Note", "Same body.\n", nil)
	if err != nil {
		t.Fatalf("creating note: %v", err)
	}

	dir := t.TempDir()
	mdFile := writeMDFile(t, dir, "dup.md", "---\ntitle: Existing Note\n---\nSame body.\n")

	cmd := newImportCmd()
	cmd.SetArgs([]string{"--no-context", mdFile})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("import: %v", err)
	}

	notes, err := db.ListNotes(model.NoteFilter{})
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if len(notes) != 1 {
		t.Errorf("got %d notes, want 1 (duplicate should be skipped)", len(notes))
	}
}

func TestImportMarkdown_ForceSkipsDedup(t *testing.T) {
	s := openTestStore(t, testDBPath(t))
	setDB(t, s)

	// Pre-create a note with the same title and body
	_, err := s.CreateNote("Force Note", "Same body.\n", nil)
	if err != nil {
		t.Fatalf("creating note: %v", err)
	}

	dir := t.TempDir()
	mdFile := writeMDFile(t, dir, "force.md", "---\ntitle: Force Note\n---\nSame body.\n")

	cmd := newImportCmd()
	cmd.SetArgs([]string{"--force", "--no-context", mdFile})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("import: %v", err)
	}

	notes, err := db.ListNotes(model.NoteFilter{})
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if len(notes) != 2 {
		t.Errorf("got %d notes, want 2 (force should skip dedup)", len(notes))
	}
}

func TestImportMarkdown_ExtraTags(t *testing.T) {
	s := openTestStore(t, testDBPath(t))
	setDB(t, s)

	dir := t.TempDir()
	mdFile := writeMDFile(t, dir, "tagged.md", "# Tagged\n\nBody.\n")

	cmd := newImportCmd()
	cmd.SetArgs([]string{"--no-context", "--tag", "source:import", mdFile})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("import: %v", err)
	}

	notes, err := db.ListNotes(model.NoteFilter{})
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("got %d notes, want 1", len(notes))
	}
	found := false
	for _, tag := range notes[0].Tags {
		if tag.Key == "source" && tag.Value == "import" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected source:import tag, got %v", notes[0].Tags)
	}
}

func TestImportMarkdown_SubdirectoryWalk(t *testing.T) {
	s := openTestStore(t, testDBPath(t))
	setDB(t, s)

	dir := t.TempDir()
	subDir := filepath.Join(dir, "subdir")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeMDFile(t, dir, "root.md", "# Root\n\nBody.\n")
	writeMDFile(t, subDir, "nested.md", "# Nested\n\nBody.\n")

	cmd := newImportCmd()
	cmd.SetArgs([]string{"--no-context", dir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("import: %v", err)
	}

	notes, err := db.ListNotes(model.NoteFilter{})
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if len(notes) != 2 {
		t.Errorf("got %d notes, want 2 (should walk subdirectories)", len(notes))
	}
}

func TestImportExportRoundTrip(t *testing.T) {
	s := openTestStore(t, testDBPath(t))
	setDB(t, s)

	tags := []model.Tag{{Key: "project", Value: "roundtrip"}}
	orig, err := s.CreateNote("Round Trip", "Testing full cycle", tags)
	if err != nil {
		t.Fatalf("creating note: %v", err)
	}

	// Export
	exportPath := filepath.Join(t.TempDir(), "roundtrip.json")
	ecmd := newExportCmd()
	ecmd.SetArgs([]string{"--format", "json", "--output", exportPath})
	if err := ecmd.Execute(); err != nil {
		t.Fatalf("export: %v", err)
	}

	// Import into fresh DB
	s2 := openTestStore(t, testDBPath(t))
	setDB(t, s2)

	icmd := newImportCmd()
	icmd.SetArgs([]string{"--no-context", exportPath})
	if err := icmd.Execute(); err != nil {
		t.Fatalf("import: %v", err)
	}

	// Verify
	got, err := db.GetNote(orig.ID)
	if err != nil {
		t.Fatalf("getting imported note: %v", err)
	}
	if got.Title != orig.Title {
		t.Errorf("title = %q, want %q", got.Title, orig.Title)
	}
	if got.Body != orig.Body {
		t.Errorf("body = %q, want %q", got.Body, orig.Body)
	}
}
