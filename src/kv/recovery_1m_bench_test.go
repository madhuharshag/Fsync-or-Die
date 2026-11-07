package kv

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// TestRecovery_1MillionKeys_ColdOpen generates a real on-disk database with 1,000,000 unique
// keys across production committed transactions, closes it, and measures a fresh cold Open()
// and crash recovery, recording file size, transaction count, duration, and memory metrics.
func TestRecovery_1MillionKeys_ColdOpen(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping 1,000,000-key benchmark in short mode")
	}

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "recovery_1m.db")

	t.Log("Populating 1,000,000 unique keys using production write and commit path...")

	db, _, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	// 1,000,000 keys written in 100 transactions of 10,000 operations each
	const totalKeys = 1000000
	const batchSize = 10000
	const numBatches = totalKeys / batchSize

	popStart := time.Now()
	for b := 0; b < numBatches; b++ {
		ops := make([]Op, batchSize)
		base := b * batchSize
		for i := 0; i < batchSize; i++ {
			k := fmt.Sprintf("k%08d", base+i)
			v := fmt.Sprintf("val%08d", base+i)
			ops[i] = Op{
				Kind:  PutOp,
				Key:   []byte(k),
				Value: []byte(v),
			}
		}
		if err := db.Batch(ops); err != nil {
			t.Fatalf("Batch %d failed: %v", b, err)
		}
	}

	popDuration := time.Since(popStart)
	t.Logf("Populated 1,000,000 keys in %v (avg %v/batch)", popDuration, popDuration/numBatches)

	if err := db.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	fi, err := os.Stat(dbPath)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	fileSize := fi.Size()

	// Force GC before measuring recovery to isolate recovery allocations
	runtime.GC()
	var mBefore runtime.MemStats
	runtime.ReadMemStats(&mBefore)

	// MEASURE COLD OPEN & RECOVERY
	recStart := time.Now()
	reopened, info, err := Open(dbPath, Options{})
	recDuration := time.Since(recStart)

	if err != nil {
		t.Fatalf("Cold Open() and recovery failed: %v", err)
	}
	defer reopened.Close()

	var mAfter runtime.MemStats
	runtime.ReadMemStats(&mAfter)
	allocBytes := mAfter.Alloc - mBefore.Alloc

	// Verify key count
	if info.LiveKeyCount != totalKeys {
		t.Fatalf("LiveKeyCount mismatch: got %d, expected %d", info.LiveKeyCount, totalKeys)
	}

	// Verify a sample of keys to ensure data integrity
	for _, sampleIdx := range []int{0, 1, 99999, 500000, 999999} {
		sampleKey := fmt.Sprintf("k%08d", sampleIdx)
		expectedVal := fmt.Sprintf("val%08d", sampleIdx)
		val, err := reopened.Get([]byte(sampleKey))
		if err != nil || string(val) != expectedVal {
			t.Fatalf("Sample key %s mismatch: got %q, want %q (err=%v)", sampleKey, val, expectedVal, err)
		}
	}

	t.Log("================================================================")
	t.Log(" 1,000,000-KEY COLD RECOVERY BENCHMARK RESULTS")
	t.Log("================================================================")
	t.Logf("Environment:          %s / %s (Go %s)", runtime.GOOS, runtime.GOARCH, runtime.Version())
	t.Logf("CPUs Available:       %d", runtime.NumCPU())
	t.Logf("Database File Size:   %.2f MB (%d bytes)", float64(fileSize)/(1024*1024), fileSize)
	t.Logf("Recovered Txns:       %d", info.RecoveredTransactions)
	t.Logf("Recovered Live Keys:  %d", info.LiveKeyCount)
	t.Logf("Internal Timer:       %.3f ms (%d ns)", float64(info.RecoveryDurationNs)/1e6, info.RecoveryDurationNs)
	t.Logf("Wall-Clock Elapsed:   %v (%.3f seconds)", recDuration, recDuration.Seconds())
	t.Logf("Memory Allocated:     %.2f MB (%d bytes)", float64(allocBytes)/(1024*1024), allocBytes)
	t.Logf("Meets < 2.0s Goal:    %v", recDuration < 2*time.Second)
	t.Log("================================================================")
}

// BenchmarkRecovery_1MillionKeys measures repeated recovery performance.
func BenchmarkRecovery_1MillionKeys(b *testing.B) {
	dir := b.TempDir()
	dbPath := filepath.Join(dir, "bench_1m.db")

	db, _, err := Open(dbPath, Options{})
	if err != nil {
		b.Fatalf("Open failed: %v", err)
	}

	const totalKeys = 1000000
	const batchSize = 10000
	const numBatches = totalKeys / batchSize

	for batch := 0; batch < numBatches; batch++ {
		ops := make([]Op, batchSize)
		base := batch * batchSize
		for i := 0; i < batchSize; i++ {
			k := fmt.Sprintf("k%08d", base+i)
			v := fmt.Sprintf("v%08d", base+i)
			ops[i] = Op{
				Kind:  PutOp,
				Key:   []byte(k),
				Value: []byte(v),
			}
		}
		if err := db.Batch(ops); err != nil {
			b.Fatalf("Batch failed: %v", err)
		}
	}
	_ = db.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reopened, info, err := Open(dbPath, Options{})
		if err != nil {
			b.Fatalf("Reopen failed: %v", err)
		}
		if info.LiveKeyCount != totalKeys {
			b.Fatalf("LiveKeyCount mismatch: got %d", info.LiveKeyCount)
		}
		_ = reopened.Close()
	}
}
