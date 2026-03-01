package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/danjdewhurst/jot-cli/internal/model"
	"github.com/danjdewhurst/jot-cli/internal/store"
	"github.com/spf13/cobra"
)

func testDBPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "test.db")
}

func openTestStore(t *testing.T, path string) *store.Store {
	t.Helper()
	s, err := store.Open(path)
	if err != nil {
		t.Fatalf("opening store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// setDB sets the package-level db variable for testing and restores it on cleanup.
func setDB(t *testing.T, s *store.Store) {
	t.Helper()
	old := db
	db = s
	t.Cleanup(func() { db = old })
}

// newExportCmd creates a fresh export cobra command for testing.
func newExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "export",
		RunE: runExport,
	}
	cmd.Flags().StringP("output", "o", "", "Output file path")
	cmd.Flags().StringP("format", "f", "json", "Export format: json, md")
	cmd.Flags().StringSlice("tag", nil, "Filter by tag (key:value)")
	cmd.Flags().Bool("archived", false, "Include archived notes")
	cmd.Flags().StringP("search", "s", "", "Filter by search query")
	cmd.Flags().String("since", "", "Only notes created after this date")
	cmd.Flags().String("until", "", "Only notes created before this date")
	return cmd
}

func TestExportJSON_AllNotes(t *testing.T) {
	s := openTestStore(t, testDBPath(t))
	setDB(t, s)

	for i, title := range []string{"Note A", "Note B", "Note C"} {
		if _, err := s.CreateNote(title, "Body "+string(rune('A'+i)), nil); err != nil {
			t.Fatalf("creating note: %v", err)
		}
	}

	outPath := filepath.Join(t.TempDir(), "export.json")
	cmd := newExportCmd()
	cmd.SetArgs([]string{"--format", "json", "--output", outPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("export: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading export: %v", err)
	}

	var envelope model.ExportEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("decoding export: %v", err)
	}

	if envelope.Version != model.ExportVersion {
		t.Errorf("version = %d, want %d", envelope.Version, model.ExportVersion)
	}
	if envelope.Count != 3 {
		t.Errorf("count = %d, want 3", envelope.Count)
	}
	if len(envelope.Notes) != 3 {
		t.Errorf("notes length = %d, want 3", len(envelope.Notes))
	}
}

func TestExportJSON_FilterByTag(t *testing.T) {
	s := openTestStore(t, testDBPath(t))
	setDB(t, s)

	if _, err := s.CreateNote("Tagged", "body", []model.Tag{{Key: "project", Value: "alpha"}}); err != nil {
		t.Fatalf("creating note: %v", err)
	}
	if _, err := s.CreateNote("Untagged", "body", nil); err != nil {
		t.Fatalf("creating note: %v", err)
	}

	outPath := filepath.Join(t.TempDir(), "export.json")
	cmd := newExportCmd()
	cmd.SetArgs([]string{"--format", "json", "--output", outPath, "--tag", "project:alpha"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("export: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading: %v", err)
	}

	var envelope model.ExportEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	if envelope.Count != 1 {
		t.Errorf("count = %d, want 1", envelope.Count)
	}
	if envelope.Notes[0].Title != "Tagged" {
		t.Errorf("title = %q, want %q", envelope.Notes[0].Title, "Tagged")
	}
}

func TestExportJSON_DateRange(t *testing.T) {
	s := openTestStore(t, testDBPath(t))
	setDB(t, s)

	if _, err := s.CreateNote("Recent", "body", nil); err != nil {
		t.Fatalf("creating note: %v", err)
	}

	outPath := filepath.Join(t.TempDir(), "export.json")
	tomorrow := time.Now().UTC().Add(24 * time.Hour).Format("2006-01-02")

	cmd := newExportCmd()
	cmd.SetArgs([]string{"--format", "json", "--output", outPath, "--since", tomorrow})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("export: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading: %v", err)
	}
	var envelope model.ExportEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	if envelope.Count != 0 {
		t.Errorf("count = %d, want 0 (all notes should be before --since)", envelope.Count)
	}
}

func TestExportMarkdown(t *testing.T) {
	s := openTestStore(t, testDBPath(t))
	setDB(t, s)

	if _, err := s.CreateNote("MD Note", "Markdown body", nil); err != nil {
		t.Fatalf("creating note: %v", err)
	}

	outPath := filepath.Join(t.TempDir(), "export.md")
	cmd := newExportCmd()
	cmd.SetArgs([]string{"--format", "md", "--output", outPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("export: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading: %v", err)
	}

	out := string(data)
	if !strings.Contains(out, "# MD Note") {
		t.Error("missing title heading")
	}
	if !strings.Contains(out, "Markdown body") {
		t.Error("missing body")
	}
}
