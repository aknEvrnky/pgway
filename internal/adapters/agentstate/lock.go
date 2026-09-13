package agentstate

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// Lock holds an exclusive non-blocking flock released on Close.
type Lock struct {
	path string
}

func NewLock(statePath string) *Lock {
	return &Lock{path: LockPath(statePath)}
}

func (l *Lock) Acquire() (io.Closer, error) {
	if err := os.MkdirAll(filepath.Dir(l.path), 0o700); err != nil {
		return nil, fmt.Errorf("create agent lock dir: %w", err)
	}

	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open agent lock %s: %w", l.path, err)
	}

	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("another pgway-dp is running (lock %s): %w", l.path, err)
	}

	return &heldLock{f: f}, nil
}

type heldLock struct {
	f *os.File
}

func (h *heldLock) Close() error {
	if h == nil || h.f == nil {
		return nil
	}
	_ = unix.Flock(int(h.f.Fd()), unix.LOCK_UN)
	err := h.f.Close()
	h.f = nil
	return err
}
