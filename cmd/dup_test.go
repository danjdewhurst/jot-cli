package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/danjdewhurst/jot-cli/internal/model"
	"github.com/spf13/cobra"
)

func newDupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "dup",
		Args: cobra.ExactArgs(1),
		RunE: dupCmd.RunE,
	}
	return cmd
}

func TestDup_CopiesTitleAndBody(t *testing.T) {
	s := openTestStore(t, testDBPath(t))
	setDB(t, s)

	orig, err := s.CreateNote("My Title", "My Body", nil)
	if err != nil {
		t.Fatalf("creating note: %v", err)
	}

	cmd := newDupCmd()
	cmd.SetArgs([]string{orig.ID})

	// Capture stdout for the printed ID
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = oldStdout })

	if err := cmd.Execute(); err != nil {
		t.Fatalf("dup should succeed: %v", err)
	}
	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	os.Stdout = oldStdout

	// List notes and find the duplicate
	notes, err := s.ListNotes(model.NoteFilter{})
	if err != nil {
		t.Fatalf("listing notes: %v", err)
	}
	if len(notes) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(notes))
	}

	var dup model.Note
	for _, n := range notes {
		if n.ID != orig.ID {
			dup = n
		}
	}

	if dup.Title != "My Title" {
		t.Errorf("title = %q, want %q", dup.Title, "My Title")
	}
	if dup.Body != "My Body" {
		t.Errorf("body = %q, want %q", dup.Body, "My Body")
	}
	if dup.ID == orig.ID {
		t.Error("duplicate should have a different ID")
	}
}

func TestDup_CopiesUserTags_ExcludesAutoContext(t *testing.T) {
	s := openTestStore(t, testDBPath(t))
	setDB(t, s)

	orig, err := s.CreateNote("Tagged", "body", []model.Tag{
		{Key: "project", Value: "alpha"},
		{Key: "priority", Value: "high"},
		{Key: "folder", Value: "/old/path"},
		{Key: "git_repo", Value: "old-repo"},
		{Key: "git_branch", Value: "old-branch"},
	})
	if err != nil {
		t.Fatalf("creating note: %v", err)
	}

	cmd := newDupCmd()
	cmd.SetArgs([]string{orig.ID})

	// Suppress stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = oldStdout })

	if err := cmd.Execute(); err != nil {
		t.Fatalf("dup should succeed: %v", err)
	}
	_ = w.Close()
	os.Stdout = oldStdout

	notes, err := s.ListNotes(model.NoteFilter{})
	if err != nil {
		t.Fatalf("listing notes: %v", err)
	}

	var dup model.Note
	for _, n := range notes {
		if n.ID != orig.ID {
			dup = n
		}
	}

	// Build a map of duplicated tags for easy lookup
	tagMap := make(map[string]string)
	for _, tag := range dup.Tags {
		tagMap[tag.Key] = tag.Value
	}

	// User tags should be copied
	if tagMap["project"] != "alpha" {
		t.Errorf("expected project:alpha tag, got %q", tagMap["project"])
	}
	if tagMap["priority"] != "high" {
		t.Errorf("expected priority:high tag, got %q", tagMap["priority"])
	}

	// Auto-context tags from the original should NOT be copied verbatim
	if v, ok := tagMap["folder"]; ok && v == "/old/path" {
		t.Error("old folder tag should not be copied; should be regenerated from current environment")
	}
	if v, ok := tagMap["git_repo"]; ok && v == "old-repo" {
		t.Error("old git_repo tag should not be copied; should be regenerated from current environment")
	}
	if v, ok := tagMap["git_branch"]; ok && v == "old-branch" {
		t.Error("old git_branch tag should not be copied; should be regenerated from current environment")
	}
}

func TestDup_JSONOutput(t *testing.T) {
	s := openTestStore(t, testDBPath(t))
	setDB(t, s)

	orig, err := s.CreateNote("JSON Dup", "body", nil)
	if err != nil {
		t.Fatalf("creating note: %v", err)
	}

	oldJSON := flagJSON
	flagJSON = true
	t.Cleanup(func() { flagJSON = oldJSON })

	cmd := newDupCmd()
	cmd.SetArgs([]string{orig.ID})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = oldStdout })

	if err := cmd.Execute(); err != nil {
		t.Fatalf("dup --json should succeed: %v", err)
	}
	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	os.Stdout = oldStdout

	var got model.Note
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("decoding JSON: %v (output: %q)", err, buf.String())
	}

	if got.ID == orig.ID {
		t.Error("JSON output ID should differ from original")
	}
	if got.Title != "JSON Dup" {
		t.Errorf("title = %q, want %q", got.Title, "JSON Dup")
	}
}

func TestDup_MissingID(t *testing.T) {
	s := openTestStore(t, testDBPath(t))
	setDB(t, s)

	cmd := newDupCmd()
	cmd.SetArgs([]string{"nonexistent"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	if err := cmd.Execute(); err == nil {
		t.Error("expected error for nonexistent note")
	}
}
