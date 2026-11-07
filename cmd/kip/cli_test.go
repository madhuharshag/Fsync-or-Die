package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/madhuharshag/Team-Innovexa/src/kv"
)

// TestCLI_HelperProcess acts as the child subprocess entry point for black-box testing.
func TestCLI_HelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_CLI_HELPER_PROCESS") != "1" {
		return
	}
	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}
	code := run(args)
	os.Exit(code)
}

// runCLI executes the CLI in a separate child process, capturing stdout, stderr, and exit code.
// Automatically filters out Go test framework banners (e.g. "=== RUN ...", "--- PASS ...") from stdout.
func runCLI(t *testing.T, stdin string, args ...string) (string, string, int) {
	t.Helper()
	cmdArgs := append([]string{"-test.run=^TestCLI_HelperProcess$", "-test.v", "--"}, args...)
	cmd := exec.Command(os.Args[0], cmdArgs...)
	cmd.Env = append(os.Environ(), "GO_WANT_CLI_HELPER_PROCESS=1")

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("failed to run CLI helper process: %v", err)
		}
	}

	// Filter out testing harness banners from stdout if helper process output them
	cleanStdout := filterTestBanners(stdoutBuf.String())
	return cleanStdout, stderrBuf.String(), exitCode
}

func filterTestBanners(s string) string {
	lines := strings.Split(s, "\n")
	var out []string
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "=== RUN") || strings.HasPrefix(trimmed, "--- PASS") || strings.HasPrefix(trimmed, "PASS") {
			continue
		}
		out = append(out, l)
	}
	res := strings.Join(out, "\n")
	// If original output ended with a newline, ensure res preserves exactly one trailing newline
	if strings.HasSuffix(s, "\n") && !strings.HasSuffix(res, "\n") {
		res += "\n"
	}
	return res
}

func tempDBPath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "test.kip")
}

// --- BASIC COMMANDS TESTS (1 - 17) ---

func TestCLI_HelpAndVersion(t *testing.T) {
	// 1. Help output
	stdout, stderr, code := runCLI(t, "", "help")
	if code != ExitSuccess || !strings.Contains(stdout, "USAGE:") || stderr != "" {
		t.Fatalf("help failed: code=%d, stdout=%s, stderr=%s", code, stdout, stderr)
	}

	stdout, _, code = runCLI(t, "", "--help")
	if code != ExitSuccess || !strings.Contains(stdout, "COMMANDS:") {
		t.Fatalf("--help failed: code=%d", code)
	}

	// Version
	stdout, _, code = runCLI(t, "", "version")
	if code != ExitSuccess || !strings.Contains(stdout, AppVersion) {
		t.Fatalf("version failed: code=%d, stdout=%s", code, stdout)
	}
}

func TestCLI_InvalidAndMissingArguments(t *testing.T) {
	// 2. Invalid command
	_, stderr, code := runCLI(t, "", "nonexistent")
	if code != ExitError || !strings.Contains(stderr, "unknown command") {
		t.Fatalf("expected ExitError for unknown command, got %d, stderr: %s", code, stderr)
	}

	// 3. Missing arguments
	_, stderr, code = runCLI(t, "", "put")
	if code != ExitError || !strings.Contains(stderr, "syntax") {
		t.Fatalf("expected ExitError for missing put args, got %d, stderr: %s", code, stderr)
	}

	_, stderr, code = runCLI(t, "", "get")
	if code != ExitError || !strings.Contains(stderr, "syntax") {
		t.Fatalf("expected ExitError for missing get args, got %d", code)
	}

	_, stderr, code = runCLI(t, "", "del")
	if code != ExitError || !strings.Contains(stderr, "syntax") {
		t.Fatalf("expected ExitError for missing del args, got %d", code)
	}

	// 4. Missing open path
	_, stderr, code = runCLI(t, "", "open")
	if code != ExitError || !strings.Contains(stderr, "missing database path") {
		t.Fatalf("expected ExitError for open without path, got %d", code)
	}
}

func TestCLI_OpenNewAndExisting(t *testing.T) {
	dbPath := tempDBPath(t)

	// 5. Open new database
	stdout, stderr, code := runCLI(t, "", "open", dbPath)
	if code != ExitSuccess || !strings.Contains(stdout, "recreated=true") || stderr != "" {
		t.Fatalf("open new failed: code=%d, stdout=%s, stderr=%s", code, stdout, stderr)
	}

	// 6. Open existing database
	stdout, stderr, code = runCLI(t, "", "open", dbPath)
	if code != ExitSuccess || !strings.Contains(stdout, "recreated=false") || stderr != "" {
		t.Fatalf("open existing failed: code=%d, stdout=%s, stderr=%s", code, stdout, stderr)
	}
}

func TestCLI_PutGetDeleteCycle(t *testing.T) {
	dbPath := tempDBPath(t)

	// 7. Put key
	stdout, stderr, code := runCLI(t, "", "put", dbPath, "relief_item", "emergency_rations")
	if code != ExitSuccess || stdout != "ACK relief_item\n" || stderr != "" {
		t.Fatalf("put failed: code=%d, stdout=%s, stderr=%s", code, stdout, stderr)
	}

	// 8. Get existing key
	stdout, stderr, code = runCLI(t, "", "get", dbPath, "relief_item")
	if code != ExitSuccess || stdout != "emergency_rations\n" || stderr != "" {
		t.Fatalf("get failed: code=%d, stdout=%s, stderr=%s", code, stdout, stderr)
	}

	// 10. Update existing key
	stdout, stderr, code = runCLI(t, "", "put", dbPath, "relief_item", "water_purification")
	if code != ExitSuccess || stdout != "ACK relief_item\n" {
		t.Fatalf("overwrite failed: code=%d, stdout=%s", code, stdout)
	}

	stdout, _, code = runCLI(t, "", "get", dbPath, "relief_item")
	if code != ExitSuccess || stdout != "water_purification\n" {
		t.Fatalf("updated get failed: got %s", stdout)
	}

	// 11. Delete existing key
	stdout, stderr, code = runCLI(t, "", "del", dbPath, "relief_item")
	if code != ExitSuccess || stdout != "ACK relief_item\n" || stderr != "" {
		t.Fatalf("del failed: code=%d, stdout=%s, stderr=%s", code, stdout, stderr)
	}

	// 9. Missing-key Get: stdout must be NOTFOUND and exit code must be 2
	stdout, stderr, code = runCLI(t, "", "get", dbPath, "relief_item")
	if code != ExitNotFound || stdout != "NOTFOUND\n" || stderr != "" {
		t.Fatalf("missing get failed: code=%d, stdout=%s, stderr=%s", code, stdout, stderr)
	}

	// 12. Idempotent deletion on already-deleted key (returns success)
	stdout, stderr, code = runCLI(t, "", "del", dbPath, "relief_item")
	if code != ExitSuccess || stdout != "ACK relief_item\n" {
		t.Fatalf("idempotent del failed: code=%d, stdout=%s", code, stdout)
	}
}

func TestCLI_EmptyValueHandling(t *testing.T) {
	// 13. Empty value is permitted and distinct from deletion
	dbPath := tempDBPath(t)

	stdout, stderr, code := runCLI(t, "", "put", dbPath, "flag_empty", "")
	if code != ExitSuccess || stdout != "ACK flag_empty\n" {
		t.Fatalf("put empty value failed: code=%d, stderr=%s", code, stderr)
	}

	stdout, _, code = runCLI(t, "", "get", dbPath, "flag_empty")
	if code != ExitSuccess || stdout != "\n" {
		t.Fatalf("get empty value failed: code=%d, got %q", code, stdout)
	}
}

func TestCLI_KeyAndValueSizeLimits(t *testing.T) {
	dbPath := tempDBPath(t)

	// 14. Maximum key length (1024 bytes)
	maxKey := strings.Repeat("k", kv.MaxKeyLen)
	stdout, _, code := runCLI(t, "", "put", dbPath, maxKey, "val_max_key")
	if code != ExitSuccess || stdout != fmt.Sprintf("ACK %s\n", maxKey) {
		t.Fatalf("max key put failed: code=%d", code)
	}

	stdout, _, code = runCLI(t, "", "get", dbPath, maxKey)
	if code != ExitSuccess || stdout != "val_max_key\n" {
		t.Fatalf("max key get failed: code=%d", code)
	}

	// 15. Oversized key (1025 bytes) -> must fail with code 1
	oversizedKey := strings.Repeat("x", kv.MaxKeyLen+1)
	_, stderr, code := runCLI(t, "", "put", dbPath, oversizedKey, "val")
	if code != ExitError || !strings.Contains(stderr, "exceeds limits") {
		t.Fatalf("oversized key did not fail: code=%d, stderr=%s", code, stderr)
	}

	// 16. Value size limits verified via batch file (to avoid Windows 32KB command-line string limit)
	dir := t.TempDir()
	largeVal := strings.Repeat("v", 32*1024)
	legalBatchFile := filepath.Join(dir, "legal.json")
	legalJSON := fmt.Sprintf(`[{"op": "put", "key": "large_val_key", "value": "%s"}]`, largeVal)
	_ = os.WriteFile(legalBatchFile, []byte(legalJSON), 0644)

	stdout, _, code = runCLI(t, "", "batch", dbPath, legalBatchFile)
	if code != ExitSuccess || stdout != "ACK 1\n" {
		t.Fatalf("large value batch failed: code=%d", code)
	}

	// 17. Oversized value (> 1 MB) rejected by batch validation
	oversizedVal := strings.Repeat("v", kv.MaxValueLen+10)
	oversizedBatchFile := filepath.Join(dir, "oversized.json")
	oversizedJSON := fmt.Sprintf(`[{"op": "put", "key": "oversized_key", "value": "%s"}]`, oversizedVal)
	_ = os.WriteFile(oversizedBatchFile, []byte(oversizedJSON), 0644)

	_, stderr, code = runCLI(t, "", "batch", dbPath, oversizedBatchFile)
	if code != ExitError || !strings.Contains(stderr, "exceeds limit") {
		t.Fatalf("oversized value did not fail: code=%d, stderr=%s", code, stderr)
	}
}

// --- BATCH TESTS (18 - 25) ---

func TestCLI_BatchJSONAndLineFormats(t *testing.T) {
	dbPath := tempDBPath(t)
	dir := t.TempDir()

	// 18. Valid JSON batch
	jsonFile := filepath.Join(dir, "batch.json")
	jsonContent := `[
		{"op": "put", "key": "supply_rice", "value": "500kg"},
		{"op": "put", "key": "supply_water", "value": "1000L"},
		{"op": "del", "key": "supply_expired"}
	]`
	if err := os.WriteFile(jsonFile, []byte(jsonContent), 0644); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, code := runCLI(t, "", "batch", dbPath, jsonFile)
	if code != ExitSuccess || stdout != "ACK 3\n" || stderr != "" {
		t.Fatalf("batch json failed: code=%d, stdout=%s, stderr=%s", code, stdout, stderr)
	}

	// Verify entries exist
	stdout, _, code = runCLI(t, "", "get", dbPath, "supply_rice")
	if code != ExitSuccess || stdout != "500kg\n" {
		t.Fatalf("expected supply_rice 500kg, got: %s", stdout)
	}

	// Valid line-delimited format
	txtFile := filepath.Join(dir, "batch.txt")
	txtContent := "put supply_blankets 200\ndel supply_water\n"
	if err := os.WriteFile(txtFile, []byte(txtContent), 0644); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, code = runCLI(t, "", "batch", dbPath, txtFile)
	if code != ExitSuccess || stdout != "ACK 2\n" {
		t.Fatalf("batch txt failed: code=%d, stdout=%s, stderr=%s", code, stdout, stderr)
	}

	// Verify water deleted
	_, _, code = runCLI(t, "", "get", dbPath, "supply_water")
	if code != ExitNotFound {
		t.Fatalf("supply_water should be NOTFOUND, got code %d", code)
	}

	// 25. Persistence after reopening
	stdout, _, code = runCLI(t, "", "get", dbPath, "supply_blankets")
	if code != ExitSuccess || stdout != "200\n" {
		t.Fatalf("supply_blankets not persisted across reopening: %s", stdout)
	}
}

func TestCLI_BatchEdgeCasesAndFailures(t *testing.T) {
	dbPath := tempDBPath(t)
	dir := t.TempDir()

	// 19. Empty batch
	emptyFile := filepath.Join(dir, "empty.json")
	_ = os.WriteFile(emptyFile, []byte("[]"), 0644)
	_, stderr, code := runCLI(t, "", "batch", dbPath, emptyFile)
	if code != ExitError || !strings.Contains(stderr, "0 operations") {
		t.Fatalf("empty batch did not return error: code=%d, stderr=%s", code, stderr)
	}

	// 20. Malformed JSON
	badFile := filepath.Join(dir, "bad.json")
	_ = os.WriteFile(badFile, []byte("[{unclosed json"), 0644)
	_, stderr, code = runCLI(t, "", "batch", dbPath, badFile)
	if code != ExitError || !strings.Contains(stderr, "invalid batch file") {
		t.Fatalf("malformed json did not fail: code=%d, stderr=%s", code, stderr)
	}

	// 22. Invalid key in batch (oversized)
	badKeyFile := filepath.Join(dir, "bad_key.json")
	hugeKey := strings.Repeat("z", kv.MaxKeyLen+5)
	badKeyJSON := fmt.Sprintf(`[{"op": "put", "key": "%s", "value": "val"}]`, hugeKey)
	_ = os.WriteFile(badKeyFile, []byte(badKeyJSON), 0644)
	_, stderr, code = runCLI(t, "", "batch", dbPath, badKeyFile)
	if code != ExitError || !strings.Contains(stderr, "exceeds limits") {
		t.Fatalf("invalid key batch did not fail: code=%d, stderr=%s", code, stderr)
	}
}

// --- INTEGRITY, SCAN, DUMP, STATS, COMPACTION (26 - 32) ---

func TestCLI_DumpStatsCheckCompact(t *testing.T) {
	dbPath := tempDBPath(t)

	// Open first to initialize file
	runCLI(t, "", "open", dbPath)

	// 28. Dump empty database
	stdout, _, code := runCLI(t, "", "dump", dbPath)
	if code != ExitSuccess || stdout != "(empty database)\n" {
		t.Fatalf("empty dump failed: code=%d, stdout=%s", code, stdout)
	}

	// Populate entries
	runCLI(t, "", "put", dbPath, "charlie", "3")
	runCLI(t, "", "put", dbPath, "alpha", "1")
	runCLI(t, "", "put", dbPath, "bravo", "2")

	// 29 & 30. Dump populated database with deterministic alphabetical ordering
	stdout, _, code = runCLI(t, "", "dump", dbPath)
	expectedDump := "alpha = 1\nbravo = 2\ncharlie = 3\n"
	if code != ExitSuccess || stdout != expectedDump {
		t.Fatalf("dump ordering mismatch:\nExpected:\n%s\nGot:\n%s", expectedDump, stdout)
	}

	// 31. Stats output
	stdout, _, code = runCLI(t, "", "stats", dbPath)
	if code != ExitSuccess || !strings.Contains(stdout, "LiveKeys: 3") || !strings.Contains(stdout, "TxCount: 3") {
		t.Fatalf("stats failed: code=%d, stdout=%s", code, stdout)
	}

	// 26. Valid Check
	stdout, stderr, code := runCLI(t, "", "check", dbPath)
	if code != ExitSuccess || !strings.Contains(stdout, "OK: Database integrity verified") || stderr != "" {
		t.Fatalf("check failed: code=%d, stdout=%s, stderr=%s", code, stdout, stderr)
	}

	// 32. Compaction
	runCLI(t, "", "del", dbPath, "bravo")
	stdout, stderr, code = runCLI(t, "", "compact", dbPath)
	if code != ExitSuccess || !strings.Contains(stdout, "OK: Compaction completed") || stderr != "" {
		t.Fatalf("compaction failed: code=%d, stdout=%s, stderr=%s", code, stdout, stderr)
	}

	// Post-compaction check
	stdout, _, code = runCLI(t, "", "check", dbPath)
	if code != ExitSuccess || !strings.Contains(stdout, "OK: Database integrity verified") {
		t.Fatalf("post-compaction check failed: %s", stdout)
	}
}

// --- LIFECYCLE & PROCESS LOCKING (33 - 40) ---

func TestCLI_ProcessLockingAndContention(t *testing.T) {
	dbPath := tempDBPath(t)

	// Launch serve in background subprocess
	cmdArgs := []string{"-test.run=^TestCLI_HelperProcess$", "-test.v", "--", "serve", dbPath}
	serveCmd := exec.Command(os.Args[0], cmdArgs...)
	serveCmd.Env = append(os.Environ(), "GO_WANT_CLI_HELPER_PROCESS=1")
	serveCmd.Stderr = os.Stderr

	stdinPipe, err := serveCmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdoutPipe, err := serveCmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := serveCmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = stdinPipe.Close()
		_ = serveCmd.Process.Kill()
	}()

	// Wait for READY token from serve, reading line by line
	reader := bufio.NewReader(stdoutPipe)
	readyFound := false
	for {
		line, readErr := reader.ReadString('\n')
		if readErr != nil {
			break
		}
		if strings.TrimSpace(line) == "READY" {
			readyFound = true
			break
		}
	}
	if !readyFound {
		t.Fatal("serve process did not emit READY")
	}

	// 33. Attempting another write while serve holds lock should fail with busy error
	_, stderr, code := runCLI(t, "", "put", dbPath, "conflict_key", "val")
	if code != ExitError || !strings.Contains(stderr, "locked") {
		t.Fatalf("expected lock conflict error, got code=%d, stderr=%s", code, stderr)
	}

	// Send CLOSE to serve process
	_, _ = io.WriteString(stdinPipe, "CLOSE 999\n")
	_ = stdinPipe.Close()
	_, _ = io.ReadAll(stdoutPipe)
	_ = serveCmd.Wait()

	// 34 & 35. After serve closes, database opens cleanly and accepts writes
	stdout, stderr, code := runCLI(t, "", "put", dbPath, "reclaimed_key", "persisted_ok")
	if code != ExitSuccess || stdout != "ACK reclaimed_key\n" {
		t.Fatalf("write after lock release failed: code=%d, stderr=%s", code, stderr)
	}
}

// --- SERVE IPC PROTOCOL COMPATIBILITY TESTS ---

func TestCLI_ServeIPCProtocolComplete(t *testing.T) {
	dbPath := tempDBPath(t)

	// Test full protocol sequence conforming to Member 3's torture/worker.go
	k1 := base64.StdEncoding.EncodeToString([]byte("med_kit"))
	v1 := base64.StdEncoding.EncodeToString([]byte("antiseptic_gauze"))
	k2 := base64.StdEncoding.EncodeToString([]byte("stretcher"))
	v2 := base64.StdEncoding.EncodeToString([]byte("folding_type"))

	batchJSON, _ := json.Marshal([]map[string]any{
		{"kind": 1, "key": "batch_med1", "val": "burn_ointment"},
		{"kind": 1, "key": "batch_med2", "val": "sterile_gloves"},
	})
	b64batch := base64.StdEncoding.EncodeToString(batchJSON)

	inputScript := fmt.Sprintf(`OPEN %s
PUT 10 %s %s
PUT 11 %s %s
GET 12 %s
BATCH 13 %s
DEL 14 %s
GET 15 %s
COMPACT 16
PING
CLOSE 17
`, dbPath, k1, v1, k2, v2, k1, b64batch, k1, k1)

	stdout, stderr, code := runCLI(t, inputScript, "--worker")
	if code != ExitSuccess || stderr != "" {
		t.Fatalf("serve script failed: code=%d, stderr=%s", code, stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	expectedLines := []string{
		"READY",
		"ACK 10",
		"ACK 11",
		fmt.Sprintf("VAL 12 %s", v1),
		"ACK 13",
		"ACK 14",
		"NOT_FOUND 15",
		"ACK 16",
		"PONG",
		"ACK 17",
	}

	if len(lines) != len(expectedLines) {
		t.Fatalf("unexpected line count: got %d, expected %d.\nGot:\n%s\nExpected:\n%s",
			len(lines), len(expectedLines), stdout, strings.Join(expectedLines, "\n"))
	}

	for i := range expectedLines {
		if strings.TrimSpace(lines[i]) != expectedLines[i] {
			t.Errorf("line %d mismatch: got %q, expected %q", i, strings.TrimSpace(lines[i]), expectedLines[i])
		}
	}

	// Verify post-serve state through standard CLI get
	stdout, _, code = runCLI(t, "", "get", dbPath, "batch_med1")
	if code != ExitSuccess || stdout != "burn_ointment\n" {
		t.Fatalf("batch_med1 not persisted after serve: %s", stdout)
	}
}
