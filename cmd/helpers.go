package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/danjdewhurst/jot-cli/internal/context"
	"github.com/danjdewhurst/jot-cli/internal/linking"
	"github.com/danjdewhurst/jot-cli/internal/model"
	"github.com/spf13/cobra"
)

// maxStdinSize is the maximum number of bytes read from stdin (1 MiB).
const maxStdinSize = 1 << 20

// parseTags reads the --tag flag from the command and parses each value
// into a model.Tag.
func parseTags(cmd *cobra.Command) ([]model.Tag, error) {
	tagStrs, _ := cmd.Flags().GetStringSlice("tag")
	var tags []model.Tag
	for _, s := range tagStrs {
		t, err := model.ParseTag(s)
		if err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, nil
}

// buildNoteFilter constructs a NoteFilter from common flags shared by
// the list and log commands (--tag, --folder, --repo, --branch, --archived,
// --pinned, --limit).
func buildNoteFilter(cmd *cobra.Command) (model.NoteFilter, error) {
	filter := model.NoteFilter{}

	tags, err := parseTags(cmd)
	if err != nil {
		return filter, err
	}
	filter.Tags = tags

	if f, _ := cmd.Flags().GetBool("folder"); f {
		if folder, err := context.DetectFolder(); err == nil && folder != "" {
			filter.Tags = append(filter.Tags, model.Tag{Key: "folder", Value: folder})
		}
	}
	if r, _ := cmd.Flags().GetBool("repo"); r {
		if repo, err := context.DetectRepo(); err == nil && repo != "" {
			filter.Tags = append(filter.Tags, model.Tag{Key: "git_repo", Value: repo})
		}
	}
	if b, _ := cmd.Flags().GetBool("branch"); b {
		if branch, err := context.DetectBranch(); err == nil && branch != "" {
			filter.Tags = append(filter.Tags, model.Tag{Key: "git_branch", Value: branch})
		}
	}

	archived, _ := cmd.Flags().GetBool("archived")
	filter.Archived = archived

	if cmd.Flags().Lookup("pinned") != nil {
		if pinned, _ := cmd.Flags().GetBool("pinned"); pinned {
			filter.PinnedOnly = true
		}
	}

	limit, _ := cmd.Flags().GetInt("limit")
	filter.Limit = limit

	return filter, nil
}

// resolveNote finds a note by full or prefix ID.
func resolveNote(idOrPrefix string) (model.Note, error) {
	// Try exact match first
	note, err := db.GetNote(idOrPrefix)
	if err == nil {
		return note, nil
	}

	// Try prefix match
	notes, err := db.ListNotes(model.NoteFilter{Archived: true})
	if err != nil {
		return model.Note{}, fmt.Errorf("listing notes: %w", err)
	}

	var matches []model.Note
	for _, n := range notes {
		if strings.HasPrefix(n.ID, idOrPrefix) {
			matches = append(matches, n)
		}
	}

	switch len(matches) {
	case 0:
		return model.Note{}, fmt.Errorf("no note matching %q", idOrPrefix)
	case 1:
		return matches[0], nil
	default:
		return model.Note{}, fmt.Errorf("ambiguous prefix %q matches %d notes", idOrPrefix, len(matches))
	}
}

// parseDate parses a date string in RFC 3339 or YYYY-MM-DD format.
// Returns zero time for empty input.
func parseDate(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("cannot parse %q as RFC 3339 or YYYY-MM-DD", s)
}

// syncNoteRefs scans a note body for @references, resolves them to full
// note IDs, and syncs the ref tags. Unresolvable references are logged
// as warnings to stderr. selfID is excluded from resolution to prevent
// self-references.
func syncNoteRefs(noteID, body string) {
	prefixes := linking.ExtractRefs(body)
	if len(prefixes) == 0 {
		_ = db.SyncRefs(noteID, nil)
		return
	}

	var resolved []string
	for _, prefix := range prefixes {
		note, err := resolveNote(prefix)
		if err != nil {
			if flagVerbose {
				fmt.Fprintf(os.Stderr, "Warning: could not resolve reference @%s: %v\n", prefix, err)
			}
			continue
		}
		if note.ID == noteID {
			continue // skip self-references
		}
		resolved = append(resolved, note.ID)
	}

	if err := db.SyncRefs(noteID, resolved); err != nil && flagVerbose {
		fmt.Fprintf(os.Stderr, "Warning: failed to sync references: %v\n", err)
	}
}

func filterByDateRange(notes []model.Note, since, until time.Time) []model.Note {
	if since.IsZero() && until.IsZero() {
		return notes
	}
	var filtered []model.Note
	for _, n := range notes {
		if !since.IsZero() && n.CreatedAt.Before(since) {
			continue
		}
		if !until.IsZero() && n.CreatedAt.After(until) {
			continue
		}
		filtered = append(filtered, n)
	}
	return filtered
}
