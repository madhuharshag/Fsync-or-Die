package kv

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkDB_Put(b *testing.B) {
	dir, err := os.MkdirTemp("", "kipdb_bench_*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(dir)

	dbPath := filepath.Join(dir, "bench.kip")
	db, _, err := Open(dbPath, Options{})
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	key := []byte("bench/key")
	val := []byte("bench/value/payload")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := db.Put(key, val); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDB_Get(b *testing.B) {
	dir, err := os.MkdirTemp("", "kipdb_bench_get_*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(dir)

	dbPath := filepath.Join(dir, "bench_get.kip")
	db, _, err := Open(dbPath, Options{})
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	key := []byte("bench/get/key")
	val := []byte("bench/get/value")
	_ = db.Put(key, val)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := db.Get(key)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDB_Batch100(b *testing.B) {
	dir, err := os.MkdirTemp("", "kipdb_bench_batch_*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(dir)

	dbPath := filepath.Join(dir, "bench_batch.kip")
	db, _, err := Open(dbPath, Options{})
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	ops := make([]Op, 100)
	for i := 0; i < 100; i++ {
		ops[i] = Op{
			Kind:  PutOp,
			Key:   []byte(fmt.Sprintf("k%d", i)),
			Value: []byte(fmt.Sprintf("v%d", i)),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := db.Batch(ops); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRecovery_50k(b *testing.B) {
	dir, err := os.MkdirTemp("", "kipdb_bench_rec_*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(dir)

	dbPath := filepath.Join(dir, "bench_rec.kip")
	db, _, err := Open(dbPath, Options{})
	if err != nil {
		b.Fatal(err)
	}

	// Insert 50,000 keys across 50 batches of 1,000 ops
	const totalKeys = 50000
	const batchSize = 1000
	for batch := 0; batch < totalKeys/batchSize; batch++ {
		ops := make([]Op, batchSize)
		for j := 0; j < batchSize; j++ {
			idx := batch*batchSize + j
			ops[j] = Op{
				Kind:  PutOp,
				Key:   []byte(fmt.Sprintf("bench_key_%08d", idx)),
				Value: []byte("recovery_benchmark_payload_value"),
			}
		}
		if err := db.Batch(ops); err != nil {
			b.Fatal(err)
		}
	}
	db.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		openedDB, info, err := Open(dbPath, Options{ReadOnly: true})
		if err != nil {
			b.Fatal(err)
		}
		if info.LiveKeyCount != totalKeys {
			b.Fatalf("expected %d keys, got %d", totalKeys, info.LiveKeyCount)
		}
		openedDB.Close()
	}
}

func BenchmarkRecovery_1M(b *testing.B) {
	dir, err := os.MkdirTemp("", "kipdb_bench_rec1m_*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(dir)

	dbPath := filepath.Join(dir, "bench_rec1m.kip")
	db, _, err := Open(dbPath, Options{})
	if err != nil {
		b.Fatal(err)
	}

	// Insert 1,000,000 keys across 200 batches of 5,000 ops
	const totalKeys = 1000000
	const batchSize = 5000
	for batch := 0; batch < totalKeys/batchSize; batch++ {
		ops := make([]Op, batchSize)
		for j := 0; j < batchSize; j++ {
			idx := batch*batchSize + j
			ops[j] = Op{
				Kind:  PutOp,
				Key:   []byte(fmt.Sprintf("k%07d", idx)),
				Value: []byte("v1"),
			}
		}
		if err := db.Batch(ops); err != nil {
			b.Fatal(err)
		}
	}
	db.Close()

	fi, err := os.Stat(dbPath)
	if err != nil {
		b.Fatal(err)
	}
	b.Logf("Database prepared for 1M keys. File size: %d bytes (%.2f MB)", fi.Size(), float64(fi.Size())/(1024*1024))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		openedDB, info, err := Open(dbPath, Options{ReadOnly: true})
		if err != nil {
			b.Fatal(err)
		}
		if info.LiveKeyCount != totalKeys {
			b.Fatalf("expected %d keys, got %d", totalKeys, info.LiveKeyCount)
		}
		b.Logf("Open & Recovery completed: %d transactions, %d keys, measured duration: %d ns (%.3f ms / %.4f s)",
			info.RecoveredTransactions, info.LiveKeyCount, info.RecoveryDurationNs,
			float64(info.RecoveryDurationNs)/1e6, float64(info.RecoveryDurationNs)/1e9)
		openedDB.Close()
	}
}
