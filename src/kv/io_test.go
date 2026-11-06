package kv

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"
)

// mockFile is a test double for simulating edge-case I/O errors and zero-progress.
type mockFile struct {
	data      []byte
	zeroRead  bool
	zeroWrite bool
	syncErr   error
}

func (m *mockFile) ReadAt(p []byte, off int64) (int, error) {
	if m.zeroRead {
		return 0, nil
	}
	if off >= int64(len(m.data)) {
		return 0, io.EOF
	}
	n := copy(p, m.data[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

func (m *mockFile) WriteAt(p []byte, off int64) (int, error) {
	if m.zeroWrite {
		return 0, nil
	}
	end := off + int64(len(p))
	if end > int64(len(m.data)) {
		expanded := make([]byte, end)
		copy(expanded, m.data)
		m.data = expanded
	}
	n := copy(m.data[off:], p)
	return n, nil
}

func (m *mockFile) Sync() error {
	return m.syncErr
}

func (m *mockFile) Truncate(size int64) error {
	if size <= int64(len(m.data)) {
		m.data = m.data[:size]
	}
	return nil
}

func (m *mockFile) Stat() (os.FileInfo, error) {
	return nil, nil
}

func TestIO_RealFileWriteRead(t *testing.T) {
	tmp, err := os.CreateTemp("", "kipdb_io_test_*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	payload := []byte("hello-storage-core-world-data")
	offset := int64(32)

	// Write
	if err := WriteAtFull(tmp, payload, offset); err != nil {
		t.Fatalf("WriteAtFull failed: %v", err)
	}

	// Sync
	if err := SyncFile(tmp); err != nil {
		t.Fatalf("SyncFile failed: %v", err)
	}

	// Read
	readBuf := make([]byte, len(payload))
	if err := ReadAtFull(tmp, readBuf, offset); err != nil {
		t.Fatalf("ReadAtFull failed: %v", err)
	}
	if !bytes.Equal(readBuf, payload) {
		t.Fatalf("read buffer %q != written %q", readBuf, payload)
	}

	// Short read past EOF returns UnexpectedEOF
	oversized := make([]byte, 100)
	err = ReadAtFull(tmp, oversized, offset)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("expected io.ErrUnexpectedEOF, got %v", err)
	}

	// Truncate
	if err := TruncateFile(tmp, 32); err != nil {
		t.Fatalf("TruncateFile failed: %v", err)
	}
	fi, _ := tmp.Stat()
	if fi.Size() != 32 {
		t.Fatalf("expected size 32 after truncate, got %d", fi.Size())
	}
}

func TestIO_NegativeOffset(t *testing.T) {
	m := &mockFile{data: make([]byte, 100)}
	buf := make([]byte, 10)
	if err := ReadAtFull(m, buf, -1); !errors.Is(err, ErrNegativeOffset) {
		t.Fatalf("expected ErrNegativeOffset for read, got %v", err)
	}
	if err := WriteAtFull(m, buf, -5); !errors.Is(err, ErrNegativeOffset) {
		t.Fatalf("expected ErrNegativeOffset for write, got %v", err)
	}
}

func TestIO_ZeroProgressDetection(t *testing.T) {
	m := &mockFile{data: make([]byte, 100), zeroRead: true, zeroWrite: true}
	buf := make([]byte, 10)

	if err := ReadAtFull(m, buf, 0); !errors.Is(err, ErrZeroProgress) {
		t.Fatalf("expected ErrZeroProgress on read, got %v", err)
	}
	if err := WriteAtFull(m, buf, 0); !errors.Is(err, ErrZeroProgress) {
		t.Fatalf("expected ErrZeroProgress on write, got %v", err)
	}
}

func TestIO_SyncErrorPropagation(t *testing.T) {
	customErr := errors.New("simulated disk sync error")
	m := &mockFile{syncErr: customErr}
	if err := SyncFile(m); !errors.Is(err, ErrUncertain) {
		t.Fatalf("expected ErrUncertain wrapping sync error, got %v", err)
	}
}
