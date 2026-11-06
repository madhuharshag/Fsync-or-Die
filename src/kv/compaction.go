package kv

import (
	"fmt"
	"hash/crc32"
	"math"
	"os"
)

// Default limits for compaction transaction chunking.
const (
	// DefaultCompactionBatchOps is the maximum number of operations bundled into a single
	// compaction transaction.
	DefaultCompactionBatchOps = 1000

	// DefaultCompactionBatchBytes is the maximum encoded payload byte size bundled into
	// a single compaction transaction (16 MiB). This guarantees bounded RAM usage even with
	// multiple large values, preventing 32-bit OOM errors.
	DefaultCompactionBatchBytes uint64 = 16 * 1024 * 1024
)

// CompactionHookFunc defines a callback invoked during specific phases of compaction.
type CompactionHookFunc func(stage string) error

// Active compaction batch thresholds and test hooks (overridable in tests).
var (
	compactionBatchOps   = DefaultCompactionBatchOps
	compactionBatchBytes = DefaultCompactionBatchBytes
	compactionHook       CompactionHookFunc
	compactionRenameFn   = os.Rename
)

// compactLocked executes crash-safe compaction with db.mu held.
// It iterates all live entries in the index, writes them to a temporary file (path + ".compact.tmp"),
// ensures durability via SyncFile, replaces the authoritative file atomically, and switches
// the descriptor and in-memory index.
func (db *DB) compactLocked() error {
	if err := db.checkStateLocked(); err != nil {
		return err
	}
	if db.readOnly {
		return fmt.Errorf("%w: cannot compact in read-only mode", ErrInvalid)
	}

	tmpPath := db.path + ".compact.tmp"

	// 1. Create temporary replacement file in the same directory (same filesystem for atomic rename)
	tmpFile, err := os.OpenFile(tmpPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("compaction: failed to create temporary file %s: %w", tmpPath, err)
	}
	defer func() {
		if tmpFile != nil {
			_ = tmpFile.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	// 2. Write immutable file header (32 bytes at offset 0)
	fileHeader := NewFileHeader(db.maxFileBytes)
	fileHeaderBytes := EncodeFileHeader(fileHeader)
	if err := WriteAtFull(tmpFile, fileHeaderBytes, 0); err != nil {
		return fmt.Errorf("compaction: failed to write file header: %w", err)
	}

	newIndex := NewIndex()
	var newCommittedEnd int64 = FileHeaderSize
	var nextTxID uint64 = 1

	// 3. Snapshot live keys from the current index in deterministic sorted order
	liveKeys := db.index.Keys()

	if compactionHook != nil {
		if err := compactionHook("writing_tmp"); err != nil {
			return err
		}
	}

	// If there are live keys, stream them into the new file in bounded transactions
	// constrained by both operation count and cumulative payload byte size.
	keyIdx := 0
	for keyIdx < len(liveKeys) {
		var batchOps []Op
		var batchPayloadBytes uint64

		for keyIdx < len(liveKeys) {
			k := liveKeys[keyIdx]
			entry, found := db.index.Get(k)
			if !found {
				keyIdx++
				continue
			}

			// Checked calculation of op size: 7-byte header + key len + val len
			kLen := uint64(len(k))
			vLen := uint64(entry.ValueLength)
			opBytes := uint64(OpHeaderSize) + kLen + vLen

			// Check whether an individual operation is too large to fit in any compaction batch
			if opBytes > compactionBatchBytes {
				return fmt.Errorf("compaction: operation for key %q (encoded size %d bytes) exceeds compaction batch limit of %d bytes: %w", k, opBytes, compactionBatchBytes, ErrInvalid)
			}

			// If the batch already has at least one operation, check whether adding this
			// operation would exceed either the op-count limit or the byte-size limit.
			// Checking BEFORE reading/allocating ensures large values are bounded.
			if len(batchOps) > 0 {
				if len(batchOps) >= compactionBatchOps {
					break
				}
				if math.MaxUint64-batchPayloadBytes < opBytes || batchPayloadBytes+opBytes > compactionBatchBytes {
					break
				}
			}

			// Read live value from the authoritative file
			var val []byte
			if entry.ValueLength > 0 {
				val = make([]byte, entry.ValueLength)
				if err := ReadAtFull(db.file, val, entry.ValueOffset); err != nil {
					return fmt.Errorf("compaction: failed to read live value for key %q at offset %d: %w", k, entry.ValueOffset, err)
				}
			}

			batchOps = append(batchOps, Op{
				Kind:  PutOp,
				Key:   k,
				Value: val,
			})
			batchPayloadBytes += opBytes
			keyIdx++
		}

		if len(batchOps) == 0 {
			continue
		}

		// Calculate sizes with checked arithmetic
		payloadLen, err := CalculatePayloadSize(batchOps)
		if err != nil {
			return fmt.Errorf("compaction: payload size calculation failed: %w", err)
		}
		totalTxBytes, err := CalculateTotalTxnSize(batchOps)
		if err != nil {
			return fmt.Errorf("compaction: total transaction size calculation failed: %w", err)
		}

		// Preflight capacity check for the replacement file (Scenario E4)
		if err := CheckCapacity(newCommittedEnd, totalTxBytes, db.maxFileBytes); err != nil {
			return fmt.Errorf("compaction: %w", err)
		}

		// Build transaction header
		txnHeader := TxnHeader{
			Magic:      TxnHeaderMagic,
			Version:    CurrentTxnVersion,
			Flags:      0,
			TxID:       nextTxID,
			OpCount:    uint32(len(batchOps)),
			Reserved:   0,
			PayloadLen: payloadLen,
		}
		headerBytes := EncodeTxnHeader(txnHeader)

		crcHasher := crc32.NewIEEE()
		crcHasher.Write(headerBytes)

		encodedOps := make([][]byte, len(batchOps))
		curPayloadRel := int64(TxnHeaderSize)
		batchUpdates := make([]PendingIndexUpdate, len(batchOps))

		for j, op := range batchOps {
			encOp, opErr := EncodeOp(op)
			if opErr != nil {
				return fmt.Errorf("compaction: encode op %d failed: %w", j, opErr)
			}
			encodedOps[j] = encOp
			crcHasher.Write(encOp)

			valOffset := newCommittedEnd + curPayloadRel + int64(OpHeaderSize) + int64(len(op.Key))
			batchUpdates[j] = PendingIndexUpdate{
				Key: op.Key,
				Entry: IndexEntry{
					ValueOffset: valOffset,
					ValueLength: uint32(len(op.Value)),
				},
				IsDelete: false,
			}
			curPayloadRel += int64(len(encOp))
		}

		txnCRC := crcHasher.Sum32()
		footer := NewCommitFooter(nextTxID, totalTxBytes, txnCRC)
		footerBytes := EncodeCommitFooter(footer)

		// Write Header to tmpFile
		if err := WriteAtFull(tmpFile, headerBytes, newCommittedEnd); err != nil {
			return fmt.Errorf("compaction: failed to write txn header: %w", err)
		}

		// Write Operations to tmpFile
		writeOffset := newCommittedEnd + int64(TxnHeaderSize)
		for _, encOp := range encodedOps {
			if err := WriteAtFull(tmpFile, encOp, writeOffset); err != nil {
				return fmt.Errorf("compaction: failed to write op payload: %w", err)
			}
			writeOffset += int64(len(encOp))
		}

		// Write Footer to tmpFile
		footerOffset := newCommittedEnd + int64(TxnHeaderSize) + int64(payloadLen)
		if err := WriteAtFull(tmpFile, footerBytes, footerOffset); err != nil {
			return fmt.Errorf("compaction: failed to write txn footer: %w", err)
		}

		// Record updates into replacement index
		newIndex.ApplyUpdates(batchUpdates)

		newCommittedEnd += int64(totalTxBytes)
		nextTxID++
	}

	// 4. Ensure complete physical durability of the replacement file before swapping
	if err := SyncFile(tmpFile); err != nil {
		return fmt.Errorf("compaction: failed to sync temporary file: %w", err)
	}

	// 5. Atomic descriptor & file transition
	// On Windows, open handles must be closed before os.Rename can replace the destination.
	// On POSIX, closing before rename is also safe when protected by db.mu and the process lock file.
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("compaction: failed to close temporary file before swap: %w", err)
	}
	tmpFile = nil // Prevent deferred cleanup from removing tmpFile after successful close

	if err := db.file.Close(); err != nil {
		return fmt.Errorf("compaction: failed to close authoritative file before swap: %w", err)
	}

	if compactionHook != nil {
		if err := compactionHook("after_sync"); err != nil {
			reopenFile, reopenErr := os.OpenFile(db.path, os.O_RDWR, 0644)
			if reopenErr == nil {
				db.file = reopenFile
			} else {
				db.poisoned = fmt.Errorf("compaction failed to restore file: %w", reopenErr)
			}
			_ = os.Remove(tmpPath)
			return err
		}
	}

	// Atomic rename: replacement file becomes the authoritative database
	if err := compactionRenameFn(tmpPath, db.path); err != nil {
		// If rename fails, attempt reopening the old file
		reopenFile, reopenErr := os.OpenFile(db.path, os.O_RDWR, 0644)
		if reopenErr == nil {
			db.file = reopenFile
		} else {
			db.poisoned = fmt.Errorf("compaction failed to restore file after rename error: %w", reopenErr)
		}
		_ = os.Remove(tmpPath)
		return fmt.Errorf("compaction: atomic rename failed: %w", err)
	}

	if compactionHook != nil {
		if err := compactionHook("after_rename"); err != nil {
			return err
		}
	}

	// 6. Reopen the newly installed authoritative file
	newFile, err := os.OpenFile(db.path, os.O_RDWR, 0644)
	if err != nil {
		db.poisoned = fmt.Errorf("compaction failed to reopen database after swap: %w", err)
		return fmt.Errorf("compaction: reopen failed: %w", err)
	}

	// 7. Publish replacement state under db.mu
	db.file = newFile
	db.index.ReplaceWith(newIndex)
	db.committedEnd = newCommittedEnd
	db.nextTxID = nextTxID

	return nil
}
