package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"os"
	"sync"

	"github.com/madhuharshag/Fsync-or-Die/src/kv"
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

	// 3. Resolve any indeterminate atomic batches first (All-or-Nothing rule)
	for _, ind := range o.indeterminateOps {
		if ind.Kind == OpBatch && len(ind.BatchOps) > 0 {
			// Determine effective post-batch state for each distinct key modified in this batch
			// (handling duplicate keys within the batch sequentially: last write wins)
			effectiveBatchOps := make(map[string]OpRecord)
			for _, bOp := range ind.BatchOps {
				effectiveBatchOps[bOp.Key] = bOp
			}

			// Identify informative keys where the batch would change the pre-batch state
			type keyDelta struct {
				postOp    OpRecord
				hadPreVal bool
				preVal    []byte
			}
			deltas := make(map[string]keyDelta)

			for k, postOp := range effectiveBatchOps {
				preVal, hadPreVal := o.ackedKeys[k]
				preDeleted := o.ackedDeletes[k]

				isNoop := false
				if postOp.Kind == OpPut {
					if hadPreVal && bytes.Equal(preVal, postOp.Value) {
						isNoop = true // Pre-existing value matches new value
					}
				} else if postOp.Kind == OpDelete {
					if !hadPreVal && preDeleted {
						isNoop = true // Already deleted or absent
					}
				}

				if !isNoop {
					deltas[k] = keyDelta{
						postOp:    postOp,
						hadPreVal: hadPreVal,
						preVal:    preVal,
					}
				}
			}

			// If no keys would change state, adopt the batch's sequential state updates
			if len(deltas) == 0 {
				for _, bOp := range ind.BatchOps {
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
				continue
			}

			// Check disk state for all delta keys
			matchedPost := 0
			matchedPre := 0

			for k, delta := range deltas {
				actualVal, err := db.Get([]byte(k))
				postMatches := false
				if delta.postOp.Kind == OpPut {
					if err == nil && bytes.Equal(actualVal, delta.postOp.Value) {
						postMatches = true
					}
				} else if delta.postOp.Kind == OpDelete {
					if errors.Is(err, kv.ErrNotFound) {
						postMatches = true
					}
				}

				preMatches := false
				if delta.hadPreVal {
					if err == nil && bytes.Equal(actualVal, delta.preVal) {
						preMatches = true
					}
				} else {
					if errors.Is(err, kv.ErrNotFound) {
						preMatches = true
					}
				}

				if postMatches {
					matchedPost++
				} else if preMatches {
					matchedPre++
				} else {
					return fmt.Errorf("oracle verify: atomic batch %d key %q has unexpected value (len %d, err %v)",
						ind.BatchID, k, len(actualVal), err)
				}
			}

			// Invariant verification: All-or-Nothing
			if matchedPost > 0 && matchedPost < len(deltas) {
				return fmt.Errorf("oracle verify: atomic batch %d violated: %d of %d changing ops applied (partial batch exposed!)",
					ind.BatchID, matchedPost, len(deltas))
			}

			if matchedPost == len(deltas) {
				// The in-flight batch committed before process death. Adopt its state in sequential order.
				for _, bOp := range ind.BatchOps {
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
			// If matchedPre == len(deltas), the batch committed 0%; do nothing, pre-state remains intact!
		}
	}

	// 4. Resolve indeterminate single Puts and Deletes
	for _, ind := range o.indeterminateOps {
		if ind.Kind == OpPut {
			actualVal, err := db.Get([]byte(ind.Key))
			if err == nil && bytes.Equal(actualVal, ind.Value) {
				// The in-flight Put committed before process death. Adopt into ackedKeys.
				valCopy := make([]byte, len(ind.Value))
				copy(valCopy, ind.Value)
				o.ackedKeys[ind.Key] = valCopy
				delete(o.ackedDeletes, ind.Key)
			}
		} else if ind.Kind == OpDelete {
			_, err := db.Get([]byte(ind.Key))
			if errors.Is(err, kv.ErrNotFound) {
				// The in-flight Delete committed. Adopt into ackedDeletes.
				delete(o.ackedKeys, ind.Key)
				o.ackedDeletes[ind.Key] = true
			}
		}
	}

	// 5. Verify every acknowledged Put survives with exact byte content
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

	// 6. Verify every acknowledged Delete remains deleted (ErrNotFound)
	for k := range o.ackedDeletes {
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

	// 7. Scan all keys in database to verify no invented/phantom keys exist
	err = db.Scan(func(k, v []byte) error {
		keyStr := string(k)
		if expectedVal, exists := o.ackedKeys[keyStr]; exists {
			if !bytes.Equal(v, expectedVal) {
				return fmt.Errorf("oracle verify: scanned key %q value mismatch", keyStr)
			}
			return nil
		}
		return fmt.Errorf("oracle verify: invented/corrupted key %q found in database (not acknowledged or submitted)", keyStr)
	})
	if err != nil {
		return err
	}

	// Clear resolved indeterminate ops for next cycle
	o.indeterminateOps = nil

	return nil
}

// AckedCount returns the number of live keys acknowledged in the oracle.
func (o *Oracle) AckedCount() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return len(o.ackedKeys)
}
