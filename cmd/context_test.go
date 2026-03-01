package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/danjdewhurst/jot-cli/internal/model"
	"github.com/spf13/cobra"
)

func newContextCmd() *cobra.Command {
	return &cobra.Command{
		Use:  "context",
		RunE: runContext,
	}
}

func TestContextCmd_Table(t *testing.T) {
	var buf bytes.Buffer
	cmd := newContextCmd()
	cmd.SetOut(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("context: %v", err)
	}

	out := buf.String()

	// Should always contain the folder key (we're running in a directory)
	if !strings.Contains(out, "folder") {
		t.Errorf("expected 'folder' in output, got %q", out)
	}
}

func TestContextCmd_MissingValues(t *testing.T) {
	var buf bytes.Buffer
	cmd := newContextCmd()
	cmd.SetOut(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("context: %v", err)
	}

	out := buf.String()

	// All three keys should always be present
	for _, key := range []string{"folder", "git_repo", "git_branch"} {
		if !strings.Contains(out, key) {
			t.Errorf("expected %q key in output, got %q", key, out)
		}
	}
}

func TestContextCmd_JSON(t *testing.T) {
	oldJSON := flagJSON
	flagJSON = true
	t.Cleanup(func() { flagJSON = oldJSON })

	var buf bytes.Buffer
	cmd := newContextCmd()
	cmd.SetOut(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("context: %v", err)
	}

	var tags []model.Tag
	if err := json.Unmarshal(buf.Bytes(), &tags); err != nil {
		t.Fatalf("decoding JSON: %v (output: %q)", err, buf.String())
	}

	// Should have at least the folder tag
	found := false
	for _, tag := range tags {
		if tag.Key == "folder" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected folder tag in JSON output, got %v", tags)
	}
}
