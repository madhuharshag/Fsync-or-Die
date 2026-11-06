//go:build windows

package kv

import (
	"fmt"
	"os"
)

// acquireProcessLock on native Windows explicitly rejects execution.
// The project's official target and judging environment is Linux under WSL2.
// Running natively on Windows without POSIX file replacement and flock semantics is unsupported.
func acquireProcessLock(lockPath string) (*os.File, error) {
	return nil, fmt.Errorf("%w: native Windows execution is unsupported; run inside Linux under WSL2", ErrConfig)
}

// releaseProcessLock closes the lock handle if present.
func releaseProcessLock(f *os.File) error {
	if f != nil {
		return f.Close()
	}
	return nil
}
