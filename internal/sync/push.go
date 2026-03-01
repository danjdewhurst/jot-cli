package sync

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Push exports unsynced changes to the sync directory.
// Returns the number of entries pushed.
func (s *Syncer) Push() (int, error) {
	if err := s.ensureSyncDir(); err != nil {
		return 0, fmt.Errorf("creating sync directory: %w", err)
	}

	entries, err := s.store.UnsyncedChanges()
	if err != nil {
		return 0, err
	}
	if len(entries) == 0 {
		return 0, nil
	}

	// Deduplicate: keep only the latest entry per note_id.
	seen := make(map[string]int) // note_id → index in deduped
	var deduped []ChangeEntry
	for _, e := range entries {
		ce := ChangeEntry{
			Action:  e.Action,
			NoteID:  e.NoteID,
		}

		if e.Action == "upsert" {
			note, err := s.store.GetNote(e.NoteID)
			if err != nil {
				// Note may have been deleted after the upsert was logged;
				// skip this entry — the delete trigger will have its own entry.
				continue
			}
			ce.Note = &note
		} else {
			ce.DeletedAt = e.ChangedAt
		}

		if idx, ok := seen[e.NoteID]; ok {
			deduped[idx] = ce // replace with latest
		} else {
			seen[e.NoteID] = len(deduped)
			deduped = append(deduped, ce)
		}
	}

	if len(deduped) == 0 {
		// All entries were for notes that no longer exist and had no delete entries.
		maxID := entries[len(entries)-1].ID
		return 0, s.store.MarkChangesSynced(maxID)
	}

	mid, err := s.machineID()
	if err != nil {
		return 0, err
	}

	now := time.Now().UTC()
	filename := fmt.Sprintf("%s_%s.ndjson", mid, now.Format(time.RFC3339))
	path := filepath.Join(s.syncDir, "changesets", filename)

	f, err := os.Create(path)
	if err != nil {
		return 0, fmt.Errorf("creating changeset file: %w", err)
	}
	defer f.Close() //nolint:errcheck

	enc := json.NewEncoder(f)
	for _, ce := range deduped {
		if err := enc.Encode(ce); err != nil {
			return 0, fmt.Errorf("writing changeset entry: %w", err)
		}
	}

	if err := f.Close(); err != nil {
		return 0, fmt.Errorf("closing changeset file: %w", err)
	}

	// Mark all processed changelog entries as synced.
	maxID := entries[len(entries)-1].ID
	if err := s.store.MarkChangesSynced(maxID); err != nil {
		return 0, err
	}

	if err := s.store.SetSyncMeta("last_sync", now.Format(time.RFC3339)); err != nil {
		return 0, err
	}

	return len(deduped), nil
}
