package sync

import (
	"bufio"
	"bytes"
	"crypto/sha256"
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

	lock, err := acquireLock(s.syncDir)
	if err != nil {
		return 0, 0, err
	}
	defer lock.release()

	return s.pull()
}

// pull is the internal implementation without locking.
func (s *Syncer) pull() (int, int, error) {

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

		// Accept .ndjson and .ndjson.age files.
		encrypted := strings.HasSuffix(name, ".ndjson.age")
		if !encrypted && !strings.HasSuffix(name, ".ndjson") {
			continue
		}

		// Skip our own changesets.
		if strings.HasPrefix(name, mid+"_") {
			continue
		}

		// Skip files older than last sync. The timestamp is embedded after
		// the machine ID prefix: <machine_id>_<RFC3339>.ndjson[.age]
		idx := strings.Index(name, "_")
		if idx >= 0 {
			tsStr := name[idx+1:]
			tsStr = strings.TrimSuffix(tsStr, ".age")
			tsStr = strings.TrimSuffix(tsStr, ".ndjson")
			if fileTime, err := time.Parse(time.RFC3339, tsStr); err == nil {
				if !lastSync.IsZero() && fileTime.Before(lastSync) {
					continue
				}
			}
		}

		p, c, noteIDs, err := s.processChangeset(filepath.Join(changesetsDir, name), encrypted)
		if err != nil {
			return pulled, conflicts, fmt.Errorf("processing %s: %w", name, err)
		}
		pulled += p
		conflicts += c
		importedNoteIDs = append(importedNoteIDs, noteIDs...)

		// Update last_sync after each successfully processed changeset for
		// resumability — if a later file fails, we won't re-process this one.
		now := time.Now().UTC()
		if err := s.store.SetSyncMeta("last_sync", now.Format(time.RFC3339)); err != nil {
			return pulled, conflicts, err
		}
	}

	// Clear changelog entries created by triggers during import
	// to prevent re-exporting them on next push.
	if len(importedNoteIDs) > 0 {
		if err := s.store.ClearChangelogForNotes(importedNoteIDs); err != nil {
			return pulled, conflicts, err
		}
	}

	return pulled, conflicts, nil
}

// processChangeset reads and applies a single changeset file.
// Returns (pulled, conflicts, importedNoteIDs, err).
func (s *Syncer) processChangeset(path string, encrypted bool) (int, int, []string, error) {
	var reader *bufio.Scanner

	if encrypted {
		data, err := os.ReadFile(path)
		if err != nil {
			return 0, 0, nil, fmt.Errorf("reading encrypted file: %w", err)
		}
		plaintext, err := Decrypt(s.passphrase, data)
		if err != nil {
			return 0, 0, nil, fmt.Errorf("decrypting changeset: %w", err)
		}
		reader = bufio.NewScanner(bytes.NewReader(plaintext))
	} else {
		f, err := os.Open(path)
		if err != nil {
			return 0, 0, nil, err
		}
		defer f.Close() //nolint:errcheck
		reader = bufio.NewScanner(f)
	}

	reader.Buffer(make([]byte, 0, 1024*1024), 1024*1024) // 1MB max line

	var pulled, conflicts int
	var importedNoteIDs []string

	for reader.Scan() {
		var ce ChangeEntry
		if err := json.Unmarshal(reader.Bytes(), &ce); err != nil {
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
			case ce.Note.UpdatedAt.Equal(local.UpdatedAt):
				// Equal timestamps — use deterministic tiebreaker.
				// Lower body hash wins to ensure both machines converge.
				remoteHash := fmt.Sprintf("%x", sha256.Sum256([]byte(ce.Note.Body)))
				localHash := fmt.Sprintf("%x", sha256.Sum256([]byte(local.Body)))
				if remoteHash < localHash {
					if err := s.store.UpsertNote(*ce.Note); err != nil {
						return pulled, conflicts, importedNoteIDs, fmt.Errorf("updating note %s: %w", ce.Note.ID, err)
					}
					pulled++
					importedNoteIDs = append(importedNoteIDs, ce.Note.ID)
				} else {
					// Local hash wins (or hashes are identical — no change needed).
					conflicts++
				}
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

	return pulled, conflicts, importedNoteIDs, reader.Err()
}
