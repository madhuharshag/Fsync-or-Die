// Package kv implements the Fsync-or-Die core storage engine for Team Innovexa.
// Design invariants:
// - Single file append-only write-ahead log.
// - All mutations are committed via dual-checksummed transaction frames.
// - Exclusive advisory process lock on <path>.lock.
// - RAM index stores 64-bit file offsets and 32-bit lengths; values stay on disk.
// - fsync/fdatasync before acknowledging success.
package kv

import (
	"bytes"
	"errors"
	"fmt"
	"hash/crc32"
	"os"
	"sync"
	"time"
)

// DB represents an open Fsync-or-Die storage engine instance.
type DB struct {
	path     string   // Authoritative data file path
	lockPath string   // Advisory process lock path (<path>.lock)
	file     *os.File // Open file descriptor for data file
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

	mu sync.RWMutex // Protects all internal state and serializes mutations
}

// Open opens or creates an authoritative database file at path.
// It acquires an exclusive advisory lock, validates or initializes the file header,
// and recovers committed transactions, truncating any incomplete tail.
func Open(path string, opts Options) (Engine, OpenInfo, error) {
	startTime := time.Now()

	// 1. Acquire non-blocking exclusive advisory lock on <path>.lock
	lockPath := path + ".lock"
	lockFile, err := acquireProcessLock(lockPath)
	if err != nil {
		return nil, OpenInfo{}, err
	}

	// Clean up any stray temporary file from an interrupted compaction (Scenario C5)
	if !opts.ReadOnly {
		_ = os.Remove(path + ".compact.tmp")
	}

	// 3. Open authoritative data file (NO O_TRUNC, NO O_APPEND)
	openFlags := os.O_RDWR | os.O_CREATE
	if opts.ReadOnly {
		openFlags = os.O_RDONLY
	}
	f, err := os.OpenFile(path, openFlags, 0644)
	if err != nil {
		_ = releaseProcessLock(lockFile)
		return nil, OpenInfo{}, fmt.Errorf("failed to open database file: %w", err)
	}

	db := &DB{
		path:         path,
		lockPath:     lockPath,
		file:         f,
		lockFile:     lockFile,
		index:        NewIndex(),
		maxFileBytes: opts.MaxFileBytes,
		readOnly:     opts.ReadOnly,
	}

	// 4. Inspect file size
	fi, err := f.Stat()
	if err != nil {
		_ = db.Close()
		return nil, OpenInfo{}, fmt.Errorf("failed to stat database file: %w", err)
	}

	var openInfo OpenInfo

	if fi.Size() == 0 {
		if opts.ReadOnly {
			_ = db.Close()
			return nil, OpenInfo{}, fmt.Errorf("%w: cannot initialize new database in read-only mode", ErrCorrupt)
		}
		// Fresh file: write immutable 32-byte file header
		hdrBytes := EncodeFileHeader(NewFileHeader())
		if err := WriteAtFull(db.file, hdrBytes, 0); err != nil {
			_ = db.Close()
			return nil, OpenInfo{}, fmt.Errorf("failed to write initial file header: %w", err)
		}
		if err := SyncFile(db.file); err != nil {
			_ = db.Close()
			return nil, OpenInfo{}, fmt.Errorf("failed to sync initial file header: %w", err)
		}
		db.committedEnd = int64(FileHeaderSize)
		db.nextTxID = 1
		openInfo.Recreated = true
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
			return nil, OpenInfo{}, fmt.Errorf("%w: %v", ErrCorrupt, err)
		}
		if !bytes.Equal(header.Magic[:], []byte(FileMagic)) {
			_ = db.Close()
			return nil, OpenInfo{}, fmt.Errorf("%w: invalid file magic %q", ErrCorrupt, string(header.Magic[:]))
		}
		if header.Version != FileFormatVersion {
			_ = db.Close()
			return nil, OpenInfo{}, fmt.Errorf("%w: unsupported file version %d", ErrCorrupt, header.Version)
		}

		// Sequential scan and recovery
		info, err := db.recoverAndRepairLocked()
		if err != nil {
			_ = db.Close()
			return nil, OpenInfo{}, err
		}
		openInfo = info
	}

	openInfo.RecoveryDurationNs = time.Since(startTime).Nanoseconds()
	return db, openInfo, nil
}

// Close gracefully closes the database. It flushes unwritten data if any, closes
// the underlying file descriptor, releases the advisory process lock, and frees memory.
func (db *DB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return ErrClosed
	}
	db.closed = true

	var firstErr error
	if db.file != nil {
		if !db.readOnly && db.poisoned == nil {
			if err := SyncFile(db.file); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		if err := db.file.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		db.file = nil
	}

	if db.lockFile != nil {
		if err := releaseProcessLock(db.lockFile); err != nil && firstErr == nil {
			firstErr = err
		}
		db.lockFile = nil
	}

	return firstErr
}

// checkStateLocked verifies that the database is open, writable (if required),
// and not poisoned by an earlier fatal I/O failure. Must be called under db.mu.
func (db *DB) checkStateLocked() error {
	if db.closed {
		return ErrClosed
	}
	if db.poisoned != nil {
		return fmt.Errorf("%w: %w", ErrPoisoned, db.poisoned)
	}
	return nil
}

// Stats returns current database telemetry snapshot.
func (db *DB) Stats() Stats {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var size uint64
	if db.file != nil {
		if fi, err := db.file.Stat(); err == nil {
			size = uint64(fi.Size())
		}
	}

	return Stats{
		LiveKeyCount:    db.index.Count(),
		FileSizeBytes:   size,
		MaxFileBytes:    db.maxFileBytes,
		LastTxID:        db.nextTxID - 1,
		TxCount:         db.txCount,
		CompactionCount: db.compactionCount,
	}
}

// Check performs a full structural integrity scan of the authoritative file.
// It verifies FileHeader, all TxnHeaders, payload bounds, OpCount consistency,
// and dual CRC32 checksums for every transaction without altering state.
func (db *DB) Check() error {
	db.mu.RLock()
	defer db.mu.RUnlock()

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

		remaining := uint64(db.committedEnd - offset)
		overhead := uint64(TxnHeaderSize + CommitFooterSize)
		if remaining < overhead || txnHeader.PayloadLen > remaining-overhead {
			return fmt.Errorf("%w: check: transaction at %d exceeds committed end %d (payload %d bytes)", ErrCorrupt, offset, db.committedEnd, txnHeader.PayloadLen)
		}

		totalTxBytes := int64(overhead + txnHeader.PayloadLen)
		payloadBuf := make([]byte, txnHeader.PayloadLen)
		if err := ReadAtFull(db.file, payloadBuf, offset+int64(TxnHeaderSize)); err != nil {
			return fmt.Errorf("check: read payload at %d: %w", offset, err)
		}

		// Verify individual operations in payload
		opCount := uint32(0)
		cur := 0
		for cur < len(payloadBuf) {
			_, consumed, opErr := DecodeOp(payloadBuf[cur:])
			if opErr != nil {
				return fmt.Errorf("check: decode op in txn %d at offset %d: %w", txnHeader.TxID, offset+int64(TxnHeaderSize)+int64(cur), opErr)
			}
			opCount++
			cur += consumed
		}
		if opCount != txnHeader.OpCount {
			return fmt.Errorf("%w: check: txn %d decoded op count %d mismatch declared %d", ErrCorrupt, txnHeader.TxID, opCount, txnHeader.OpCount)
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

// Scan iterates over all live key-value pairs in lexicographical key order.
func (db *DB) Scan(fn func(key, value []byte) error) error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if err := db.checkStateLocked(); err != nil {
		return err
	}

	snap := db.index.Snapshot()
	for _, entry := range snap {
		valBuf := make([]byte, entry.Length)
		if err := ReadAtFull(db.file, valBuf, int64(entry.Offset)); err != nil {
			return fmt.Errorf("scan read error for key %s: %w", string(entry.Key), err)
		}
		if err := fn(entry.Key, valBuf); err != nil {
			if errors.Is(err, ErrIterationStop) {
				return nil
			}
			return err
		}
	}
	return nil
}

// Compact executes a crash-safe compaction of live data into a new log file,
// reclaiming space occupied by tombstones and overwritten keys.
func (db *DB) Compact() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	if err := db.compactLocked(); err != nil {
		return err
	}
	db.compactionCount++
	return nil
}

// recoverAndRepairLocked performs sequential recovery of committed transactions
// starting immediately after the 32-byte FileHeader. If an incomplete transaction
// is encountered at EOF, it truncates the torn write (if not read-only).
func (db *DB) recoverAndRepairLocked() (OpenInfo, error) {
	fi, err := db.file.Stat()
	if err != nil {
		return OpenInfo{}, fmt.Errorf("stat failed during recovery: %w", err)
	}
	fileSize := fi.Size()

	offset := int64(FileHeaderSize)
	var txCount uint64
	var lastTxID uint64

	for offset < fileSize {
		// Read TxnHeader
		if offset+int64(TxnHeaderSize) > fileSize {
			// Incomplete transaction header at EOF
			break
		}
		txnHeaderBuf := make([]byte, TxnHeaderSize)
		if err := ReadAtFull(db.file, txnHeaderBuf, offset); err != nil {
			break
		}
		txnHeader, err := DecodeTxnHeader(txnHeaderBuf)
		if err != nil {
			break
		}
		if !bytes.Equal(txnHeader.Magic[:], []byte(TxnMagic)) {
			break
		}

		remaining := uint64(fileSize - offset)
		overhead := uint64(TxnHeaderSize + CommitFooterSize)
		if remaining < overhead || txnHeader.PayloadLen > remaining-overhead {
			// Incomplete transaction payload/footer
			break
		}
		totalTxBytes := int64(overhead + txnHeader.PayloadLen)

		// Read payload
		payloadBuf := make([]byte, txnHeader.PayloadLen)
		if err := ReadAtFull(db.file, payloadBuf, offset+int64(TxnHeaderSize)); err != nil {
			break
		}

		// Read CommitFooter
		footerBuf := make([]byte, CommitFooterSize)
		footerOffset := offset + int64(TxnHeaderSize) + int64(txnHeader.PayloadLen)
		if err := ReadAtFull(db.file, footerBuf, footerOffset); err != nil {
			break
		}
		footer, err := DecodeCommitFooter(footerBuf)
		if err != nil {
			break
		}
		if !bytes.Equal(footer.Magic[:], []byte(FooterMagic)) {
			break
		}
		if footer.TxID != txnHeader.TxID {
			break
		}
		if footer.TotalTxBytes != uint64(totalTxBytes) {
			break
		}

		// Check dual CRCs
		h := crc32.NewIEEE()
		h.Write(txnHeaderBuf)
		h.Write(payloadBuf)
		if h.Sum32() != footer.TxnCRC {
			// Corrupt transaction
			break
		}
		if crc32.ChecksumIEEE(footerBuf[0:28]) != footer.FooterCRC {
			// Corrupt footer
			break
		}

		// Transaction is valid. Replay operations into RAM index.
		cur := 0
		var updates []IndexUpdate
		validOps := true
		for cur < len(payloadBuf) {
			op, consumed, opErr := DecodeOp(payloadBuf[cur:])
			if opErr != nil {
				validOps = false
				break
			}
			valOffsetInPayload := cur + 7 + len(op.Key)
			absoluteValOffset := uint64(offset + int64(TxnHeaderSize) + int64(valOffsetInPayload))

			if op.Kind == PutOp {
				updates = append(updates, IndexUpdate{
					Key:    op.Key,
					Offset: absoluteValOffset,
					Length: uint32(len(op.Value)),
				})
			} else if op.Kind == DeleteOp {
				updates = append(updates, IndexUpdate{
					Key:      op.Key,
					IsDelete: true,
				})
			}
			cur += consumed
		}
		if !validOps || uint32(len(updates)) != txnHeader.OpCount {
			break
		}

		db.index.ApplyUpdates(updates)
		lastTxID = txnHeader.TxID
		txCount++
		offset += totalTxBytes
	}

	// Tail repair: if uncommitted or torn bytes exist past the last valid transaction
	var repairedBytes int64
	if offset < fileSize {
		if db.readOnly {
			// In read-only mode, cannot truncate. Return ErrCorrupt if uncommitted bytes exist
			return OpenInfo{}, fmt.Errorf("%w: read-only database contains incomplete or corrupt tail at offset %d", ErrCorrupt, offset)
		}
		repairedBytes = fileSize - offset
		if err := TruncateFile(db.file, offset); err != nil {
			return OpenInfo{}, fmt.Errorf("failed to truncate corrupt tail at offset %d: %w", offset, err)
		}
		if err := SyncFile(db.file); err != nil {
			return OpenInfo{}, fmt.Errorf("failed to sync truncated database: %w", err)
		}
	}

	db.committedEnd = offset
	db.nextTxID = lastTxID + 1
	db.txCount = txCount

	return OpenInfo{
		Recreated:             false,
		LiveKeyCount:          db.index.Count(),
		RecoveredTransactions: txCount,
		RepairedBytes:         repairedBytes,
	}, nil
}
