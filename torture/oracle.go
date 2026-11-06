package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"os"
	"sync"

	"github.com/madhuharshag/Team-Innovexa/src/kv"
)

// Oracle is an independent verifier that maintains the expected database state
// separately from the database implementation.
type Oracle struct {
	mu sync.Mutex

	journalPath string
	journalFile *os.File

	// Acknowledged state: only mutations that successfully returned to the caller
	ackedKeys    map[string][]byte
	ackedDeletes map[string]bool

	// History of operations for diagnosis
	history []OpRecord

	// In-flight mutations interrupted by a crash whose commit status was unknown
	indeterminateOps []OpRecord
}

// NewOracle constructs an independent verification oracle with durable journal persistence.
func NewOracle(journalPath string) (*Oracle, error) {
	var f *os.File
	var err error
	if journalPath != "" {
		f, err = os.OpenFile(journalPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open oracle journal: %w", err)
		}
	}

	return &Oracle{
		journalPath:  journalPath,
		journalFile:  f,
		ackedKeys:    make(map[string][]byte),
		ackedDeletes: make(map[string]bool),
		history:      make([]OpRecord, 0),
	}, nil
}

// Close closes the journal file.
func (o *Oracle) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.journalFile != nil {
		_ = o.journalFile.Sync()
		return o.journalFile.Close()
	}
	return nil
}

// RecordAck records an operation that was observed to succeed.
// It durably persists the acknowledgement before returning.
func (o *Oracle) RecordAck(op OpRecord) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	op.Status = StatusAcknowledged
	o.history = append(o.history, op)

	// Update expected state
	switch op.Kind {
	case OpPut:
		valCopy := make([]byte, len(op.Value))
		copy(valCopy, op.Value)
		o.ackedKeys[op.Key] = valCopy
		delete(o.ackedDeletes, op.Key)

	case OpDelete:
		delete(o.ackedKeys, op.Key)
		o.ackedDeletes[op.Key] = true

	case OpBatch:
		for _, bOp := range op.BatchOps {
			if bOp.Kind == OpPut {
				valCopy := make([]byte, len(bOp.Value))
				copy(valCopy, bOp.Value)
				o.ackedKeys[bOp.Key] = valCopy
				delete(o.ackedDeletes, bOp.Key)
			} else if bOp.Kind == OpDelete {
				delete(o.ackedKeys, bOp.Key)
				o.ackedDeletes[bOp.Key] = true
			}
		}
	}

	// Persist to journal file with Sync to guarantee oracle durability before crashes
	if o.journalFile != nil {
		data, err := json.Marshal(op)
		if err == nil {
			_, _ = o.journalFile.Write(append(data, '\n'))
			_ = o.journalFile.Sync()
		}
	}

	return nil
}

// RecordIndeterminate records an operation that was sent but interrupted before an ACK was received.
func (o *Oracle) RecordIndeterminate(op OpRecord) {
	o.mu.Lock()
	defer o.mu.Unlock()

	op.Status = StatusIndeterminate
	o.history = append(o.history, op)
	o.indeterminateOps = append(o.indeterminateOps, op)

	if o.journalFile != nil {
		data, err := json.Marshal(op)
		if err == nil {
			_, _ = o.journalFile.Write(append(data, '\n'))
			_ = o.journalFile.Sync()
		}
	}
}

// RecordRejected records an operation that failed or was rejected with an error.
func (o *Oracle) RecordRejected(op OpRecord, errMsg string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	op.Status = StatusRejected
	op.ErrorMsg = errMsg
	o.history = append(o.history, op)
}

// InjectSimulatedLoss modifies the oracle's expected state to simulate data loss.
// Used exclusively for testing the oracle's own ability to detect discrepancies.
func (o *Oracle) InjectSimulatedLoss(key string, inventedVal []byte) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.ackedKeys[key] = inventedVal
}

// Verify inspects the recovered database using the public kv.Engine interface
// and rigorously validates all invariants.
func (o *Oracle) Verify(dbPath string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	// 1. Open database through official public contract
	db, info, err := kv.Open(dbPath, kv.Options{})
	if err != nil {
		return fmt.Errorf("oracle verify: failed to open database %s: %w", dbPath, err)
	}
	defer db.Close()

	_ = info

	// 2. Run integrity check
	if err := db.Check(); err != nil {
		return fmt.Errorf("oracle verify: database Check() integrity failed: %w", err)
	}

	// 3. Verify every acknowledged Put survives with exact byte content
	for k, expectedVal := range o.ackedKeys {
		actualVal, err := db.Get([]byte(k))
		if err != nil {
			if errors.Is(err, kv.ErrNotFound) {
				return fmt.Errorf("oracle verify: acknowledged key %q was lost after recovery", k)
			}
			return fmt.Errorf("oracle verify: Get for acknowledged key %q failed: %w", k, err)
		}
		if !bytes.Equal(actualVal, expectedVal) {
			return fmt.Errorf("oracle verify: acknowledged key %q value mismatch: got CRC %08x (len %d), want CRC %08x (len %d)",
				k, crc32.ChecksumIEEE(actualVal), len(actualVal),
				crc32.ChecksumIEEE(expectedVal), len(expectedVal))
		}
	}

	// 4. Verify every acknowledged Delete remains deleted (ErrNotFound)
	for k := range o.ackedDeletes {
		// Only check if key was not subsequently re-Put
		if _, reAdded := o.ackedKeys[k]; !reAdded {
			val, err := db.Get([]byte(k))
			if err == nil {
				return fmt.Errorf("oracle verify: acknowledged deleted key %q is present with value len %d", k, len(val))
			}
			if !errors.Is(err, kv.ErrNotFound) {
				return fmt.Errorf("oracle verify: unexpected error checking deleted key %q: %w", k, err)
			}
		}
	}

	// 5. Verify batch atomicity for indeterminate batches (All-or-Nothing rule)
	for _, ind := range o.indeterminateOps {
		if ind.Kind == OpBatch && len(ind.BatchOps) > 0 {
			appliedCount := 0
			for _, bOp := range ind.BatchOps {
				val, err := db.Get([]byte(bOp.Key))
				if bOp.Kind == OpPut {
					if err == nil && bytes.Equal(val, bOp.Value) {
						appliedCount++
					}
				} else if bOp.Kind == OpDelete {
					if errors.Is(err, kv.ErrNotFound) {
						appliedCount++
					}
				}
			}
			// Invariant: batch must either be 100% applied or 0% applied.
			// Any partial count is a violation of atomicity!
			if appliedCount > 0 && appliedCount < len(ind.BatchOps) {
				return fmt.Errorf("oracle verify: atomic batch %d violated: %d of %d ops applied (partial batch exposed!)",
					ind.BatchID, appliedCount, len(ind.BatchOps))
			}
		}
	}

	// 6. Scan all keys in database to verify no invented/phantom keys exist
	discoveredCount := 0
	err = db.Scan(func(k, v []byte) error {
		discoveredCount++
		keyStr := string(k)
		// Key must be in ackedKeys OR belong to an in-flight indeterminate Put
		if _, exists := o.ackedKeys[keyStr]; exists {
			return nil
		}

		// Check if it was from an indeterminate op
		for _, ind := range o.indeterminateOps {
			if ind.Kind == OpPut && ind.Key == keyStr && bytes.Equal(v, ind.Value) {
				return nil
			}
			if ind.Kind == OpBatch {
				for _, bOp := range ind.BatchOps {
					if bOp.Kind == OpPut && bOp.Key == keyStr && bytes.Equal(v, bOp.Value) {
						return nil
					}
				}
			}
		}

		return fmt.Errorf("oracle verify: invented/corrupted key %q found in database (not acknowledged or submitted)", keyStr)
	})
	if err != nil {
		return err
	}

	return nil
}

// AckedCount returns the number of live keys acknowledged in the oracle.
func (o *Oracle) AckedCount() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return len(o.ackedKeys)
}
