package sync

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/danjdewhurst/jot-cli/internal/store"
	"github.com/oklog/ulid/v2"
)

// SyncResult summarises what happened during a sync operation.
type SyncResult struct {
	Pushed    int       `json:"pushed"`
	Pulled    int       `json:"pulled"`
	Conflicts int       `json:"conflicts"`
	SyncedAt  time.Time `json:"synced_at"`
}

// StatusResult summarises the current sync state.
type StatusResult struct {
	Pending  int       `json:"pending"`
	LastSync time.Time `json:"last_sync,omitempty"`
}

// Syncer coordinates push/pull operations against a sync directory.
type Syncer struct {
	store      *store.Store
	syncDir    string
	passphrase string
}

// New creates a Syncer for the given store and sync directory.
func New(s *store.Store, syncDir string) *Syncer {
	return &Syncer{store: s, syncDir: syncDir}
}

// NewEncrypted creates a Syncer that encrypts changesets using the given passphrase.
func NewEncrypted(s *store.Store, syncDir, passphrase string) *Syncer {
	return &Syncer{store: s, syncDir: syncDir, passphrase: passphrase}
}

// Encrypted returns true if the syncer is configured for encryption.
func (s *Syncer) Encrypted() bool {
	return s.passphrase != ""
}

// SyncDir returns the path to the sync directory.
func (s *Syncer) SyncDir() string {
	return s.syncDir
}

// Sync performs a full push-then-pull cycle with file locking.
func (s *Syncer) Sync() (SyncResult, error) {
	if err := s.ensureSyncDir(); err != nil {
		return SyncResult{}, err
	}

	lock, err := acquireLock(s.syncDir)
	if err != nil {
		return SyncResult{}, err
	}
	defer lock.release()

	pushed, err := s.push()
	if err != nil {
		return SyncResult{}, fmt.Errorf("push: %w", err)
	}

	pulled, conflicts, err := s.pull()
	if err != nil {
		return SyncResult{}, fmt.Errorf("pull: %w", err)
	}

	now := time.Now().UTC()
	return SyncResult{
		Pushed:    pushed,
		Pulled:    pulled,
		Conflicts: conflicts,
		SyncedAt:  now,
	}, nil
}

// Status returns a summary of pending changes.
func (s *Syncer) Status() (StatusResult, error) {
	entries, err := s.store.UnsyncedChanges()
	if err != nil {
		return StatusResult{}, err
	}

	var lastSync time.Time
	if ts, err := s.store.GetSyncMeta("last_sync"); err == nil {
		lastSync, _ = time.Parse(time.RFC3339, ts)
	}

	return StatusResult{
		Pending:  len(entries),
		LastSync: lastSync,
	}, nil
}

// ensureSyncDir creates the sync directory structure if it doesn't exist.
func (s *Syncer) ensureSyncDir() error {
	return os.MkdirAll(filepath.Join(s.syncDir, "changesets"), 0o755)
}

// machineID returns (or generates) a stable machine identifier.
func (s *Syncer) machineID() (string, error) {
	id, err := s.store.GetSyncMeta("machine_id")
	if err == nil && id != "" {
		return id, nil
	}
	id = ulid.Make().String()
	return id, s.store.SetSyncMeta("machine_id", id)
}
