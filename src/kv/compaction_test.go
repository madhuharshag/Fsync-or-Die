package kv

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestCompaction_ReclaimsDeletedSpace(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "compact_test.db")

	engine, _, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer engine.Close()

	// 1. Insert 500 keys
	for i := 0; i < 500; i++ {
		k := []byte(fmt.Sprintf("key_%04d", i))
		v := []byte(fmt.Sprintf("value_payload_%04d_with_padding_data_to_consume_bytes", i))
		if err := engine.Put(k, v); err != nil {
			t.Fatalf("Put failed at %d: %v", i, err)
		}
	}

	// 2. Delete 400 keys
	for i := 0; i < 400; i++ {
		k := []byte(fmt.Sprintf("key_%04d", i))
		if err := engine.Delete(k); err != nil {
			t.Fatalf("Delete failed at %d: %v", i, err)
		}
	}

	fiBefore, err := os.Stat(dbPath)
	if err != nil {
		t.Fatalf("Stat before failed: %v", err)
	}
	sizeBefore := fiBefore.Size()

	// 3. Trigger Compaction
	if err := engine.Compact(); err != nil {
		t.Fatalf("Compact failed: %v", err)
	}

	fiAfter, err := os.Stat(dbPath)
	if err != nil {
		t.Fatalf("Stat after failed: %v", err)
	}
	sizeAfter := fiAfter.Size()

	if sizeAfter >= sizeBefore {
		t.Fatalf("Expected file to shrink after compaction: before=%d, after=%d", sizeBefore, sizeAfter)
	}

	// 4. Verify deleted keys return ErrNotFound
	for i := 0; i < 400; i++ {
		k := []byte(fmt.Sprintf("key_%04d", i))
		_, err := engine.Get(k)
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("Expected ErrNotFound for deleted key %s, got: %v", k, err)
		}
	}

	// 5. Verify live 100 keys return correct values
	for i := 400; i < 500; i++ {
		k := []byte(fmt.Sprintf("key_%04d", i))
		expectedVal := []byte(fmt.Sprintf("value_payload_%04d_with_padding_data_to_consume_bytes", i))
		val, err := engine.Get(k)
		if err != nil {
			t.Fatalf("Get failed for live key %s: %v", k, err)
		}
		if !bytes.Equal(val, expectedVal) {
			t.Fatalf("Value mismatch for key %s: got %q, want %q", k, val, expectedVal)
		}
	}
}

func TestCompaction_AllDeletedShrinksToHeader(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "all_deleted.db")

	engine, _, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer engine.Close()

	// Insert and then delete all keys
	for i := 0; i < 50; i++ {
		k := []byte(fmt.Sprintf("k%d", i))
		v := []byte(fmt.Sprintf("v%d", i))
		_ = engine.Put(k, v)
		_ = engine.Delete(k)
	}

	if err := engine.Compact(); err != nil {
		t.Fatalf("Compact failed: %v", err)
	}

	fi, err := os.Stat(dbPath)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if fi.Size() != FileHeaderSize {
		t.Fatalf("Expected file size %d (header only), got %d", FileHeaderSize, fi.Size())
	}

	stats := engine.Stats()
	if stats.LiveKeys != 0 {
		t.Fatalf("Expected 0 live keys, got %d", stats.LiveKeys)
	}
}

func TestCompaction_CleansUpStrayTempFileOnOpen(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "stray_tmp.db")
	tmpPath := dbPath + ".compact.tmp"

	// Create a database and populate some keys
	engine, _, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	_ = engine.Put([]byte("key1"), []byte("val1"))
	_ = engine.Close()

	// Simulate a crash during compaction that left a stray .compact.tmp file
	if err := os.WriteFile(tmpPath, []byte("stray garbage from crashed compaction"), 0644); err != nil {
		t.Fatalf("WriteFile stray tmp failed: %v", err)
	}

	// Reopen database
	reopened, _, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Reopen failed: %v", err)
	}
	defer reopened.Close()

	// Stray temp file must have been cleaned up (Scenario C5)
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Fatalf("Expected stray .compact.tmp to be cleaned up, but it still exists")
	}

	// Verify original data intact
	val, err := reopened.Get([]byte("key1"))
	if err != nil || !bytes.Equal(val, []byte("val1")) {
		t.Fatalf("Expected val1, got: %s (err: %v)", val, err)
	}
}

func TestCompaction_ReopeningAfterCompactionPreservesData(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "reopen_after_compact.db")

	engine, _, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	for i := 0; i < 100; i++ {
		_ = engine.Put([]byte(fmt.Sprintf("k_%d", i)), []byte(fmt.Sprintf("v_%d", i)))
	}
	for i := 0; i < 50; i++ {
		_ = engine.Delete([]byte(fmt.Sprintf("k_%d", i)))
	}

	if err := engine.Compact(); err != nil {
		t.Fatalf("Compact failed: %v", err)
	}
	_ = engine.Close()

	// Reopen database
	reopened, info, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Reopen failed: %v", err)
	}
	defer reopened.Close()

	if info.LiveKeyCount != 50 {
		t.Fatalf("Expected 50 live keys after reopen, got %d", info.LiveKeyCount)
	}

	for i := 50; i < 100; i++ {
		k := []byte(fmt.Sprintf("k_%d", i))
		expected := []byte(fmt.Sprintf("v_%d", i))
		val, err := reopened.Get(k)
		if err != nil || !bytes.Equal(val, expected) {
			t.Fatalf("Mismatch on %s: got %q, want %q", k, val, expected)
		}
	}
}

func TestCompaction_ReadOnlyRejection(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "readonly.db")

	engine, _, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	_ = engine.Put([]byte("key"), []byte("value"))
	_ = engine.Close()

	roEngine, _, err := Open(dbPath, Options{ReadOnly: true})
	if err != nil {
		t.Fatalf("Open read-only failed: %v", err)
	}
	defer roEngine.Close()

	if err := roEngine.Compact(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("Expected ErrInvalid when compacting read-only db, got: %v", err)
	}
}
