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

func TestCompaction_ByteBoundarySplitting(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "byte_boundary.db")

	engine, _, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer engine.Close()

	// Insert 20 keys with 60-byte values (each op ~ 78 bytes)
	for i := 0; i < 20; i++ {
		k := []byte(fmt.Sprintf("split_key_%02d", i))
		v := bytes.Repeat([]byte("X"), 60)
		if err := engine.Put(k, v); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
	}

	// Override compaction thresholds to force splitting across multiple transactions.
	// Threshold: 300 bytes payload (~3-4 ops per transaction).
	oldBatchBytes := compactionBatchBytes
	oldBatchOps := compactionBatchOps
	compactionBatchBytes = 300
	compactionBatchOps = 1000
	defer func() {
		compactionBatchBytes = oldBatchBytes
		compactionBatchOps = oldBatchOps
	}()

	if err := engine.Compact(); err != nil {
		t.Fatalf("Compact failed: %v", err)
	}

	// Verify all 20 keys survive intact with exact values
	for i := 0; i < 20; i++ {
		k := []byte(fmt.Sprintf("split_key_%02d", i))
		expected := bytes.Repeat([]byte("X"), 60)
		val, err := engine.Get(k)
		if err != nil || !bytes.Equal(val, expected) {
			t.Fatalf("Mismatch on %s: got %v (len %d), want len %d", k, err, len(val), len(expected))
		}
	}
}

func TestCompaction_SingleMaxSizedValue(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "single_max_val.db")

	engine, _, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer engine.Close()

	// Insert 1 MB value (MaxValueLen = 1024 * 1024)
	largeVal := bytes.Repeat([]byte("M"), MaxValueLen)
	if err := engine.Put([]byte("large_key"), largeVal); err != nil {
		t.Fatalf("Put 1MB failed: %v", err)
	}

	if err := engine.Compact(); err != nil {
		t.Fatalf("Compact with 1MB value failed: %v", err)
	}

	got, err := engine.Get([]byte("large_key"))
	if err != nil {
		t.Fatalf("Get 1MB failed: %v", err)
	}
	if len(got) != MaxValueLen || !bytes.Equal(got, largeVal) {
		t.Fatalf("1MB value corrupted after compaction: got len %d, want %d", len(got), MaxValueLen)
	}
}

func TestCompaction_MultipleLargeValues(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "multi_large_val.db")

	engine, _, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	// Write three 500 KB values (total ~1.5 MB)
	vSize := 500 * 1024
	v1 := bytes.Repeat([]byte("A"), vSize)
	v2 := bytes.Repeat([]byte("B"), vSize)
	v3 := bytes.Repeat([]byte("C"), vSize)

	if err := engine.Put([]byte("k1"), v1); err != nil {
		t.Fatalf("Put k1 failed: %v", err)
	}
	if err := engine.Put([]byte("k2"), v2); err != nil {
		t.Fatalf("Put k2 failed: %v", err)
	}
	if err := engine.Put([]byte("k3"), v3); err != nil {
		t.Fatalf("Put k3 failed: %v", err)
	}

	// Delete k2
	if err := engine.Delete([]byte("k2")); err != nil {
		t.Fatalf("Delete k2 failed: %v", err)
	}

	fiBefore, _ := os.Stat(dbPath)
	sizeBefore := fiBefore.Size()

	if err := engine.Compact(); err != nil {
		t.Fatalf("Compact failed: %v", err)
	}

	fiAfter, _ := os.Stat(dbPath)
	sizeAfter := fiAfter.Size()

	// File must shrink by at least the deleted 500 KB value
	if sizeAfter >= sizeBefore {
		t.Fatalf("Expected file to shrink: before=%d, after=%d", sizeBefore, sizeAfter)
	}

	// Reopen to verify recovery from disk
	_ = engine.Close()
	reopened, info, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Reopen failed: %v", err)
	}
	defer reopened.Close()

	if info.LiveKeyCount != 2 {
		t.Fatalf("Expected 2 live keys, got %d", info.LiveKeyCount)
	}

	// k2 must be not found
	if _, err := reopened.Get([]byte("k2")); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Expected ErrNotFound for k2, got %v", err)
	}

	// k1 and k3 must match exactly
	got1, err := reopened.Get([]byte("k1"))
	if err != nil || !bytes.Equal(got1, v1) {
		t.Fatalf("k1 mismatch: got len %d, want %d", len(got1), len(v1))
	}
	got3, err := reopened.Get([]byte("k3"))
	if err != nil || !bytes.Equal(got3, v3) {
		t.Fatalf("k3 mismatch: got len %d, want %d", len(got3), len(v3))
	}
}

func TestCompaction_OperationIndividuallyTooLarge(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "too_large_op.db")

	engine, _, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer engine.Close()

	// Insert a 10 KB value
	largeVal := bytes.Repeat([]byte("L"), 10*1024)
	if err := engine.Put([]byte("large_key"), largeVal); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Set compactionBatchBytes lower than the single op size (e.g. 5 KB)
	oldBytes := compactionBatchBytes
	compactionBatchBytes = 5 * 1024
	defer func() {
		compactionBatchBytes = oldBytes
	}()

	// Compaction must return ErrInvalid and fail cleanly
	err = engine.Compact()
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected ErrInvalid when operation exceeds compaction batch byte policy, got: %v", err)
	}

	// The original database must remain intact and completely readable
	val, err := engine.Get([]byte("large_key"))
	if err != nil || !bytes.Equal(val, largeVal) {
		t.Fatalf("original key corrupted after rejected compaction: %v", err)
	}
}

func TestCompaction_ManySmallValuesOpCountLimit(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "many_small_ops.db")

	engine, _, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer engine.Close()

	// Write 500 small keys
	const totalKeys = 500
	for i := 0; i < totalKeys; i++ {
		k := []byte(fmt.Sprintf("k_%04d", i))
		v := []byte(fmt.Sprintf("v_%04d", i))
		if err := engine.Put(k, v); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
	}

	// Constrain compaction to 50 operations per batch
	oldOps := compactionBatchOps
	compactionBatchOps = 50
	defer func() {
		compactionBatchOps = oldOps
	}()

	if err := engine.Compact(); err != nil {
		t.Fatalf("Compact with 50-op batches failed: %v", err)
	}

	// Verify all 500 keys survive compaction
	for i := 0; i < totalKeys; i++ {
		k := []byte(fmt.Sprintf("k_%04d", i))
		expected := fmt.Sprintf("v_%04d", i)
		val, err := engine.Get(k)
		if err != nil || string(val) != expected {
			t.Fatalf("key %s corrupted or missing: got %q, want %q", k, val, expected)
		}
	}
}
