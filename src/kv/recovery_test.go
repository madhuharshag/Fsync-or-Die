package kv

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestRecovery_TornTxnHeader(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "torn_header.db")

	engine, _, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	for i := 0; i < 5; i++ {
		_ = engine.Put([]byte(fmt.Sprintf("k%d", i)), []byte(fmt.Sprintf("v%d", i)))
	}
	_ = engine.Close()

	// Append 15 bytes of torn header bytes (< 32 bytes)
	f, err := os.OpenFile(dbPath, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	_, _ = f.Write([]byte("TXN1_partial_15"))
	_ = f.Close()

	// Reopen must succeed, truncate the 15 bytes, and recover 5 transactions
	reopened, info, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Reopen failed: %v", err)
	}
	defer reopened.Close()

	if info.RecoveredTransactions != 5 {
		t.Fatalf("Expected 5 recovered txns, got: %d", info.RecoveredTransactions)
	}
	if info.RepairedTailBytes != 15 {
		t.Fatalf("Expected 15 repaired tail bytes, got: %d", info.RepairedTailBytes)
	}

	// Verify all 5 keys are present
	for i := 0; i < 5; i++ {
		val, err := reopened.Get([]byte(fmt.Sprintf("k%d", i)))
		if err != nil || string(val) != fmt.Sprintf("v%d", i) {
			t.Fatalf("Missing key k%d: %v", i, err)
		}
	}

	// Subsequent write must succeed cleanly
	if err := reopened.Put([]byte("new_key"), []byte("new_val")); err != nil {
		t.Fatalf("Subsequent put failed: %v", err)
	}
}

func TestRecovery_TornPayload(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "torn_payload.db")

	engine, _, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	_ = engine.Put([]byte("valid_k"), []byte("valid_v"))
	_ = engine.Close()

	// Append a valid 32B TxnHeader claiming 200B of payload, but only write 50B
	h := TxnHeader{
		Magic:      TxnHeaderMagic,
		Version:    CurrentTxnVersion,
		TxID:       2,
		OpCount:    1,
		PayloadLen: 200,
	}
	hBytes := EncodeTxnHeader(h)

	f, err := os.OpenFile(dbPath, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	_, _ = f.Write(hBytes)
	_, _ = f.Write(make([]byte, 50)) // 50 bytes instead of 200 + 32
	_ = f.Close()

	reopened, info, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Reopen failed: %v", err)
	}
	defer reopened.Close()

	if info.RecoveredTransactions != 1 {
		t.Fatalf("Expected 1 recovered txn, got: %d", info.RecoveredTransactions)
	}
	if info.RepairedTailBytes != 32+50 {
		t.Fatalf("Expected 82 repaired tail bytes, got: %d", info.RepairedTailBytes)
	}

	val, err := reopened.Get([]byte("valid_k"))
	if err != nil || string(val) != "valid_v" {
		t.Fatalf("Expected valid_v, got: %s (err: %v)", val, err)
	}
}

func TestRecovery_TornFooterOrCorruptCRC(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "torn_footer.db")

	engine, _, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	_ = engine.Put([]byte("base_k"), []byte("base_v"))
	_ = engine.Close()

	// Append a complete op, but write an invalid CRC in the footer
	op := Op{Kind: PutOp, Key: []byte("bad_k"), Value: []byte("bad_v")}
	encOp, _ := EncodeOp(op)
	h := TxnHeader{
		Magic:      TxnHeaderMagic,
		Version:    CurrentTxnVersion,
		TxID:       2,
		OpCount:    1,
		PayloadLen: uint64(len(encOp)),
	}
	hBytes := EncodeTxnHeader(h)

	badFooter := CommitFooter{
		Magic:        CommitFooterMagic,
		TxID:         2,
		TotalTxBytes: uint64(32 + len(encOp) + 32),
		TxnCRC:       0xDEADBEEF, // Wrong CRC
		FooterCRC:    0xBAADF00D,
	}
	badFooterBytes := EncodeCommitFooter(badFooter)

	f, err := os.OpenFile(dbPath, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	_, _ = f.Write(hBytes)
	_, _ = f.Write(encOp)
	_, _ = f.Write(badFooterBytes)
	_ = f.Close()

	reopened, info, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Reopen failed: %v", err)
	}
	defer reopened.Close()

	if info.RecoveredTransactions != 1 {
		t.Fatalf("Expected 1 recovered txn, got %d", info.RecoveredTransactions)
	}
	_, err = reopened.Get([]byte("bad_k"))
	if err == nil {
		t.Fatalf("Expected bad_k to be discarded due to bad CRC, but it was found")
	}
}

func TestRecovery_OneByteTruncation_F4(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "one_byte_trunc.db")

	engine, _, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	_ = engine.Put([]byte("first"), []byte("value1"))
	_ = engine.Put([]byte("second"), []byte("value2"))
	_ = engine.Close()

	fi, _ := os.Stat(dbPath)
	originalSize := fi.Size()

	// Truncate the file by 1 byte (Scenario F4)
	if err := os.Truncate(dbPath, originalSize-1); err != nil {
		t.Fatalf("Truncate failed: %v", err)
	}

	reopened, info, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Open failed on 1-byte truncated file: %v", err)
	}
	defer reopened.Close()

	// First key must survive
	val, err := reopened.Get([]byte("first"))
	if err != nil || string(val) != "value1" {
		t.Fatalf("Expected first key to survive: %v", err)
	}
	if info.RecoveredTransactions != 1 {
		t.Fatalf("Expected 1 recovered transaction, got %d", info.RecoveredTransactions)
	}
}

func TestRecovery_Idempotence_F2(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "idempotent.db")

	engine, _, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	_ = engine.Put([]byte("key1"), []byte("val1"))
	_ = engine.Close()

	// Append corrupt trailing bytes
	f, _ := os.OpenFile(dbPath, os.O_WRONLY|os.O_APPEND, 0644)
	_, _ = f.Write([]byte("random_tail_bytes_that_are_incomplete"))
	_ = f.Close()

	// 1st Recovery
	r1, info1, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("First Open failed: %v", err)
	}
	if info1.RepairedTailBytes == 0 {
		t.Fatalf("Expected repaired bytes on 1st open")
	}
	_ = r1.Close()

	// 2nd Recovery: must produce identical state with 0 repaired bytes
	r2, info2, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Second Open failed: %v", err)
	}
	defer r2.Close()

	if info2.RecoveredTransactions != info1.RecoveredTransactions {
		t.Fatalf("Transaction count mismatch between runs: %d vs %d", info1.RecoveredTransactions, info2.RecoveredTransactions)
	}
	if info2.RepairedTailBytes != 0 {
		t.Fatalf("Expected 0 repaired tail bytes on second open, got %d", info2.RepairedTailBytes)
	}

	val, err := r2.Get([]byte("key1"))
	if err != nil || string(val) != "val1" {
		t.Fatalf("Expected val1 on second open: %v", err)
	}
}

func TestRecovery_LargeBatch10000_B6(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "large_batch_10000.db")

	engine, _, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	// 1. Write a complete 10,000-operation batch
	ops := make([]Op, 10000)
	for i := 0; i < 10000; i++ {
		ops[i] = Op{
			Kind:  PutOp,
			Key:   []byte(fmt.Sprintf("k_%05d", i)),
			Value: []byte(fmt.Sprintf("v_%05d", i)),
		}
	}
	if err := engine.Batch(ops); err != nil {
		t.Fatalf("Batch of 10,000 failed: %v", err)
	}
	_ = engine.Close()

	// 2. Reopen and verify all 10,000 are recovered
	reopened, info, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Reopen failed: %v", err)
	}
	defer reopened.Close()

	if info.LiveKeyCount != 10000 {
		t.Fatalf("Expected 10,000 live keys, got: %d", info.LiveKeyCount)
	}
	val, err := reopened.Get([]byte("k_00000"))
	if err != nil || !bytes.Equal(val, []byte("v_00000")) {
		t.Fatalf("Expected v_00000, got: %s (err: %v)", val, err)
	}
	val, err = reopened.Get([]byte("k_09999"))
	if err != nil || !bytes.Equal(val, []byte("v_09999")) {
		t.Fatalf("Expected v_09999, got: %s (err: %v)", val, err)
	}
}
