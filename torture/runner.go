package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

// SubprocessWorker manages communication with an active database child process.
type SubprocessWorker struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	scanner *bufio.Scanner
	dbPath  string
}

// StartWorker spawns a new independent database process using the current executable.
func StartWorker(dbPath string) (*SubprocessWorker, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	var cmd *exec.Cmd
	if strings.Contains(strings.ToLower(exe), ".test") {
		cmd = exec.Command(exe, "-test.run=^TestWorkerProcess$")
	} else {
		cmd = exec.Command(exe, "--worker")
	}
	cmd.Env = append(os.Environ(), "TORTURE_WORKER=1")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start worker process: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	worker := &SubprocessWorker{
		cmd:     cmd,
		stdin:   stdin,
		scanner: scanner,
		dbPath:  dbPath,
	}

	// Send OPEN command
	if err := worker.SendCommand(fmt.Sprintf("OPEN %s", dbPath)); err != nil {
		_ = worker.Kill()
		return nil, err
	}

	resp, err := worker.ReadResponse()
	if err != nil || resp != "READY" {
		_ = worker.Kill()
		return nil, fmt.Errorf("worker failed to initialize: resp=%q, err=%v", resp, err)
	}

	return worker, nil
}

// SendCommand writes a command line to the worker's standard input.
func (w *SubprocessWorker) SendCommand(cmd string) error {
	_, err := fmt.Fprintln(w.stdin, cmd)
	return err
}

// ReadResponse reads the next response line from the worker's standard output.
func (w *SubprocessWorker) ReadResponse() (string, error) {
	if w.scanner.Scan() {
		return strings.TrimSpace(w.scanner.Text()), nil
	}
	if err := w.scanner.Err(); err != nil {
		return "", err
	}
	return "", io.EOF
}

// Kill forces immediate non-graceful process termination (TerminateProcess / SIGKILL).
func (w *SubprocessWorker) Kill() error {
	if w.cmd != nil && w.cmd.Process != nil {
		_ = w.cmd.Process.Kill()
		_ = w.cmd.Wait()
	}
	return nil
}

// Close gracefully closes the database and waits for the worker process to exit.
func (w *SubprocessWorker) Close() error {
	_ = w.SendCommand("CLOSE 0")
	_, _ = w.ReadResponse()
	if w.stdin != nil {
		_ = w.stdin.Close()
	}
	if w.cmd != nil {
		_ = w.cmd.Wait()
	}
	return nil
}

// HarnessRunner coordinates workloads, crash schedules, and verification.
type HarnessRunner struct {
	config HarnessConfig
	oracle *Oracle
	rng    *rand.Rand

	opsAttempted     uint64
	opsAcknowledged  uint64
	opsIndeterminate uint64
	opsRejected      uint64
	crashes          int
	recoveries       int
}

// NewHarnessRunner constructs a torture harness runner.
func NewHarnessRunner(cfg HarnessConfig, oracle *Oracle) *HarnessRunner {
	return &HarnessRunner{
		config: cfg,
		oracle: oracle,
		rng:    rand.New(rand.NewSource(cfg.Seed)),
	}
}

// Run executes the configured torture test lifecycle.
func (r *HarnessRunner) Run() (*HarnessReport, error) {
	startTime := time.Now()
	exe, _ := os.Executable()

	report := &HarnessReport{
		Seed:       r.config.Seed,
		GoVersion:  runtime.Version(),
		OSArch:     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		BinaryPath: exe,
	}

	worker, err := StartWorker(r.config.DbPath)
	if err != nil {
		return nil, fmt.Errorf("initial worker start failed: %w", err)
	}

	opsPerCrash := r.config.TotalOps / (r.config.Crashes + 1)
	if opsPerCrash < 1 {
		opsPerCrash = 10
	}

	var opSeq uint64 = 0

	for cycle := 0; cycle < r.config.Crashes; cycle++ {
		// Run a burst of operations
		for i := 0; i < opsPerCrash; i++ {
			opSeq++
			opID := atomic.AddUint64(&r.opsAttempted, 1)

			// Select operation kind
			dice := r.rng.Intn(100)
			writerID := r.rng.Intn(r.config.Writers)

			if dice < 60 {
				// PUT
				k := fmt.Sprintf("w%d/key_%06d", writerID, r.rng.Intn(500))
				vLen := 16 + r.rng.Intn(256)
				if r.rng.Intn(100) == 0 {
					// 1% of the time, write a large value (500 KB)
					vLen = 500 * 1024
				}
				v := make([]byte, vLen)
				r.rng.Read(v)

				rec := OpRecord{
					OpID:       opID,
					Seed:       r.config.Seed,
					Kind:       OpPut,
					Key:        k,
					Value:      v,
					ValLen:     len(v),
					Status:     StatusPending,
					WriterID:   writerID,
					SeqNum:     opSeq,
					CrashCycle: cycle,
					Timestamp:  time.Now(),
				}

				encK := base64.StdEncoding.EncodeToString([]byte(k))
				encV := base64.StdEncoding.EncodeToString(v)
				cmd := fmt.Sprintf("PUT %d %s %s", opID, encK, encV)

				if err := worker.SendCommand(cmd); err != nil {
					r.oracle.RecordIndeterminate(rec)
					atomic.AddUint64(&r.opsIndeterminate, 1)
					break
				}

				resp, err := worker.ReadResponse()
				if err != nil {
					// Interrupted in flight
					r.oracle.RecordIndeterminate(rec)
					atomic.AddUint64(&r.opsIndeterminate, 1)
					break
				}

				if strings.HasPrefix(resp, "ACK") {
					_ = r.oracle.RecordAck(rec)
					atomic.AddUint64(&r.opsAcknowledged, 1)
				} else {
					r.oracle.RecordRejected(rec, resp)
					atomic.AddUint64(&r.opsRejected, 1)
				}

			} else if dice < 80 {
				// GET
				k := fmt.Sprintf("w%d/key_%06d", writerID, r.rng.Intn(500))
				encK := base64.StdEncoding.EncodeToString([]byte(k))
				_ = worker.SendCommand(fmt.Sprintf("GET %d %s", opID, encK))
				_, _ = worker.ReadResponse()

			} else if dice < 90 {
				// DELETE
				k := fmt.Sprintf("w%d/key_%06d", writerID, r.rng.Intn(500))
				rec := OpRecord{
					OpID:       opID,
					Seed:       r.config.Seed,
					Kind:       OpDelete,
					Key:        k,
					Status:     StatusPending,
					WriterID:   writerID,
					SeqNum:     opSeq,
					CrashCycle: cycle,
					Timestamp:  time.Now(),
				}

				encK := base64.StdEncoding.EncodeToString([]byte(k))
				if err := worker.SendCommand(fmt.Sprintf("DEL %d %s", opID, encK)); err != nil {
					r.oracle.RecordIndeterminate(rec)
					atomic.AddUint64(&r.opsIndeterminate, 1)
					break
				}

				resp, err := worker.ReadResponse()
				if err != nil {
					r.oracle.RecordIndeterminate(rec)
					atomic.AddUint64(&r.opsIndeterminate, 1)
					break
				}

				if strings.HasPrefix(resp, "ACK") {
					_ = r.oracle.RecordAck(rec)
					atomic.AddUint64(&r.opsAcknowledged, 1)
				} else {
					r.oracle.RecordRejected(rec, resp)
					atomic.AddUint64(&r.opsRejected, 1)
				}

			} else if dice < 95 {
				// BATCH
				batchSize := 2 + r.rng.Intn(10)
				type WireOp struct {
					Kind  uint8  `json:"kind"`
					Key   string `json:"key"`
					Value string `json:"val"`
				}
				var wireOps []WireOp
				var batchRecs []OpRecord

				for b := 0; b < batchSize; b++ {
					bk := fmt.Sprintf("w%d/batch_%06d_%d", writerID, r.rng.Intn(200), b)
					bv := fmt.Sprintf("batch_val_%d_%d", opID, b)
					wireOps = append(wireOps, WireOp{Kind: 1, Key: bk, Value: bv})
					batchRecs = append(batchRecs, OpRecord{
						Kind:     OpPut,
						Key:      bk,
						Value:    []byte(bv),
						ValLen:   len(bv),
						BatchID:  opID,
						WriterID: writerID,
					})
				}

				rawJson, _ := json.Marshal(wireOps)
				encJson := base64.StdEncoding.EncodeToString(rawJson)

				rec := OpRecord{
					OpID:       opID,
					Seed:       r.config.Seed,
					Kind:       OpBatch,
					BatchID:    opID,
					BatchOps:   batchRecs,
					Status:     StatusPending,
					WriterID:   writerID,
					SeqNum:     opSeq,
					CrashCycle: cycle,
					Timestamp:  time.Now(),
				}

				if err := worker.SendCommand(fmt.Sprintf("BATCH %d %s", opID, encJson)); err != nil {
					r.oracle.RecordIndeterminate(rec)
					atomic.AddUint64(&r.opsIndeterminate, 1)
					break
				}

				resp, err := worker.ReadResponse()
				if err != nil {
					r.oracle.RecordIndeterminate(rec)
					atomic.AddUint64(&r.opsIndeterminate, 1)
					break
				}

				if strings.HasPrefix(resp, "ACK") {
					_ = r.oracle.RecordAck(rec)
					atomic.AddUint64(&r.opsAcknowledged, 1)
				} else {
					r.oracle.RecordRejected(rec, resp)
					atomic.AddUint64(&r.opsRejected, 1)
				}

			} else {
				// COMPACT
				_ = worker.SendCommand(fmt.Sprintf("COMPACT %d", opID))
				_, _ = worker.ReadResponse()
			}
		}

		// DELIBERATE FORCED PROCESS CRASH
		_ = worker.Kill()
		r.crashes++

		// RECOVERY & INDEPENDENT ORACLE VERIFICATION
		if err := r.oracle.Verify(r.config.DbPath); err != nil {
			report.Passed = false
			report.VerificationMsg = fmt.Sprintf("verification failed after crash cycle %d: %v", cycle, err)
			return report, fmt.Errorf("%s", report.VerificationMsg)
		}
		r.recoveries++

		// Start fresh worker process for next cycle
		worker, err = StartWorker(r.config.DbPath)
		if err != nil {
			report.Passed = false
			report.VerificationMsg = fmt.Sprintf("failed to restart worker after crash cycle %d: %v", cycle, err)
			return report, fmt.Errorf("%s", report.VerificationMsg)
		}
	}

	// Clean shutdown of final worker
	_ = worker.Close()

	// Final verification
	if err := r.oracle.Verify(r.config.DbPath); err != nil {
		report.Passed = false
		report.VerificationMsg = fmt.Sprintf("final verification failed: %v", err)
		return report, fmt.Errorf("%s", report.VerificationMsg)
	}

	report.Duration = time.Since(startTime)
	report.OpsAttempted = atomic.LoadUint64(&r.opsAttempted)
	report.OpsAcknowledged = atomic.LoadUint64(&r.opsAcknowledged)
	report.OpsIndeterminate = atomic.LoadUint64(&r.opsIndeterminate)
	report.OpsRejected = atomic.LoadUint64(&r.opsRejected)
	report.Crashes = r.crashes
	report.Recoveries = r.recoveries
	report.Passed = true
	report.VerificationMsg = "All acknowledged mutations survived. Zero partial batches. Full integrity confirmed."

	return report, nil
}
