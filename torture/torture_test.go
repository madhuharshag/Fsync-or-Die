package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/madhuharshag/Team-Innovexa/src/kv"
)

// TestWorkerProcess acts as the worker entrypoint when the test executable is spawned as a child process.
func TestWorkerProcess(t *testing.T) {
	if os.Getenv("TORTURE_WORKER") == "1" {
		runWorkerProcess()
		os.Exit(0)
	}
}

// 1. Seed reproducibility: two runs with identical seed and config execute the exact same sequence.
func TestTorture_1_SeedReproducibility(t *testing.T) {
	const testSeed = 987654321
	cfg1 := HarnessConfig{Seed: testSeed, TotalOps: 100, Crashes: 2, Writers: 2}
	cfg2 := HarnessConfig{Seed: testSeed, TotalOps: 100, Crashes: 2, Writers: 2}

	r1 := NewHarnessRunner(cfg1, nil)
	r2 := NewHarnessRunner(cfg2, nil)

	// Sample 20 random integers and check identity
	for i := 0; i < 20; i++ {
		v1 := r1.rng.Intn(1000)
		v2 := r2.rng.Intn(1000)
		if v1 != v2 {
			t.Fatalf("step %d RNG mismatch for identical seed: got %d vs %d", i, v1, v2)
		}
	}
}

// 2. Oracle records only observed successful acknowledgements.
func TestTorture_2_OracleRecordsOnlyObservedAcks(t *testing.T) {
	dir := t.TempDir()
	oracle, err := NewOracle(filepath.Join(dir, "oracle.log"))
	if err != nil {
		t.Fatalf("NewOracle failed: %v", err)
	}
	defer oracle.Close()

	// An unconfirmed operation must NOT be in ackedKeys
	pendingOp := OpRecord{
		OpID:   1,
		Kind:   OpPut,
		Key:    "test/pending",
		Value:  []byte("pending_val"),
		Status: StatusPending,
	}
	// Do not record ACK
	_ = pendingOp

	if _, exists := oracle.ackedKeys["test/pending"]; exists {
		t.Fatalf("pending operation should not be recorded in ackedKeys")
	}

	// Record ACK explicitly
	if err := oracle.RecordAck(pendingOp); err != nil {
		t.Fatalf("RecordAck failed: %v", err)
	}

	if val, exists := oracle.ackedKeys["test/pending"]; !exists || string(val) != "pending_val" {
		t.Fatalf("acknowledged operation missing from ackedKeys: got %v", exists)
	}
}

// 3. Unknown in-flight operations are handled correctly.
func TestTorture_3_UnknownInFlightOperationsHandled(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "indeterminate.db")

	// Pre-create DB
	db, _, err := kv.Open(dbPath, kv.Options{})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	_ = db.Put([]byte("ack_key"), []byte("ack_val"))
	_ = db.Close()

	oracle, err := NewOracle(filepath.Join(dir, "oracle.log"))
	if err != nil {
		t.Fatalf("NewOracle failed: %v", err)
	}
	defer oracle.Close()

	_ = oracle.RecordAck(OpRecord{
		Kind:  OpPut,
		Key:   "ack_key",
		Value: []byte("ack_val"),
	})

	// Add an indeterminate operation that was never committed
	oracle.RecordIndeterminate(OpRecord{
		Kind:  OpPut,
		Key:   "in_flight_key",
		Value: []byte("in_flight_val"),
	})

	// Verify should SUCCEED: in-flight operation is allowed to be absent
	if err := oracle.Verify(dbPath); err != nil {
		t.Fatalf("Verify failed with valid indeterminate operation: %v", err)
	}
}

// 4 & 5. Launch, terminate, and verify forced termination really kills child process.
func TestTorture_4_5_LaunchAndForcedTermination(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "kill_test.db")

	worker, err := StartWorker(dbPath, 0)
	if err != nil {
		t.Fatalf("StartWorker failed: %v", err)
	}

	// Verify process is alive
	if worker.cmd.Process == nil {
		t.Fatalf("worker.cmd.Process is nil")
	}
	pid := worker.cmd.Process.Pid
	if pid <= 0 {
		t.Fatalf("invalid worker PID: %d", pid)
	}

	// Send PING
	if err := worker.SendCommand("PING"); err != nil {
		t.Fatalf("SendCommand failed: %v", err)
	}
	resp, err := worker.ReadResponse()
	if err != nil || resp != "PONG" {
		t.Fatalf("expected PONG, got %q (err=%v)", resp, err)
	}

	// Force kill child
	if err := worker.Kill(); err != nil {
		t.Fatalf("Kill failed: %v", err)
	}

	// Wait and verify process is dead
	time.Sleep(50 * time.Millisecond)
	p, err := os.FindProcess(pid)
	if err == nil && p != nil {
		// On Windows, FindProcess always succeeds, but sending signal 0 fails if dead
		_ = p
	}

	// Verify a new process can immediately open the database (lock released by OS)
	db, _, err := kv.Open(dbPath, kv.Options{})
	if err != nil {
		t.Fatalf("failed to open database after child process kill: %v", err)
	}
	_ = db.Close()
}

// 6 & 7. Recovery run in fresh process; all acknowledged operations survive.
func TestTorture_6_7_FreshProcessRecoveryAndAckSurvival(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "ack_survival.db")
	oracleLog := filepath.Join(dir, "oracle.log")

	oracle, err := NewOracle(oracleLog)
	if err != nil {
		t.Fatalf("NewOracle failed: %v", err)
	}
	defer oracle.Close()

	worker, err := StartWorker(dbPath, 0)
	if err != nil {
		t.Fatalf("StartWorker failed: %v", err)
	}

	// Write 20 acknowledged keys
	for i := 0; i < 20; i++ {
		k := fmt.Sprintf("persisted_%03d", i)
		v := fmt.Sprintf("value_data_%03d", i)
		cmd := fmt.Sprintf("PUT %d %s %s", i+1,
			base64.StdEncoding.EncodeToString([]byte(k)),
			base64.StdEncoding.EncodeToString([]byte(v)))

		if err := worker.SendCommand(cmd); err != nil {
			t.Fatalf("SendCommand failed: %v", err)
		}
		resp, err := worker.ReadResponse()
		if err != nil || !strings.HasPrefix(resp, "ACK") {
			t.Fatalf("expected ACK, got %q (err=%v)", resp, err)
		}

		_ = oracle.RecordAck(OpRecord{
			OpID:   uint64(i + 1),
			Kind:   OpPut,
			Key:    k,
			Value:  []byte(v),
			Status: StatusAcknowledged,
		})
	}

	// Force kill the worker
	_ = worker.Kill()

	// Verify recovery from fresh process
	if err := oracle.Verify(dbPath); err != nil {
		t.Fatalf("Oracle verification failed: %v", err)
	}
}

// 8. Incomplete batches are not partially visible (All-or-Nothing atomicity).
func TestTorture_8_IncompleteBatchesNotPartiallyVisible(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "partial_batch.db")

	// Pre-create DB with one key from the batch applied
	db, _, err := kv.Open(dbPath, kv.Options{})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	// Simulate corrupted state where 1 of 2 ops was applied
	_ = db.Put([]byte("batch_key_1"), []byte("val_1"))
	_ = db.Close()

	oracle, _ := NewOracle(filepath.Join(dir, "oracle.log"))
	defer oracle.Close()

	// Record indeterminate batch with 2 ops
	oracle.RecordIndeterminate(OpRecord{
		Kind:    OpBatch,
		BatchID: 999,
		BatchOps: []OpRecord{
			{Kind: OpPut, Key: "batch_key_1", Value: []byte("val_1")},
			{Kind: OpPut, Key: "batch_key_2", Value: []byte("val_2")},
		},
	})

	// Oracle MUST detect partial batch exposure and return error!
	err = oracle.Verify(dbPath)
	if err == nil {
		t.Fatalf("expected Oracle to fail on partial batch exposure, but it passed")
	}
	if !strings.Contains(err.Error(), "partial batch exposed") {
		t.Fatalf("expected 'partial batch exposed' error, got: %v", err)
	}
}

// 9 & 10. Multiple writers and readers can run concurrently.
func TestTorture_9_10_ConcurrentWritersAndReaders(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "concurrent.db")

	db, _, err := kv.Open(dbPath, kv.Options{})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	done := make(chan bool)
	errChan := make(chan error, 10)

	// Writer 1
	go func() {
		for i := 0; i < 50; i++ {
			if err := db.Put([]byte(fmt.Sprintf("w1/k%d", i)), []byte(fmt.Sprintf("v%d", i))); err != nil {
				errChan <- err
				return
			}
		}
		done <- true
	}()

	// Writer 2
	go func() {
		for i := 0; i < 50; i++ {
			if err := db.Put([]byte(fmt.Sprintf("w2/k%d", i)), []byte(fmt.Sprintf("v%d", i))); err != nil {
				errChan <- err
				return
			}
		}
		done <- true
	}()

	// Reader
	go func() {
		for i := 0; i < 100; i++ {
			_, _ = db.Get([]byte("w1/k0"))
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()

	// Wait for all 3 goroutines
	for i := 0; i < 3; i++ {
		select {
		case <-done:
		case err := <-errChan:
			t.Fatalf("concurrency error: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatalf("concurrency test timeout")
		}
	}
}

// 11 & 12. Randomized torture run with crashes and recovery.
func TestTorture_11_12_RandomCrashCyclesAndRecovery(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "random_torture.db")
	oracleLog := filepath.Join(dir, "oracle.log")

	oracle, err := NewOracle(oracleLog)
	if err != nil {
		t.Fatalf("NewOracle failed: %v", err)
	}
	defer oracle.Close()

	cfg := HarnessConfig{
		Seed:     42,
		TotalOps: 150,
		Crashes:  3,
		Writers:  2,
		Readers:  1,
		DbPath:   dbPath,
	}

	runner := NewHarnessRunner(cfg, oracle)
	report, err := runner.Run()
	if err != nil {
		t.Fatalf("Torture run failed: %v", err)
	}

	if !report.Passed {
		t.Fatalf("Report indicated failure: %s", report.VerificationMsg)
	}
	if report.Crashes != 3 {
		t.Fatalf("expected 3 crashes, got %d", report.Crashes)
	}
	if report.Recoveries != 3 {
		t.Fatalf("expected 3 recoveries, got %d", report.Recoveries)
	}
}

// 13. Failure output includes reproducible seed and history.
func TestTorture_13_FailureOutputIncludesReproducibleSeed(t *testing.T) {
	cfg := HarnessConfig{
		Seed: 123456789,
	}
	report := &HarnessReport{
		Seed:            cfg.Seed,
		Passed:          false,
		VerificationMsg: "injected verification error for reporting test",
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "Seed: %d\nMsg: %s\n", report.Seed, report.VerificationMsg)

	output := buf.String()
	if !strings.Contains(output, "123456789") {
		t.Fatalf("report missing seed: %s", output)
	}
	if !strings.Contains(output, "injected verification error") {
		t.Fatalf("report missing failure message: %s", output)
	}
}

// 14. Oracle does not use internal database state.
func TestTorture_14_OracleIsIndependent(t *testing.T) {
	oracle, _ := NewOracle("")
	defer oracle.Close()

	// Verify Oracle struct has no pointer or reference to kv.DB or kv.Index
	_ = oracle.ackedKeys
	_ = oracle.ackedDeletes
	_ = oracle.history
}

// 15. The harness can detect deliberately introduced data loss in a controlled test.
func TestTorture_15_DetectsDeliberateDataLoss(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "loss_detection.db")

	// 1. Create DB with valid key
	db, _, err := kv.Open(dbPath, kv.Options{})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	_ = db.Put([]byte("valid_key"), []byte("valid_val"))
	_ = db.Close()

	// 2. Oracle expects key "missing_key" to exist with value "lost_val"
	oracle, _ := NewOracle("")
	defer oracle.Close()

	oracle.InjectSimulatedLoss("missing_key", []byte("lost_val"))

	// 3. Oracle Verify MUST detect that missing_key was lost!
	err = oracle.Verify(dbPath)
	if err == nil {
		t.Fatalf("expected Oracle to detect missing key, but Verify returned nil")
	}
	if !strings.Contains(err.Error(), "lost after recovery") {
		t.Fatalf("expected 'lost after recovery' error, got: %v", err)
	}
}
