package agentruntime

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// Lock holds an exclusive non-blocking flock that is released on Close
// (and automatically by the kernel on process death).
type Lock struct {
	f *os.File
}

// AcquireLock opens (or creates) the lock file and takes LOCK_EX|LOCK_NB.
// Returns an error if another process already holds the lock.
func AcquireLock(lockPath string) (*Lock, error) {
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o700); err != nil {
		return nil, fmt.Errorf("create agent lock dir: %w", err)
	}

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open agent lock %s: %w", lockPath, err)
	}

	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("another pgway-dp is running (lock %s): %w", lockPath, err)
	}

	return &Lock{f: f}, nil
}

// Close releases the flock and closes the lock file.
func (l *Lock) Close() error {
	if l == nil || l.f == nil {
		return nil
	}
	_ = unix.Flock(int(l.f.Fd()), unix.LOCK_UN)
	err := l.f.Close()
	l.f = nil
	return err
}
