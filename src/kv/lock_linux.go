//go:build linux

package kv

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// acquireProcessLock acquires an exclusive non-blocking advisory file lock using flock on Linux.
func acquireProcessLock(lockPath string) (*os.File, error) {
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file %s: %w", lockPath, err)
	}
	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return nil, ErrBusy
		}
		return nil, fmt.Errorf("flock failed on %s: %w", lockPath, err)
	}
	return f, nil
}

// releaseProcessLock unlocks and closes the advisory lock file on Linux.
func releaseProcessLock(f *os.File) error {
	if f == nil {
		return nil
	}
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return f.Close()
}
