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

	"github.com/madhuharshag/Team-Innovexa/src/kv"
)

// SubprocessWorker manages communication with an active database child process.
type SubprocessWorker struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	scanner *bufio.Scanner
	dbPath  string
}

// StartWorker spawns a new independent database process using the current executable.
func StartWorker(dbPath string, maxFileBytes uint64) (*SubprocessWorker, error) {
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
	openCmd := fmt.Sprintf("OPEN %s", dbPath)
	if maxFileBytes > 0 {
		openCmd = fmt.Sprintf("OPEN %s %d", dbPath, maxFileBytes)
	}
	if err := worker.SendCommand(openCmd); err != nil {
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

	mode := r.config.Mode
	if mode == "" {
		mode = "random"
	}

	report := &HarnessReport{
		Mode:       mode,
		Seed:       r.config.Seed,
		GoVersion:  runtime.Version(),
		OSArch:     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		BinaryPath: exe,
	}

	switch mode {
	case "disk_full":
		return r.runDiskFullMode(report, startTime)
	case "compaction_crash":
		return r.runCompactionCrashMode(report, startTime)
	case "recovery_crash":
		return r.runRecoveryCrashMode(report, startTime)
	default:
		// "random", "large_values", "large_batches"
		return r.runStandardCrashMode(report, startTime, mode)
	}
}

// runStandardCrashMode executes random, large_values, or large_batches crash campaigns.
func (r *HarnessRunner) runStandardCrashMode(report *HarnessReport, startTime time.Time, mode string) (*HarnessReport, error) {
	worker, err := StartWorker(r.config.DbPath, r.config.MaxFileBytes)
	if err != nil {
		return nil, fmt.Errorf("initial worker start failed: %w", err)
	}

	crashesTarget := r.config.Crashes
	if crashesTarget <= 0 {
		crashesTarget = 1
	}

	opsPerBurst := r.config.TotalOps / (crashesTarget + 1)
	if opsPerBurst < 1 {
		opsPerBurst = 2
	}

	var opSeq uint64 = 0

	for cycle := 0; cycle < crashesTarget; cycle++ {
		// Pick whether this crash cycle terminates an in-flight operation (50% probability)
		inFlightKillCycle := (r.rng.Intn(2) == 0)
		killAtOpIndex := opsPerBurst - 1
		if killAtOpIndex < 0 {
			killAtOpIndex = 0
		}

		for i := 0; i < opsPerBurst; i++ {
			opSeq++
			opID := atomic.AddUint64(&r.opsAttempted, 1)
			writerID := r.rng.Intn(r.config.Writers)
			if writerID < 0 {
				writerID = 0
			}

			isInFlightKillOp := (inFlightKillCycle && i == killAtOpIndex)

			// Determine operation kind based on mode
			isBatch := false
			isLargeVal := (mode == "large_values")

			if mode == "large_batches" {
				isBatch = true
			} else if mode == "random" {
				dice := r.rng.Intn(100)
				if dice >= 85 {
					isBatch = true
				}
			}

			if isBatch {
				// EXECUTE ATOMIC BATCH
				batchSize := 2 + r.rng.Intn(10)
				if mode == "large_batches" {
					batchSize = 20 + r.rng.Intn(80) // 20 to 100 ops per batch
				}

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

				if isInFlightKillOp {
					// IN-FLIGHT KILL: kill worker mid-batch execution
					_ = worker.Kill()
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
				// EXECUTE PUT OR DELETE
				dice := r.rng.Intn(100)
				if dice < 85 || isLargeVal {
					// PUT
					k := fmt.Sprintf("w%d/key_%06d", writerID, r.rng.Intn(500))
					vLen := 16 + r.rng.Intn(256)
					if isLargeVal {
						// 256 KB to 1 MB values
						vLen = 256*1024 + r.rng.Intn(768*1024)
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

					if isInFlightKillOp {
						// IN-FLIGHT KILL: terminate worker while write is executing
						_ = worker.Kill()
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

					if isInFlightKillOp {
						_ = worker.Kill()
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
				}
			}
		}

		// FORCED CRASH
		_ = worker.Kill()
		r.crashes++

		// RECOVERY & INDEPENDENT ORACLE VERIFICATION
		if err := r.oracle.Verify(r.config.DbPath); err != nil {
			report.Passed = false
			report.VerificationMsg = fmt.Sprintf("oracle verification failed after crash cycle %d: %v", cycle, err)
			return report, fmt.Errorf("%s", report.VerificationMsg)
		}
		r.recoveries++

		// Restart worker for next cycle
		worker, err = StartWorker(r.config.DbPath, r.config.MaxFileBytes)
		if err != nil {
			report.Passed = false
			report.VerificationMsg = fmt.Sprintf("failed to restart worker after crash cycle %d: %v", cycle, err)
			return report, fmt.Errorf("%s", report.VerificationMsg)
		}
	}

	_ = worker.Close()

	// Final verification
	if err := r.oracle.Verify(r.config.DbPath); err != nil {
		report.Passed = false
		report.VerificationMsg = fmt.Sprintf("final oracle verification failed: %v", err)
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
	report.VerificationMsg = fmt.Sprintf("Mode %s verified successfully. All acknowledged mutations survived. All-or-Nothing atomicity preserved.", mode)

	return report, nil
}

// runCompactionCrashMode tests forced termination during active compaction.
func (r *HarnessRunner) runCompactionCrashMode(report *HarnessReport, startTime time.Time) (*HarnessReport, error) {
	crashes := r.config.Crashes
	if crashes <= 0 {
		crashes = 5
	}

	for c := 0; c < crashes; c++ {
		worker, err := StartWorker(r.config.DbPath, r.config.MaxFileBytes)
		if err != nil {
			return nil, fmt.Errorf("compaction worker start failed: %w", err)
		}

		// Insert 50 keys, delete 20 keys
		for i := 0; i < 50; i++ {
			opID := atomic.AddUint64(&r.opsAttempted, 1)
			k := fmt.Sprintf("c%d/key_%04d", c, i)
			v := fmt.Sprintf("c_val_%d_%d", c, i)
			rec := OpRecord{OpID: opID, Kind: OpPut, Key: k, Value: []byte(v)}
			_ = worker.SendCommand(fmt.Sprintf("PUT %d %s %s", opID,
				base64.StdEncoding.EncodeToString([]byte(k)),
				base64.StdEncoding.EncodeToString([]byte(v))))
			_, _ = worker.ReadResponse()
			_ = r.oracle.RecordAck(rec)
			atomic.AddUint64(&r.opsAcknowledged, 1)
		}
		for i := 0; i < 20; i++ {
			opID := atomic.AddUint64(&r.opsAttempted, 1)
			k := fmt.Sprintf("c%d/key_%04d", c, i)
			rec := OpRecord{OpID: opID, Kind: OpDelete, Key: k}
			_ = worker.SendCommand(fmt.Sprintf("DEL %d %s", opID,
				base64.StdEncoding.EncodeToString([]byte(k))))
			_, _ = worker.ReadResponse()
			_ = r.oracle.RecordAck(rec)
			atomic.AddUint64(&r.opsAcknowledged, 1)
		}

		// Trigger compaction and kill in-flight
		_ = worker.SendCommand(fmt.Sprintf("COMPACT %d", atomic.AddUint64(&r.opsAttempted, 1)))
		time.Sleep(time.Duration(r.rng.Intn(1000)) * time.Microsecond)
		_ = worker.Kill()
		r.crashes++

		// Verify database
		if err := r.oracle.Verify(r.config.DbPath); err != nil {
			report.Passed = false
			report.VerificationMsg = fmt.Sprintf("compaction crash verification failed at cycle %d: %v", c, err)
			return report, fmt.Errorf("%s", report.VerificationMsg)
		}
		r.recoveries++
	}

	report.Duration = time.Since(startTime)
	report.OpsAttempted = atomic.LoadUint64(&r.opsAttempted)
	report.OpsAcknowledged = atomic.LoadUint64(&r.opsAcknowledged)
	report.Crashes = r.crashes
	report.Recoveries = r.recoveries
	report.Passed = true
	report.VerificationMsg = "Compaction crash safety verified: authoritative DB uncorrupted, tmp files discarded."
	return report, nil
}

// runRecoveryCrashMode tests repeated process termination during recovery.
func (r *HarnessRunner) runRecoveryCrashMode(report *HarnessReport, startTime time.Time) (*HarnessReport, error) {
	// First populate database
	worker, err := StartWorker(r.config.DbPath, r.config.MaxFileBytes)
	if err != nil {
		return nil, fmt.Errorf("recovery populate start failed: %w", err)
	}

	for i := 0; i < 50; i++ {
		opID := atomic.AddUint64(&r.opsAttempted, 1)
		k := fmt.Sprintf("rec/key_%04d", i)
		v := fmt.Sprintf("rec_val_%04d", i)
		rec := OpRecord{OpID: opID, Kind: OpPut, Key: k, Value: []byte(v)}
		_ = worker.SendCommand(fmt.Sprintf("PUT %d %s %s", opID,
			base64.StdEncoding.EncodeToString([]byte(k)),
			base64.StdEncoding.EncodeToString([]byte(v))))
		_, _ = worker.ReadResponse()
		_ = r.oracle.RecordAck(rec)
		atomic.AddUint64(&r.opsAcknowledged, 1)
	}
	_ = worker.Close()

	crashes := r.config.Crashes
	if crashes <= 0 {
		crashes = 5
	}

	// Repeatedly start worker (which executes recovery) and kill it immediately
	for c := 0; c < crashes; c++ {
		exe, _ := os.Executable()
		var cmd *exec.Cmd
		if strings.Contains(strings.ToLower(exe), ".test") {
			cmd = exec.Command(exe, "-test.run=^TestWorkerProcess$")
		} else {
			cmd = exec.Command(exe, "--worker")
		}
		cmd.Env = append(os.Environ(), "TORTURE_WORKER=1")

		stdin, _ := cmd.StdinPipe()
		_ = cmd.Start()

		// Send OPEN command
		_, _ = fmt.Fprintln(stdin, fmt.Sprintf("OPEN %s", r.config.DbPath))
		// Immediately kill during recovery
		time.Sleep(time.Duration(r.rng.Intn(500)) * time.Microsecond)
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		r.crashes++
	}

	// Now open cleanly and verify idempotence
	if err := r.oracle.Verify(r.config.DbPath); err != nil {
		report.Passed = false
		report.VerificationMsg = fmt.Sprintf("recovery idempotence verification failed: %v", err)
		return report, fmt.Errorf("%s", report.VerificationMsg)
	}
	r.recoveries++

	report.Duration = time.Since(startTime)
	report.OpsAttempted = atomic.LoadUint64(&r.opsAttempted)
	report.OpsAcknowledged = atomic.LoadUint64(&r.opsAcknowledged)
	report.Crashes = r.crashes
	report.Recoveries = r.recoveries
	report.Passed = true
	report.VerificationMsg = "Recovery crash idempotence verified: repeated interrupted recovery does not corrupt DB."
	return report, nil
}

// runDiskFullMode tests database behavior when max capacity is reached.
func (r *HarnessRunner) runDiskFullMode(report *HarnessReport, startTime time.Time) (*HarnessReport, error) {
	// Initialize with 40 KB capacity limit
	maxBytes := uint64(40 * 1024)
	worker, err := StartWorker(r.config.DbPath, maxBytes)
	if err != nil {
		return nil, fmt.Errorf("disk full worker start failed: %w", err)
	}

	reachedFull := false
	for i := 0; i < 500; i++ {
		opID := atomic.AddUint64(&r.opsAttempted, 1)
		k := fmt.Sprintf("df/key_%04d", i)
		v := strings.Repeat("D", 200) // ~280 bytes per transaction
		rec := OpRecord{OpID: opID, Kind: OpPut, Key: k, Value: []byte(v)}

		_ = worker.SendCommand(fmt.Sprintf("PUT %d %s %s", opID,
			base64.StdEncoding.EncodeToString([]byte(k)),
			base64.StdEncoding.EncodeToString([]byte(v))))

		resp, err := worker.ReadResponse()
		if err != nil {
			break
		}

		if strings.HasPrefix(resp, "ACK") {
			_ = r.oracle.RecordAck(rec)
			atomic.AddUint64(&r.opsAcknowledged, 1)
		} else {
			// Expected when capacity full: "ERR <id> ... maximum database capacity exceeded"
			r.oracle.RecordRejected(rec, resp)
			atomic.AddUint64(&r.opsRejected, 1)
			if strings.Contains(resp, "maximum database capacity exceeded") {
				reachedFull = true
				break
			}
		}
	}

	_ = worker.Close()

	if !reachedFull {
		report.Passed = false
		report.VerificationMsg = "disk_full mode failed: database never encountered ErrFull capacity limit"
		return report, fmt.Errorf("%s", report.VerificationMsg)
	}

	// Verify that all data acknowledged before ErrFull survived cleanly
	if err := r.oracle.Verify(r.config.DbPath); err != nil {
		report.Passed = false
		report.VerificationMsg = fmt.Sprintf("disk_full verification failed: %v", err)
		return report, fmt.Errorf("%s", report.VerificationMsg)
	}

	// Verify that direct Open also recovers successfully
	db, _, err := kv.Open(r.config.DbPath, kv.Options{})
	if err != nil {
		report.Passed = false
		report.VerificationMsg = fmt.Sprintf("failed to reopen full database: %v", err)
		return report, fmt.Errorf("%s", report.VerificationMsg)
	}
	defer db.Close()

	report.Duration = time.Since(startTime)
	report.OpsAttempted = atomic.LoadUint64(&r.opsAttempted)
	report.OpsAcknowledged = atomic.LoadUint64(&r.opsAcknowledged)
	report.OpsRejected = atomic.LoadUint64(&r.opsRejected)
	report.Passed = true
	report.VerificationMsg = "Disk full capacity enforcement verified: ErrFull correctly returned, zero corruption."
	return report, nil
}
