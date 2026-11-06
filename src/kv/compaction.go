package kv

import (
	"fmt"
	"hash/crc32"
	"os"
)

// DefaultCompactionBatchOps is the maximum number of operations bundled into a single
// transaction during compaction. Keeping this bounded prevents massive memory consumption.
const DefaultCompactionBatchOps = 1000

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

	// If there are live keys, stream them into the new file in bounded transactions
	if len(liveKeys) > 0 {
		for i := 0; i < len(liveKeys); i += DefaultCompactionBatchOps {
			end := i + DefaultCompactionBatchOps
			if end > len(liveKeys) {
				end = len(liveKeys)
			}
			batchKeys := liveKeys[i:end]

			// Read live values from the current file and construct Ops
			ops := make([]Op, len(batchKeys))
			for j, k := range batchKeys {
				entry, found := db.index.Get(k)
				if !found {
					continue
				}
				var val []byte
				if entry.ValueLength > 0 {
					val = make([]byte, entry.ValueLength)
					if err := ReadAtFull(db.file, val, entry.ValueOffset); err != nil {
						return fmt.Errorf("compaction: failed to read live value for key %q at offset %d: %w", k, entry.ValueOffset, err)
					}
				}
				ops[j] = Op{
					Kind:  PutOp,
					Key:   k,
					Value: val,
				}
			}

			// Calculate sizes with checked arithmetic
			payloadLen, err := CalculatePayloadSize(ops)
			if err != nil {
				return fmt.Errorf("compaction: payload size calculation failed: %w", err)
			}
			totalTxBytes, err := CalculateTotalTxnSize(ops)
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
				OpCount:    uint32(len(ops)),
				Reserved:   0,
				PayloadLen: payloadLen,
			}
			headerBytes := EncodeTxnHeader(txnHeader)

			crcHasher := crc32.NewIEEE()
			crcHasher.Write(headerBytes)

			encodedOps := make([][]byte, len(ops))
			curPayloadRel := int64(TxnHeaderSize)
			batchUpdates := make([]PendingIndexUpdate, len(ops))

			for j, op := range ops {
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

	// Atomic rename: replacement file becomes the authoritative database
	if err := os.Rename(tmpPath, db.path); err != nil {
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
