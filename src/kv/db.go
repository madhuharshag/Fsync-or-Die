package kv

import (
	"errors"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DB represents an open Fsync-or-Die database instance.
// DB implements the Engine interface. All public methods serialize through mu.
type DB struct {
	mu sync.Mutex

	path     string
	file     *os.File // Primary authoritative data file
	lockFile *os.File // Advisory process lock file (<path>.lock)
	index    *Index   // RAM key-to-offset index

	committedEnd    int64  // Byte offset of the last validated committed transaction
	nextTxID        uint64 // Next sequential transaction ID to assign
	txCount         uint64 // Number of committed transactions in this file
	compactionCount uint64 // Number of successful compactions executed
	maxFileBytes    uint64 // Configured file size limit (0 = unlimited)
	readOnly        bool   // If true, mutation operations are rejected

	closed   bool  // Marked true after Close() finishes
	poisoned error // Non-nil if a fatal, unrecoverable I/O failure occurred

	syncSeam func(f *os.File) error // Internal test seam: allows injecting synchronization failures in tests
}

// Open initializes or reopens a database at path with the specified options.
func Open(path string, opts Options) (Engine, OpenInfo, error) {
	start := time.Now()
	if path == "" {
		return nil, OpenInfo{}, fmt.Errorf("%w: empty database path", ErrInvalid)
	}

	// 1. Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, OpenInfo{}, fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// 2. Acquire process-level exclusive advisory lock
	lockPath := path + ".lock"
	lockFile, err := acquireProcessLock(lockPath)
	if err != nil {
		return nil, OpenInfo{}, err
	}

	// 3. Open authoritative data file (NO O_TRUNC, NO O_APPEND)
	openFlags := os.O_RDWR | os.O_CREATE
	if opts.ReadOnly {
		openFlags = os.O_RDONLY
	}

	dataFile, err := os.OpenFile(path, openFlags, 0644)
	if err != nil {
		_ = releaseProcessLock(lockFile)
		return nil, OpenInfo{}, fmt.Errorf("failed to open data file %s: %w", path, err)
	}

	fi, err := dataFile.Stat()
	if err != nil {
		_ = dataFile.Close()
		_ = releaseProcessLock(lockFile)
		return nil, OpenInfo{}, fmt.Errorf("failed to stat data file: %w", err)
	}

	db := &DB{
		path:         path,
		file:         dataFile,
		lockFile:     lockFile,
		index:        NewIndex(),
		maxFileBytes: opts.MaxFileBytes,
		readOnly:     opts.ReadOnly,
		nextTxID:     1,
		committedEnd: 0,
	}

	var openInfo OpenInfo

	if fi.Size() == 0 {
		// New file initialization
		if opts.ReadOnly {
			_ = db.Close()
			return nil, OpenInfo{}, fmt.Errorf("%w: cannot initialize new database in read-only mode", ErrInvalid)
		}
		header := NewFileHeader(opts.MaxFileBytes)
		headerBytes := EncodeFileHeader(header)
		if err := WriteAtFull(db.file, headerBytes, 0); err != nil {
			_ = db.Close()
			return nil, OpenInfo{}, fmt.Errorf("failed to write initial file header: %w", err)
		}
		if err := SyncFile(db.file); err != nil {
			_ = db.Close()
			return nil, OpenInfo{}, err
		}
		db.committedEnd = FileHeaderSize
		db.nextTxID = 1
		openInfo.Recreated = true
		openInfo.LastTxID = 0
		openInfo.LiveKeyCount = 0
	} else {
		// Reopening existing file: read and validate immutable file header
		if fi.Size() < FileHeaderSize {
			_ = db.Close()
			return nil, OpenInfo{}, fmt.Errorf("%w: file size %d smaller than 32-byte header", ErrCorrupt, fi.Size())
		}
		headerBuf := make([]byte, FileHeaderSize)
		if err := ReadAtFull(db.file, headerBuf, 0); err != nil {
			_ = db.Close()
			return nil, OpenInfo{}, fmt.Errorf("%w: failed to read file header: %v", ErrCorrupt, err)
		}
		header, err := DecodeFileHeader(headerBuf)
		if err != nil {
			_ = db.Close()
			return nil, OpenInfo{}, err
		}

		// Persisted-cap policy: if caller specifies 0 (unlimited), adopt persisted cap from header
		if db.maxFileBytes == 0 && header.MaxFileBytes > 0 {
			db.maxFileBytes = header.MaxFileBytes
		}

		db.committedEnd = FileHeaderSize

		// Recover transactions and repair any uncommitted tail
		info, err := db.recoverAndRepairLocked()
		if err != nil {
			_ = db.Close()
			return nil, OpenInfo{}, fmt.Errorf("recovery failed: %w", err)
		}
		openInfo = info
	}

	openInfo.RecoveryDurationNs = time.Since(start).Nanoseconds()
	return db, openInfo, nil
}

// Close flushes buffers, releases locks, and marks the database closed.
func (db *DB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return nil
	}
	db.closed = true

	var errs []error
	if db.file != nil {
		if !db.readOnly {
			if err := db.file.Sync(); err != nil {
				errs = append(errs, err)
			}
		}
		if err := db.file.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if db.lockFile != nil {
		if err := releaseProcessLock(db.lockFile); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// checkStateLocked verifies that the database is usable. Must be called with db.mu held.
func (db *DB) checkStateLocked() error {
	if db.closed {
		return ErrClosed
	}
	if db.poisoned != nil {
		return fmt.Errorf("%w: %w", ErrPoisoned, db.poisoned)
	}
	return nil
}

// Stats returns a snapshot of runtime metrics.
func (db *DB) Stats() Stats {
	db.mu.Lock()
	defer db.mu.Unlock()

	var size uint64
	if db.file != nil {
		if fi, err := db.file.Stat(); err == nil {
			size = uint64(fi.Size())
		}
	}

	return Stats{
		LiveKeys:        uint64(db.index.Len()),
		FileSizeBytes:   size,
		MaxFileBytes:    db.maxFileBytes,
		LastTxID:        db.nextTxID - 1,
		TxCount:         db.txCount,
		CompactionCount: db.compactionCount,
	}
}

// Check validates format, header, and sequential transactions across the file.
func (db *DB) Check() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	if err := db.checkStateLocked(); err != nil {
		return err
	}

	fi, err := db.file.Stat()
	if err != nil {
		return fmt.Errorf("check: stat failed: %w", err)
	}
	if fi.Size() < FileHeaderSize {
		return fmt.Errorf("%w: check: file size %d smaller than header", ErrCorrupt, fi.Size())
	}
	if fi.Size() != db.committedEnd {
		return fmt.Errorf("%w: check: physical file size %d does not match committed end %d", ErrCorrupt, fi.Size(), db.committedEnd)
	}

	// 1. Verify FileHeader
	headerBuf := make([]byte, FileHeaderSize)
	if err := ReadAtFull(db.file, headerBuf, 0); err != nil {
		return fmt.Errorf("check: read file header: %w", err)
	}
	if _, err := DecodeFileHeader(headerBuf); err != nil {
		return fmt.Errorf("check: decode file header: %w", err)
	}

	// 2. Scan and verify each committed transaction
	offset := int64(FileHeaderSize)
	for offset < db.committedEnd {
		if offset+int64(TxnHeaderSize) > db.committedEnd {
			return fmt.Errorf("%w: check: truncated transaction header at offset %d", ErrCorrupt, offset)
		}
		txnHeaderBuf := make([]byte, TxnHeaderSize)
		if err := ReadAtFull(db.file, txnHeaderBuf, offset); err != nil {
			return fmt.Errorf("check: read txn header at %d: %w", offset, err)
		}
		txnHeader, err := DecodeTxnHeader(txnHeaderBuf)
		if err != nil {
			return fmt.Errorf("check: decode txn header at %d: %w", offset, err)
		}

		totalTxBytes := int64(TxnHeaderSize) + int64(txnHeader.PayloadLen) + int64(CommitFooterSize)
		if offset+totalTxBytes > db.committedEnd {
			return fmt.Errorf("%w: check: transaction at %d exceeds committed end %d", ErrCorrupt, offset, db.committedEnd)
		}

		payloadBuf := make([]byte, txnHeader.PayloadLen)
		if err := ReadAtFull(db.file, payloadBuf, offset+int64(TxnHeaderSize)); err != nil {
			return fmt.Errorf("check: read payload at %d: %w", offset, err)
		}

		// Verify individual operations in payload
		cur := 0
		for cur < len(payloadBuf) {
			_, consumed, opErr := DecodeOp(payloadBuf[cur:])
			if opErr != nil {
				return fmt.Errorf("check: decode op in txn %d at offset %d: %w", txnHeader.TxID, offset+int64(TxnHeaderSize)+int64(cur), opErr)
			}
			cur += consumed
		}

		footerBuf := make([]byte, CommitFooterSize)
		footerOffset := offset + int64(TxnHeaderSize) + int64(txnHeader.PayloadLen)
		if err := ReadAtFull(db.file, footerBuf, footerOffset); err != nil {
			return fmt.Errorf("check: read footer at %d: %w", footerOffset, err)
		}
		footer, err := DecodeCommitFooter(footerBuf)
		if err != nil {
			return fmt.Errorf("check: decode footer at %d: %w", footerOffset, err)
		}

		if footer.TxID != txnHeader.TxID {
			return fmt.Errorf("%w: check: footer TxID %d mismatch header TxID %d at offset %d", ErrCorrupt, footer.TxID, txnHeader.TxID, offset)
		}
		if footer.TotalTxBytes != uint64(totalTxBytes) {
			return fmt.Errorf("%w: check: footer total bytes %d mismatch calculated %d at offset %d", ErrCorrupt, footer.TotalTxBytes, totalTxBytes, offset)
		}

		// Dual CRC verification
		h := crc32.NewIEEE()
		h.Write(txnHeaderBuf)
		h.Write(payloadBuf)
		computedTxnCRC := h.Sum32()
		if computedTxnCRC != footer.TxnCRC {
			return fmt.Errorf("%w: check: txn CRC mismatch at %d (expected %08x, got %08x)", ErrCorrupt, offset, footer.TxnCRC, computedTxnCRC)
		}

		computedFooterCRC := crc32.ChecksumIEEE(footerBuf[0:28])
		if computedFooterCRC != footer.FooterCRC {
			return fmt.Errorf("%w: check: footer CRC mismatch at %d (expected %08x, got %08x)", ErrCorrupt, footerOffset, footer.FooterCRC, computedFooterCRC)
		}

		offset += totalTxBytes
	}

	if offset != db.committedEnd {
		return fmt.Errorf("%w: check: traversed offset %d != committedEnd %d", ErrCorrupt, offset, db.committedEnd)
	}

	return nil
}

// Scan iterates through all live entries. fn receives caller-owned slices.
// fn must not reenter db.
func (db *DB) Scan(fn func(key, value []byte) error) error {
	db.mu.Lock()
	if err := db.checkStateLocked(); err != nil {
		db.mu.Unlock()
		return err
	}

	// Collect live keys and offsets while holding lock
	type readTarget struct {
		key   []byte
		entry IndexEntry
	}
	targets := make([]readTarget, 0, db.index.Len())
	_ = db.index.ForEach(func(k []byte, e IndexEntry) error {
		targets = append(targets, readTarget{key: k, entry: e})
		return nil
	})
	db.mu.Unlock()

	// Perform individual reads and invoke callback outside mutex lock
	for _, target := range targets {
		val, err := db.Get(target.key)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				// Key was deleted concurrently
				continue
			}
			return err
		}
		if err := fn(target.key, val); err != nil {
			return err
		}
	}
	return nil
}

// Compact delegates to Member 2's crash-safe compaction implementation.
func (db *DB) Compact() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	if err := db.checkStateLocked(); err != nil {
		return err
	}
	return fmt.Errorf("%w: crash-safe compaction is owned by Member 2", ErrInvalid)
}

// recoverAndRepairLocked performs sequential recovery of committed transactions
// from offset 32 to end of file, rebuilding the RAM index and repairing any torn tail.
func (db *DB) recoverAndRepairLocked() (OpenInfo, error) {
	fi, err := db.file.Stat()
	if err != nil {
		return OpenInfo{}, err
	}
	fileSize := fi.Size()
	offset := int64(FileHeaderSize)
	var lastTxID uint64
	var txCount uint64

	for offset < fileSize {
		// Attempt reading TxnHeader
		if offset+int64(TxnHeaderSize) > fileSize {
			// Incomplete tail header
			break
		}
		headerBuf := make([]byte, TxnHeaderSize)
		if err := ReadAtFull(db.file, headerBuf, offset); err != nil {
			break
		}
		txnHeader, err := DecodeTxnHeader(headerBuf)
		if err != nil {
			// Non-transaction bytes or corrupt header
			break
		}

		totalTxBytes := int64(TxnHeaderSize) + int64(txnHeader.PayloadLen) + int64(CommitFooterSize)
		if offset+totalTxBytes > fileSize {
			// Incomplete transaction payload/footer
			break
		}

		// Read payload
		payloadBuf := make([]byte, txnHeader.PayloadLen)
		if err := ReadAtFull(db.file, payloadBuf, offset+int64(TxnHeaderSize)); err != nil {
			break
		}

		// Read footer
		footerBuf := make([]byte, CommitFooterSize)
		footerOffset := offset + int64(TxnHeaderSize) + int64(txnHeader.PayloadLen)
		if err := ReadAtFull(db.file, footerBuf, footerOffset); err != nil {
			break
		}
		footer, err := DecodeCommitFooter(footerBuf)
		if err != nil {
			break
		}

		// Verify matching TxID and total length
		if footer.TxID != txnHeader.TxID || footer.TotalTxBytes != uint64(totalTxBytes) {
			break
		}

		// Verify dual CRC: TxnCRC over (TxnHeader + Payload)
		h := crc32.NewIEEE()
		h.Write(headerBuf)
		h.Write(payloadBuf)
		computedTxnCRC := h.Sum32()
		if computedTxnCRC != footer.TxnCRC {
			break
		}

		// Verify FooterCRC over footer bytes [0..27]
		computedFooterCRC := crc32.ChecksumIEEE(footerBuf[0:28])
		if computedFooterCRC != footer.FooterCRC {
			break
		}

		// Parse operations and collect index updates
		payloadOffset := offset + int64(TxnHeaderSize)
		cur := 0
		var updates []PendingIndexUpdate
		validOps := true
		for cur < len(payloadBuf) {
			op, consumed, opErr := DecodeOp(payloadBuf[cur:])
			if opErr != nil {
				validOps = false
				break
			}
			valOffset := payloadOffset + int64(cur) + int64(OpHeaderSize) + int64(len(op.Key))
			if op.Kind == DeleteOp {
				updates = append(updates, PendingIndexUpdate{Key: op.Key, IsDelete: true})
			} else {
				updates = append(updates, PendingIndexUpdate{
					Key: op.Key,
					Entry: IndexEntry{
						ValueOffset: valOffset,
						ValueLength: uint32(len(op.Value)),
					},
					IsDelete: false,
				})
			}
			cur += consumed
		}
		if !validOps {
			break
		}

		// Apply updates to index
		db.index.ApplyUpdates(updates)
		lastTxID = txnHeader.TxID
		txCount++
		offset += totalTxBytes
	}

	var repairedBytes uint64

	// If there is an incomplete or torn tail past the last valid committed transaction:
	if fileSize > offset {
		repairedBytes = uint64(fileSize - offset)
		if db.readOnly {
			// Cannot truncate in read-only mode; mark poisoned to prevent corrupted operation
			db.poisoned = fmt.Errorf("%w: uncommitted tail of %d bytes present in read-only mode", ErrCorrupt, repairedBytes)
		} else {
			// Safely truncate the file to the last committed boundary
			if err := TruncateFile(db.file, offset); err != nil {
				return OpenInfo{}, fmt.Errorf("failed to truncate torn tail at offset %d: %w", offset, err)
			}
			if err := SyncFile(db.file); err != nil {
				return OpenInfo{}, fmt.Errorf("failed to sync file after tail truncation: %w", err)
			}
		}
	}

	db.committedEnd = offset
	db.nextTxID = lastTxID + 1
	db.txCount = txCount

	return OpenInfo{
		Recreated:             false,
		RecoveredTransactions: txCount,
		LastTxID:              lastTxID,
		LiveKeyCount:          uint64(db.index.Len()),
		RepairedTailBytes:     repairedBytes,
	}, nil
}
