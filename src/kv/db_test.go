package kv

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func createTestDB(t *testing.T, opts Options) (Engine, string, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "kipdb_db_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	dbPath := filepath.Join(dir, "test.kip")
	db, info, err := Open(dbPath, opts)
	if err != nil {
		os.RemoveAll(dir)
		t.Fatalf("failed to open database: %v", err)
	}
	_ = info

	cleanup := func() {
		_ = db.Close()
		os.RemoveAll(dir)
	}
	return db, dbPath, cleanup
}

func TestDB_PutGetOverwriteDelete(t *testing.T) {
	db, _, cleanup := createTestDB(t, Options{})
	defer cleanup()

	// Missing key
	_, err := db.Get([]byte("missing"))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	// Put
	k := []byte("account/101")
	v1 := []byte("balance=500")
	if err := db.Put(k, v1); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Get
	got, err := db.Get(k)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !bytes.Equal(got, v1) {
		t.Fatalf("expected %q, got %q", v1, got)
	}

	// Overwrite
	v2 := []byte("balance=1200")
	if err := db.Put(k, v2); err != nil {
		t.Fatalf("Overwrite failed: %v", err)
	}
	got, err = db.Get(k)
	if err != nil {
		t.Fatalf("Get after overwrite failed: %v", err)
	}
	if !bytes.Equal(got, v2) {
		t.Fatalf("expected %q, got %q", v2, got)
	}

	// Delete
	if err := db.Delete(k); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	_, err = db.Get(k)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after Delete, got %v", err)
	}

	// Delete missing key is safe no-op
	if err := db.Delete([]byte("missing_key_delete")); err != nil {
		t.Fatalf("expected nil on missing delete, got %v", err)
	}
}

func TestDB_EmptyValueIsNotDeletion(t *testing.T) {
	db, _, cleanup := createTestDB(t, Options{})
	defer cleanup()

	k := []byte("flag/active")
	emptyVal := []byte{}

	if err := db.Put(k, emptyVal); err != nil {
		t.Fatalf("Put empty value failed: %v", err)
	}

	got, err := db.Get(k)
	if err != nil {
		t.Fatalf("Get empty value returned error: %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("expected empty non-nil slice, got %v", got)
	}
}

func TestDB_CallerOwnedReturnedBytes(t *testing.T) {
	db, _, cleanup := createTestDB(t, Options{})
	defer cleanup()

	k := []byte("secret")
	v := []byte("original")
	if err := db.Put(k, v); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	got, err := db.Get(k)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Mutate returned buffer
	got[0] = 'X'

	// Re-fetch, original must remain intact
	fresh, err := db.Get(k)
	if err != nil {
		t.Fatalf("second Get failed: %v", err)
	}
	if !bytes.Equal(fresh, v) {
		t.Fatalf("caller mutation affected internal state: %q vs %q", fresh, v)
	}
}

func TestDB_BatchAtomicityAndOrdering(t *testing.T) {
	db, _, cleanup := createTestDB(t, Options{})
	defer cleanup()

	ops := []Op{
		{Kind: PutOp, Key: []byte("item1"), Value: []byte("v1")},
		{Kind: PutOp, Key: []byte("item2"), Value: []byte("v2")},
		{Kind: PutOp, Key: []byte("item3"), Value: []byte("v3")},
		{Kind: DeleteOp, Key: []byte("item2"), Value: nil}, // Delete item2 in same batch
	}

	if err := db.Batch(ops); err != nil {
		t.Fatalf("Batch failed: %v", err)
	}

	// item1 and item3 should exist; item2 should not exist
	v1, err := db.Get([]byte("item1"))
	if err != nil || !bytes.Equal(v1, []byte("v1")) {
		t.Fatalf("item1 missing or wrong")
	}
	v3, err := db.Get([]byte("item3"))
	if err != nil || !bytes.Equal(v3, []byte("v3")) {
		t.Fatalf("item3 missing or wrong")
	}
	_, err = db.Get([]byte("item2"))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("item2 should be deleted")
	}
}

func TestDB_ReopenPreservesData(t *testing.T) {
	dir, err := os.MkdirTemp("", "kipdb_reopen_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)
	dbPath := filepath.Join(dir, "reopen.kip")

	// Phase 1: Write entries and close cleanly
	db1, info1, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("initial Open failed: %v", err)
	}
	if !info1.Recreated {
		t.Fatalf("expected Recreated=true on initial open")
	}

	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("key/%03d", i)
		val := fmt.Sprintf("val/%03d", i)
		if err := db1.Put([]byte(key), []byte(val)); err != nil {
			t.Fatalf("Put %d failed: %v", i, err)
		}
	}
	// Delete key/025
	if err := db1.Delete([]byte("key/025")); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if err := db1.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Phase 2: Reopen and verify index recovery
	db2, info2, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("reopen failed: %v", err)
	}
	defer db2.Close()

	if info2.Recreated {
		t.Fatalf("expected Recreated=false on reopen")
	}
	if info2.LiveKeyCount != 49 {
		t.Fatalf("expected 49 live keys, got %d", info2.LiveKeyCount)
	}

	// Verify live keys
	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("key/%03d", i)
		expectedVal := fmt.Sprintf("val/%03d", i)
		val, err := db2.Get([]byte(key))
		if i == 25 {
			if !errors.Is(err, ErrNotFound) {
				t.Fatalf("key/025 should be deleted, got %v", err)
			}
		} else {
			if err != nil || !bytes.Equal(val, []byte(expectedVal)) {
				t.Fatalf("key %s: expected %q, got %q (err=%v)", key, expectedVal, val, err)
			}
		}
	}
}

func TestDB_TornTailRepairAndSubsequentWrite(t *testing.T) {
	dir, err := os.MkdirTemp("", "kipdb_tail_repair_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)
	dbPath := filepath.Join(dir, "torn.kip")

	// 1. Write 3 valid transactions
	db1, _, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	_ = db1.Put([]byte("alpha"), []byte("111"))
	_ = db1.Put([]byte("beta"), []byte("222"))
	_ = db1.Put([]byte("gamma"), []byte("333"))
	_ = db1.Close()

	// 2. Deliberately append a torn/incomplete transaction suffix
	f, err := os.OpenFile(dbPath, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		t.Fatalf("failed to open file for tail corruption: %v", err)
	}
	// Append 18 bytes of partial garbage (e.g. truncated header)
	_, _ = f.Write([]byte("TXN1\x01\x00\x00\x00partial-bytes"))
	_ = f.Close()

	// 3. Reopen: recovery must detect torn tail, truncate it, and report RepairedTailBytes > 0
	db2, info2, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("reopen after torn tail failed: %v", err)
	}
	if info2.RepairedTailBytes == 0 {
		t.Fatalf("expected RepairedTailBytes > 0, got 0")
	}
	if info2.LiveKeyCount != 3 {
		t.Fatalf("expected 3 valid recovered keys, got %d", info2.LiveKeyCount)
	}

	// 4. Attempt a new write right after repair
	if err := db2.Put([]byte("delta"), []byte("444")); err != nil {
		t.Fatalf("Put after torn tail repair failed: %v", err)
	}
	_ = db2.Close()

	// 5. Reopen again: verify that all 3 original keys AND delta are present and valid
	db3, info3, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("reopen 3 failed: %v", err)
	}
	defer db3.Close()

	if info3.LiveKeyCount != 4 {
		t.Fatalf("expected 4 live keys after reopen, got %d", info3.LiveKeyCount)
	}
	for k, expectedVal := range map[string]string{
		"alpha": "111",
		"beta":  "222",
		"gamma": "333",
		"delta": "444",
	} {
		val, err := db3.Get([]byte(k))
		if err != nil || string(val) != expectedVal {
			t.Fatalf("key %s mismatch: got %q, expected %q (err=%v)", k, string(val), expectedVal, err)
		}
	}
}

func TestDB_CapacityEnforcement(t *testing.T) {
	// 32-byte header + 78-byte transaction = 110 bytes needed for 1 record
	// Setting maxBytes = 100 allows file header (32 bytes) but rejects first transaction (78 bytes)
	dir, err := os.MkdirTemp("", "kipdb_cap_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)
	dbPath := filepath.Join(dir, "cap.kip")

	db, _, err := Open(dbPath, Options{MaxFileBytes: 100})
	if err != nil {
		t.Fatalf("Open with cap failed: %v", err)
	}
	defer db.Close()

	err = db.Put([]byte("rice"), []byte("500"))
	if !errors.Is(err, ErrFull) {
		t.Fatalf("expected ErrFull, got %v", err)
	}

	// Ensure no partial data appended
	fi, _ := os.Stat(dbPath)
	if fi.Size() != 32 {
		t.Fatalf("expected file to remain exactly 32 bytes (header only), got %d", fi.Size())
	}
}

func TestDB_ConcurrentWritersAndReaders(t *testing.T) {
	db, _, cleanup := createTestDB(t, Options{})
	defer cleanup()

	var wg sync.WaitGroup
	numWorkers := 4
	opsPerWorker := 50

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < opsPerWorker; i++ {
				key := []byte(fmt.Sprintf("w%d/key%d", workerID, i))
				val := []byte(fmt.Sprintf("val-%d-%d", workerID, i))
				if err := db.Put(key, val); err != nil {
					t.Errorf("worker %d Put error: %v", workerID, err)
				}
				readVal, err := db.Get(key)
				if err != nil || !bytes.Equal(readVal, val) {
					t.Errorf("worker %d Get mismatch: got %q, expected %q", workerID, readVal, val)
				}
			}
		}(w)
	}

	wg.Wait()
	stats := db.Stats()
	if stats.LiveKeys != uint64(numWorkers*opsPerWorker) {
		t.Fatalf("expected %d live keys, got %d", numWorkers*opsPerWorker, stats.LiveKeys)
	}
}

func TestDB_ProcessLockConflictAndRelease(t *testing.T) {
	dir, err := os.MkdirTemp("", "kipdb_lock_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)
	dbPath := filepath.Join(dir, "locked.kip")

	db1, _, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("db1 Open failed: %v", err)
	}

	// Concurrent open of same path from second handle must return ErrBusy
	_, _, err = Open(dbPath, Options{})
	if !errors.Is(err, ErrBusy) {
		t.Fatalf("expected ErrBusy when opening already-locked database, got %v", err)
	}

	// After closing db1, db2 can open successfully
	if err := db1.Close(); err != nil {
		t.Fatalf("db1 Close failed: %v", err)
	}

	db2, _, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("db2 Open failed after db1 closed: %v", err)
	}
	_ = db2.Close()
}

func TestDB_PreserveStoredCapOnReopenWithDefaultOptions(t *testing.T) {
	dir, err := os.MkdirTemp("", "kipdb_stored_cap_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)
	dbPath := filepath.Join(dir, "stored_cap.kip")

	// 1. Open with explicit cap = 200 bytes
	db1, _, err := Open(dbPath, Options{MaxFileBytes: 200})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	// 32 header + 78 txn = 110 bytes used
	if err := db1.Put([]byte("rice"), []byte("500")); err != nil {
		t.Fatalf("Put 1 failed: %v", err)
	}
	_ = db1.Close()

	// 2. Reopen with default Options{} (MaxFileBytes = 0)
	// Database must preserve the stored cap of 200 bytes from the file header!
	db2, _, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("reopen with default options failed: %v", err)
	}
	defer db2.Close()

	if db2.Stats().MaxFileBytes != 200 {
		t.Fatalf("expected stored cap 200 to be preserved, got %d", db2.Stats().MaxFileBytes)
	}

	// 3. Put 2nd record: 110 + 78 = 188 <= 200 (should succeed)
	if err := db2.Put([]byte("corn"), []byte("250")); err != nil {
		t.Fatalf("Put 2 failed: %v", err)
	}

	// 4. Put 3rd record: 188 + 78 = 266 > 200 (must be cleanly rejected with ErrFull)
	err = db2.Put([]byte("oats"), []byte("100"))
	if !errors.Is(err, ErrFull) {
		t.Fatalf("expected ErrFull when exceeding preserved cap, got %v", err)
	}
}

func TestDB_OperationsAfterClose(t *testing.T) {
	db, _, cleanup := createTestDB(t, Options{})
	defer cleanup()

	_ = db.Close()

	if err := db.Put([]byte("k"), []byte("v")); !errors.Is(err, ErrClosed) {
		t.Fatalf("expected ErrClosed for Put, got %v", err)
	}
	if _, err := db.Get([]byte("k")); !errors.Is(err, ErrClosed) {
		t.Fatalf("expected ErrClosed for Get, got %v", err)
	}
	if err := db.Delete([]byte("k")); !errors.Is(err, ErrClosed) {
		t.Fatalf("expected ErrClosed for Delete, got %v", err)
	}
	if err := db.Batch([]Op{{Kind: PutOp, Key: []byte("k"), Value: []byte("v")}}); !errors.Is(err, ErrClosed) {
		t.Fatalf("expected ErrClosed for Batch, got %v", err)
	}
	if err := db.Scan(func(k, v []byte) error { return nil }); !errors.Is(err, ErrClosed) {
		t.Fatalf("expected ErrClosed for Scan, got %v", err)
	}
	if err := db.Check(); !errors.Is(err, ErrClosed) {
		t.Fatalf("expected ErrClosed for Check, got %v", err)
	}
}

func TestDB_PoisonedHandleRejection(t *testing.T) {
	db, _, cleanup := createTestDB(t, Options{})
	defer cleanup()

	rawDB, ok := db.(*DB)
	if !ok {
		t.Fatalf("expected *DB concrete type")
	}

	// Simulate fatal I/O failure that poisoned the handle
	rawDB.mu.Lock()
	rawDB.poisoned = errors.New("simulated fatal disk hardware write error")
	rawDB.mu.Unlock()

	// All subsequent mutations and reads must be rejected with ErrPoisoned
	if err := db.Put([]byte("k"), []byte("v")); !errors.Is(err, ErrPoisoned) {
		t.Fatalf("expected ErrPoisoned on Put, got %v", err)
	}
	if _, err := db.Get([]byte("k")); !errors.Is(err, ErrPoisoned) {
		t.Fatalf("expected ErrPoisoned on Get, got %v", err)
	}
	if err := db.Delete([]byte("k")); !errors.Is(err, ErrPoisoned) {
		t.Fatalf("expected ErrPoisoned on Delete, got %v", err)
	}
}

func TestDB_InjectedSyncFailureOnTransactionPath(t *testing.T) {
	db, _, cleanup := createTestDB(t, Options{})
	defer cleanup()

	rawDB, ok := db.(*DB)
	if !ok {
		t.Fatalf("expected *DB concrete type")
	}

	// 1. Initial write with real sync succeeds
	if err := db.Put([]byte("key1"), []byte("val1")); err != nil {
		t.Fatalf("initial Put failed: %v", err)
	}

	// 2. Inject sync failure into the actual transaction commit path
	injectedErr := errors.New("simulated physical disk sync failure")
	rawDB.syncSeam = func(f *os.File) error {
		return injectedErr
	}

	// 3. Attempt mutation: must fail and return ErrUncertain (never nil/success)
	err := db.Put([]byte("sync_fail_key"), []byte("sync_val"))
	if err == nil {
		t.Fatalf("expected error on injected sync failure, got nil")
	}
	if !errors.Is(err, ErrUncertain) {
		t.Fatalf("expected ErrUncertain wrapping sync failure, got %v", err)
	}

	// 4. Handle must now be marked poisoned
	if rawDB.poisoned == nil {
		t.Fatalf("expected db.poisoned to be set after sync failure")
	}

	// 5. Subsequent mutations and lookups must be rejected with ErrPoisoned
	if err := db.Put([]byte("k2"), []byte("v2")); !errors.Is(err, ErrPoisoned) {
		t.Fatalf("expected ErrPoisoned on subsequent Put, got %v", err)
	}
	if _, err := db.Get([]byte("key1")); !errors.Is(err, ErrPoisoned) {
		t.Fatalf("expected ErrPoisoned on subsequent Get, got %v", err)
	}
	if err := db.Delete([]byte("key1")); !errors.Is(err, ErrPoisoned) {
		t.Fatalf("expected ErrPoisoned on subsequent Delete, got %v", err)
	}
	if err := db.Batch([]Op{{Kind: PutOp, Key: []byte("k3"), Value: []byte("v3")}}); !errors.Is(err, ErrPoisoned) {
		t.Fatalf("expected ErrPoisoned on subsequent Batch, got %v", err)
	}
}

func TestDB_CheckIntegrity(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "check_test.db")

	// 1. Initial empty database: Check must pass
	db, _, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	if err := db.Check(); err != nil {
		t.Fatalf("Check on empty database failed: %v", err)
	}

	// 2. Add multiple transactions: Check must pass
	if err := db.Put([]byte("key1"), []byte("value1")); err != nil {
		t.Fatalf("Put key1 failed: %v", err)
	}
	if err := db.Batch([]Op{
		{Kind: PutOp, Key: []byte("batch_k1"), Value: []byte("batch_v1")},
		{Kind: DeleteOp, Key: []byte("key1")},
	}); err != nil {
		t.Fatalf("Batch failed: %v", err)
	}
	if err := db.Check(); err != nil {
		t.Fatalf("Check after transactions failed: %v", err)
	}

	// Verify Stats
	stats := db.Stats()
	if stats.TxCount != 2 {
		t.Fatalf("expected TxCount 2, got %d", stats.TxCount)
	}

	if err := db.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// 3. Corrupt a byte in the middle of a transaction payload
	f, err := os.OpenFile(dbPath, os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("failed to open raw file: %v", err)
	}
	fi, _ := f.Stat()
	corruptOffset := fi.Size() - 40 // In the middle of payload/footer
	var originalByte [1]byte
	_, _ = f.ReadAt(originalByte[:], corruptOffset)
	_, _ = f.WriteAt([]byte{originalByte[0] ^ 0xFF}, corruptOffset)
	_ = f.Sync()
	_ = f.Close()

	// Reopen with ReadOnly to inspect Check failure
	dbCorrupt, _, err := Open(dbPath, Options{ReadOnly: true})
	if err != nil {
		// If Open itself refused to open due to corrupt tail, verify error
		if !errors.Is(err, ErrCorrupt) {
			t.Fatalf("expected ErrCorrupt on Open, got %v", err)
		}
	} else {
		defer dbCorrupt.Close()
		chkErr := dbCorrupt.Check()
		if chkErr == nil {
			t.Fatalf("expected Check to detect corruption, but got nil")
		}
		if !errors.Is(chkErr, ErrCorrupt) {
			t.Fatalf("expected ErrCorrupt, got %v", chkErr)
		}
	}

	// 4. Test Check with physical size smaller than FileHeader
	shortPath := filepath.Join(dir, "short.db")
	if err := os.WriteFile(shortPath, []byte("short"), 0644); err != nil {
		t.Fatalf("failed to write short file: %v", err)
	}
	_, _, err = Open(shortPath, Options{ReadOnly: true})
	if err == nil {
		t.Fatalf("expected Open of short file to fail")
	}
	if !errors.Is(err, ErrCorrupt) {
		t.Fatalf("expected ErrCorrupt for short file, got %v", err)
	}
}
