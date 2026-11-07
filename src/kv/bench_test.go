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
