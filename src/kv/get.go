package kv

import (
	"fmt"
)

// Get retrieves the current value for key.
// Returns ErrNotFound if the key does not exist or was deleted.
// Returns a caller-owned byte slice safe from concurrent mutation.
func (db *DB) Get(key []byte) ([]byte, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	return db.getLocked(key)
}

// getLocked executes index lookup and exact value read with db.mu held.
func (db *DB) getLocked(key []byte) ([]byte, error) {
	if err := db.checkStateLocked(); err != nil {
		return nil, err
	}
	if len(key) < MinKeyLen || len(key) > MaxKeyLen {
		return nil, fmt.Errorf("%w: key length %d out of bounds [%d, %d]", ErrInvalid, len(key), MinKeyLen, MaxKeyLen)
	}

	entry, found := db.index.Get(key)
	if !found {
		return nil, ErrNotFound
	}

	// Empty value is valid and distinct from deletion
	if entry.ValueLength == 0 {
		return []byte{}, nil
	}

	valBuf := make([]byte, entry.ValueLength)
	if err := ReadAtFull(db.file, valBuf, entry.ValueOffset); err != nil {
		return nil, fmt.Errorf("failed to read value at offset %d: %w", entry.ValueOffset, err)
	}

	return valBuf, nil
}
