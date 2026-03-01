package sync

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

type lockFile struct {
	f *os.File
}

// acquireLock acquires an advisory lock on the sync directory.
func acquireLock(syncDir string) (*lockFile, error) {
	lockPath := filepath.Join(syncDir, ".lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("opening lock file: %w", err)
	}

	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("another jot sync is already running (could not acquire lock on %s)", lockPath)
	}

	return &lockFile{f: f}, nil
}

// release releases the advisory lock.
func (l *lockFile) release() {
	if l.f != nil {
		_ = unix.Flock(int(l.f.Fd()), unix.LOCK_UN)
		_ = l.f.Close()
	}
}
