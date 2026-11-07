package kv

import (
	"sort"
	"sync"
)

// IndexEntry stores the location and length of a live value in the authoritative file.
// Values are not stored in memory to satisfy the 1,000,000+ key scaling requirement.
type IndexEntry struct {
	ValueOffset int64  // Byte offset in the file where value bytes start
	ValueLength uint32 // Exact byte length of the value
}

// PendingIndexUpdate represents an uncommitted change to be published atomically.
type PendingIndexUpdate struct {
	Key      []byte
	Entry    IndexEntry
	IsDelete bool
}

// Index represents the in-memory key-to-offset index.
// In the integrated database, Index is protected by DB.mu.
// A local RWMutex is also provided for safe standalone or concurrent test usage.
type Index struct {
	mu      sync.RWMutex
	entries map[string]IndexEntry
}

// NewIndex initializes an empty in-memory index.
func NewIndex() *Index {
	return &Index{
		entries: make(map[string]IndexEntry),
	}
}

// Get returns the IndexEntry for key. Returns (entry, true) if found, (empty, false) if absent.
func (idx *Index) Get(key []byte) (IndexEntry, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	entry, exists := idx.entries[string(key)]
	return entry, exists
}

// Put records or overwrites the file location of key.
func (idx *Index) Put(key []byte, offset int64, length uint32) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.entries[string(key)] = IndexEntry{
		ValueOffset: offset,
		ValueLength: length,
	}
}

// Delete removes key from the index. Deleting an absent key is a no-op.
func (idx *Index) Delete(key []byte) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	delete(idx.entries, string(key))
}

// ApplyUpdates atomically applies a list of pending transaction updates in list order.
// This is used by the common transaction commit path to publish changes only after final Sync.
func (idx *Index) ApplyUpdates(updates []PendingIndexUpdate) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	for _, u := range updates {
		k := string(u.Key)
		if u.IsDelete {
			delete(idx.entries, k)
		} else {
			idx.entries[k] = u.Entry
		}
	}
}

// Len returns the current count of active, non-deleted keys.
func (idx *Index) Len() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.entries)
}

// Keys returns a slice of all active keys in lexicographical order.
// Each returned key is an independent, caller-owned byte slice.
func (idx *Index) Keys() [][]byte {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	strKeys := make([]string, 0, len(idx.entries))
	for k := range idx.entries {
		strKeys = append(strKeys, k)
	}
	sort.Strings(strKeys)

	result := make([][]byte, len(strKeys))
	for i, k := range strKeys {
		b := []byte(k)
		result[i] = b
	}
	return result
}

// ForEach iterates through all entries in the index.
// fn receives an independent copy of key. If fn returns an error, iteration halts.
func (idx *Index) ForEach(fn func(key []byte, entry IndexEntry) error) error {
	idx.mu.RLock()
	// Take snapshot of entries to prevent deadlock if fn performs read operations
	snapshot := make(map[string]IndexEntry, len(idx.entries))
	for k, v := range idx.entries {
		snapshot[k] = v
	}
	idx.mu.RUnlock()

	for k, entry := range snapshot {
		if err := fn([]byte(k), entry); err != nil {
			return err
		}
	}
	return nil
}

// Clone creates an independent copy of the index.
// Used during compaction to construct the replacement index before descriptor swap.
func (idx *Index) Clone() *Index {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	copied := make(map[string]IndexEntry, len(idx.entries))
	for k, v := range idx.entries {
		copied[k] = v
	}
	return &Index{
		entries: copied,
	}
}

// ReplaceWith atomically replaces all index entries with those of another index.
// Used when publishing the compacted replacement index under the database lock.
func (idx *Index) ReplaceWith(other *Index) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	other.mu.RLock()
	defer other.mu.RUnlock()

	newMap := make(map[string]IndexEntry, len(other.entries))
	for k, v := range other.entries {
		newMap[k] = v
	}
	idx.entries = newMap
}
