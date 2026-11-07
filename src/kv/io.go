package kv

import (
	"errors"
	"fmt"
	"io"
	"os"
)

var (
	// ErrZeroProgress indicates an I/O operation made no forward progress (0 bytes transferred with nil error).
	ErrZeroProgress = errors.New("kv: zero progress during I/O transfer")

	// ErrNegativeOffset is returned when attempting an I/O operation at an invalid negative file offset.
	ErrNegativeOffset = errors.New("kv: negative file offset invalid")
)

// FileReadWriter defines the testable positioned file I/O interface implemented by *os.File.
type FileReadWriter interface {
	ReadAt(p []byte, off int64) (n int, err error)
	WriteAt(p []byte, off int64) (n int, err error)
	Sync() error
	Truncate(size int64) error
	Stat() (os.FileInfo, error)
}

// ReadAtFull reads exactly len(buf) bytes from f starting at off.
// If premature EOF or zero-progress is detected, it returns an explicit descriptive error.
func ReadAtFull(f FileReadWriter, buf []byte, off int64) error {
	if off < 0 {
		return ErrNegativeOffset
	}
	total := len(buf)
	read := 0
	for read < total {
		n, err := f.ReadAt(buf[read:], off+int64(read))
		if n > 0 {
			read += n
		}
		if err != nil {
			if errors.Is(err, io.EOF) && read < total {
				return fmt.Errorf("%w: read %d of %d requested bytes at offset %d: %v", io.ErrUnexpectedEOF, read, total, off, err)
			}
			return fmt.Errorf("read error at offset %d: %w", off+int64(read), err)
		}
		if n == 0 {
			return fmt.Errorf("%w: 0 bytes read at offset %d", ErrZeroProgress, off+int64(read))
		}
	}
	return nil
}

// WriteAtFull writes exactly len(buf) bytes to f starting at off.
// Loops until all bytes are transferred, detecting zero progress and propagating low-level errors.
func WriteAtFull(f FileReadWriter, buf []byte, off int64) error {
	if off < 0 {
		return ErrNegativeOffset
	}
	total := len(buf)
	written := 0
	for written < total {
		n, err := f.WriteAt(buf[written:], off+int64(written))
		if n > 0 {
			written += n
		}
		if err != nil {
			return fmt.Errorf("write failure after %d of %d bytes at offset %d: %w", written, total, off, err)
		}
		if n == 0 {
			return fmt.Errorf("%w: 0 bytes written at offset %d", ErrZeroProgress, off+int64(written))
		}
	}
	return nil
}

// SyncFile flushes OS file system buffers to non-volatile storage.
func SyncFile(f FileReadWriter) error {
	if err := f.Sync(); err != nil {
		return fmt.Errorf("%w: failed to synchronize file buffers: %v", ErrUncertain, err)
	}
	return nil
}

// TruncateFile truncates the file to the specified size in bytes.
func TruncateFile(f FileReadWriter, size int64) error {
	if size < 0 {
		return fmt.Errorf("%w: negative truncate size %d", ErrInvalid, size)
	}
	if err := f.Truncate(size); err != nil {
		return fmt.Errorf("truncate to %d bytes failed: %w", size, err)
	}
	return nil
}
