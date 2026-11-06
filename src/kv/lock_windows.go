//go:build windows

package kv

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const (
	lockfileExclusiveLock   = 0x00000002
	lockfileFailImmediately = 0x00000001
	errLockViolation        = syscall.Errno(33) // ERROR_LOCK_VIOLATION
)

var (
	modkernel32      = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = modkernel32.NewProc("LockFileEx")
	procUnlockFileEx = modkernel32.NewProc("UnlockFileEx")
)

// acquireProcessLock acquires an exclusive non-blocking advisory file lock on Windows using LockFileEx.
func acquireProcessLock(lockPath string) (*os.File, error) {
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file %s: %w", lockPath, err)
	}

	var ol syscall.Overlapped
	r1, _, errSys := procLockFileEx.Call(
		f.Fd(),
		uintptr(lockfileExclusiveLock|lockfileFailImmediately),
		0,
		1,
		0,
		uintptr(unsafe.Pointer(&ol)),
	)
	if r1 == 0 {
		f.Close()
		var errno syscall.Errno
		if errors.As(errSys, &errno) && errno == errLockViolation {
			return nil, ErrBusy
		}
		return nil, fmt.Errorf("LockFileEx failed on %s: %w", lockPath, errSys)
	}
	return f, nil
}

// releaseProcessLock unlocks and closes the advisory lock file on Windows.
func releaseProcessLock(f *os.File) error {
	if f == nil {
		return nil
	}
	var ol syscall.Overlapped
	_, _, _ = procUnlockFileEx.Call(
		f.Fd(),
		0,
		1,
		0,
		uintptr(unsafe.Pointer(&ol)),
	)
	return f.Close()
}
