package kv

import (
	"fmt"
	"hash/crc32"
)

// Put stores or overwrites a key-value pair.
// A nil return guarantees that the mutation is durably synchronized to storage.
func (db *DB) Put(key, value []byte) error {
	op := Op{
		Kind:  PutOp,
		Key:   key,
		Value: value,
	}
	return db.Batch([]Op{op})
}

// Delete durably records the removal of key.
// Deleting an absent or nonexistent key succeeds with nil error.
func (db *DB) Delete(key []byte) error {
	op := Op{
		Kind:  DeleteOp,
		Key:   key,
		Value: nil,
	}
	return db.Batch([]Op{op})
}

// Batch atomically applies a sequence of Put and Delete operations in list order.
// All operations share a single transaction header, payload, and commit footer.
func (db *DB) Batch(ops []Op) error {
	if err := ValidateOps(ops); err != nil {
		return err
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	return db.commitBatchLocked(ops)
}

// commitBatchLocked executes the transaction commit path with db.mu held.
func (db *DB) commitBatchLocked(ops []Op) error {
	if err := db.checkStateLocked(); err != nil {
		return err
	}
	if db.readOnly {
		return fmt.Errorf("%w: cannot execute mutation in read-only mode", ErrInvalid)
	}

	// 1. Calculate checked encoded sizes
	payloadLen, err := CalculatePayloadSize(ops)
	if err != nil {
		return err
	}
	totalTxBytes, err := CalculateTotalTxnSize(ops)
	if err != nil {
		return err
	}

	// 2. Preflight capacity check
	if err := CheckCapacity(db.committedEnd, totalTxBytes, db.maxFileBytes); err != nil {
		return err
	}

	// 3. Prepare headers and incremental CRC
	txID := db.nextTxID
	startOffset := db.committedEnd

	txnHeader := TxnHeader{
		Magic:      TxnHeaderMagic,
		Version:    CurrentTxnVersion,
		Flags:      0,
		TxID:       txID,
		OpCount:    uint32(len(ops)),
		Reserved:   0,
		PayloadLen: payloadLen,
	}
	headerBytes := EncodeTxnHeader(txnHeader)

	crcHasher := crc32.NewIEEE()
	crcHasher.Write(headerBytes)

	// Pre-encode ops and compute CRC incrementally without giant duplicate buffers
	pendingUpdates := make([]PendingIndexUpdate, 0, len(ops))
	encodedOps := make([][]byte, len(ops))
	curPayloadRel := int64(TxnHeaderSize)

	for i, op := range ops {
		encOp, opErr := EncodeOp(op)
		if opErr != nil {
			return opErr
		}
		encodedOps[i] = encOp
		crcHasher.Write(encOp)

		valOffset := startOffset + curPayloadRel + int64(OpHeaderSize) + int64(len(op.Key))
		if op.Kind == DeleteOp {
			pendingUpdates = append(pendingUpdates, PendingIndexUpdate{
				Key:      op.Key,
				IsDelete: true,
			})
		} else {
			pendingUpdates = append(pendingUpdates, PendingIndexUpdate{
				Key: op.Key,
				Entry: IndexEntry{
					ValueOffset: valOffset,
					ValueLength: uint32(len(op.Value)),
				},
				IsDelete: false,
			})
		}
		curPayloadRel += int64(len(encOp))
	}

	txnCRC := crcHasher.Sum32()

	// Prepare CommitFooter
	footer := NewCommitFooter(txID, totalTxBytes, txnCRC)
	footerBytes := EncodeCommitFooter(footer)

	// 4. Sequential positioned write to disk
	// Write Header
	if err := WriteAtFull(db.file, headerBytes, startOffset); err != nil {
		_ = TruncateFile(db.file, startOffset)
		return fmt.Errorf("transaction header write failed: %w", err)
	}

	// Write Operations
	writeOffset := startOffset + int64(TxnHeaderSize)
	for _, encOp := range encodedOps {
		if err := WriteAtFull(db.file, encOp, writeOffset); err != nil {
			_ = TruncateFile(db.file, startOffset)
			return fmt.Errorf("transaction payload write failed: %w", err)
		}
		writeOffset += int64(len(encOp))
	}

	// Write Footer
	footerOffset := startOffset + int64(TxnHeaderSize) + int64(payloadLen)
	if err := WriteAtFull(db.file, footerBytes, footerOffset); err != nil {
		_ = TruncateFile(db.file, startOffset)
		return fmt.Errorf("transaction footer write failed: %w", err)
	}

	// 5. Final Sync to guarantee durability (unconditional ACID guarantee)
	if db.syncSeam != nil {
		if err := db.syncSeam(db.file); err != nil {
			db.poisoned = err
			return fmt.Errorf("%w: %v", ErrUncertain, err)
		}
	} else {
		if err := SyncFile(db.file); err != nil {
			// Real sync failure after footer write leaves durability in an uncertain state
			db.poisoned = err
			return fmt.Errorf("%w: %v", ErrUncertain, err)
		}
	}

	// 6. Publish index changes in operation order
	db.index.ApplyUpdates(pendingUpdates)

	// 7. Advance committed end and transaction ID
	db.committedEnd += int64(totalTxBytes)
	db.nextTxID++

	return nil
}
