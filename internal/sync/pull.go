package sync

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Pull imports changesets from other machines.
// Returns (pulled, conflicts, err).
func (s *Syncer) Pull() (int, int, error) {
	if err := s.ensureSyncDir(); err != nil {
		return 0, 0, fmt.Errorf("creating sync directory: %w", err)
	}

	mid, err := s.machineID()
	if err != nil {
		return 0, 0, err
	}

	var lastSync time.Time
	if ts, err := s.store.GetSyncMeta("last_sync"); err == nil {
		lastSync, _ = time.Parse(time.RFC3339, ts)
	}

	changesetsDir := filepath.Join(s.syncDir, "changesets")
	dirEntries, err := os.ReadDir(changesetsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, nil
		}
		return 0, 0, fmt.Errorf("reading changesets directory: %w", err)
	}

	var pulled, conflicts int
	var importedNoteIDs []string

	for _, entry := range dirEntries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".ndjson") {
			continue
		}

		// Skip our own changesets.
		if strings.HasPrefix(name, mid+"_") {
			continue
		}

		// Skip files older than last sync. The timestamp is embedded after
		// the machine ID prefix: <machine_id>_<RFC3339>.ndjson
		idx := strings.Index(name, "_")
		if idx >= 0 {
			tsStr := strings.TrimSuffix(name[idx+1:], ".ndjson")
			if fileTime, err := time.Parse(time.RFC3339, tsStr); err == nil {
				if !lastSync.IsZero() && !fileTime.After(lastSync) {
					continue
				}
			}
		}

		p, c, noteIDs, err := s.processChangeset(filepath.Join(changesetsDir, name))
		if err != nil {
			return pulled, conflicts, fmt.Errorf("processing %s: %w", name, err)
		}
		pulled += p
		conflicts += c
		importedNoteIDs = append(importedNoteIDs, noteIDs...)
	}

	// Clear changelog entries created by triggers during import
	// to prevent re-exporting them on next push.
	if len(importedNoteIDs) > 0 {
		if err := s.store.ClearChangelogForNotes(importedNoteIDs); err != nil {
			return pulled, conflicts, err
		}
	}

	now := time.Now().UTC()
	if err := s.store.SetSyncMeta("last_sync", now.Format(time.RFC3339)); err != nil {
		return pulled, conflicts, err
	}

	return pulled, conflicts, nil
}

// processChangeset reads and applies a single changeset file.
// Returns (pulled, conflicts, importedNoteIDs, err).
func (s *Syncer) processChangeset(path string) (int, int, []string, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, nil, err
	}
	defer f.Close() //nolint:errcheck

	var pulled, conflicts int
	var importedNoteIDs []string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var ce ChangeEntry
		if err := json.Unmarshal(scanner.Bytes(), &ce); err != nil {
			return pulled, conflicts, importedNoteIDs, fmt.Errorf("decoding entry: %w", err)
		}

		switch ce.Action {
		case "upsert":
			if ce.Note == nil {
				continue
			}
			local, err := s.store.GetNote(ce.Note.ID)
			switch {
			case err != nil:
				// Note not found locally — import it.
				if err := s.store.UpsertNote(*ce.Note); err != nil {
					return pulled, conflicts, importedNoteIDs, fmt.Errorf("importing note %s: %w", ce.Note.ID, err)
				}
				pulled++
				importedNoteIDs = append(importedNoteIDs, ce.Note.ID)
			case ce.Note.UpdatedAt.After(local.UpdatedAt):
				// Remote is newer — overwrite.
				if err := s.store.UpsertNote(*ce.Note); err != nil {
					return pulled, conflicts, importedNoteIDs, fmt.Errorf("updating note %s: %w", ce.Note.ID, err)
				}
				pulled++
				importedNoteIDs = append(importedNoteIDs, ce.Note.ID)
			default:
				// Local is newer — keep ours.
				conflicts++
			}

		case "delete":
			noteID := ce.NoteID
			if noteID == "" && ce.Note != nil {
				noteID = ce.Note.ID
			}
			if noteID == "" {
				continue
			}

			local, err := s.store.GetNote(noteID)
			if err != nil {
				// Already gone locally.
				continue
			}

			deletedAt, _ := time.Parse(time.RFC3339, ce.DeletedAt)
			if !deletedAt.IsZero() && deletedAt.After(local.UpdatedAt) {
				if err := s.store.DeleteNote(noteID); err != nil {
					return pulled, conflicts, importedNoteIDs, fmt.Errorf("deleting note %s: %w", noteID, err)
				}
				pulled++
				importedNoteIDs = append(importedNoteIDs, noteID)
			}
			// If local was edited after the delete, keep it.
		}
	}

	return pulled, conflicts, importedNoteIDs, scanner.Err()
}
