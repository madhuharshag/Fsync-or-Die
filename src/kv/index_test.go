package kv

import (
	"bytes"
	"testing"
)

func TestIndex_PutGetDelete(t *testing.T) {
	idx := NewIndex()
	k := []byte("user/101")

	// Get missing
	if _, ok := idx.Get(k); ok {
		t.Fatalf("expected key to be missing")
	}

	// Put
	idx.Put(k, 1024, 128)
	entry, ok := idx.Get(k)
	if !ok || entry.ValueOffset != 1024 || entry.ValueLength != 128 {
		t.Fatalf("unexpected entry: %+v, ok=%v", entry, ok)
	}
	if idx.Len() != 1 {
		t.Fatalf("expected len 1, got %d", idx.Len())
	}

	// Overwrite
	idx.Put(k, 2048, 256)
	entry, ok = idx.Get(k)
	if !ok || entry.ValueOffset != 2048 || entry.ValueLength != 256 {
		t.Fatalf("unexpected overwrite entry: %+v, ok=%v", entry, ok)
	}
	if idx.Len() != 1 {
		t.Fatalf("expected len 1 after overwrite, got %d", idx.Len())
	}

	// Delete
	idx.Delete(k)
	if _, ok := idx.Get(k); ok {
		t.Fatalf("expected key to be deleted")
	}
	if idx.Len() != 0 {
		t.Fatalf("expected len 0 after delete, got %d", idx.Len())
	}

	// Delete nonexistent key is safe no-op
	idx.Delete([]byte("nonexistent"))
	if idx.Len() != 0 {
		t.Fatalf("expected len 0")
	}
}

func TestIndex_ApplyUpdatesOrder(t *testing.T) {
	idx := NewIndex()
	k := []byte("flag")

	// Apply batch: Put -> Delete in same batch => absent
	updates1 := []PendingIndexUpdate{
		{Key: k, Entry: IndexEntry{ValueOffset: 100, ValueLength: 10}, IsDelete: false},
		{Key: k, Entry: IndexEntry{}, IsDelete: true},
	}
	idx.ApplyUpdates(updates1)
	if _, ok := idx.Get(k); ok {
		t.Fatalf("key should be absent after Put then Delete")
	}

	// Apply batch: Delete -> Put in same batch => present with final offset
	updates2 := []PendingIndexUpdate{
		{Key: k, Entry: IndexEntry{}, IsDelete: true},
		{Key: k, Entry: IndexEntry{ValueOffset: 500, ValueLength: 25}, IsDelete: false},
	}
	idx.ApplyUpdates(updates2)
	entry, ok := idx.Get(k)
	if !ok || entry.ValueOffset != 500 || entry.ValueLength != 25 {
		t.Fatalf("expected entry offset=500 len=25, got %+v, ok=%v", entry, ok)
	}
}

func TestIndex_KeysAndForEach(t *testing.T) {
	idx := NewIndex()
	idx.Put([]byte("c"), 30, 3)
	idx.Put([]byte("a"), 10, 1)
	idx.Put([]byte("b"), 20, 2)

	keys := idx.Keys()
	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(keys))
	}
	if !bytes.Equal(keys[0], []byte("a")) || !bytes.Equal(keys[1], []byte("b")) || !bytes.Equal(keys[2], []byte("c")) {
		t.Fatalf("keys not sorted: %s, %s, %s", keys[0], keys[1], keys[2])
	}

	count := 0
	err := idx.ForEach(func(k []byte, entry IndexEntry) error {
		count++
		return nil
	})
	if err != nil || count != 3 {
		t.Fatalf("ForEach failed: err=%v, count=%d", err, count)
	}
}

func TestIndex_CloneAndReplace(t *testing.T) {
	idx1 := NewIndex()
	idx1.Put([]byte("alpha"), 100, 10)

	cloned := idx1.Clone()
	entry, ok := cloned.Get([]byte("alpha"))
	if !ok || entry.ValueOffset != 100 {
		t.Fatalf("cloned index missing alpha")
	}

	// Modify original, cloned should remain unchanged
	idx1.Put([]byte("beta"), 200, 20)
	if _, ok := cloned.Get([]byte("beta")); ok {
		t.Fatalf("cloned index was mutated by original")
	}

	// Replace
	idx1.ReplaceWith(cloned)
	if _, ok := idx1.Get([]byte("beta")); ok {
		t.Fatalf("beta should have been wiped after ReplaceWith")
	}
}
