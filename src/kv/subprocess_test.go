package kv

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestHelperProcess serves as the entrypoint for subprocesses spawned during tests.
// It is activated when TEST_SUBPROCESS=1 is present in the environment.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("TEST_SUBPROCESS") != "1" {
		return
	}

	mode := os.Getenv("SUBPROCESS_MODE")
	dbPath := os.Getenv("SUBPROCESS_DBPATH")

	switch mode {
	case "hold_db":
		db, _, err := Open(dbPath, Options{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: open failed: %v\n", err)
			os.Exit(1)
		}
		// Signal to parent that DB and lock are acquired
		fmt.Println("READY")
		// Wait for parent's instruction to quit
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			_ = db.Close()
		}
		os.Exit(0)

	case "compaction_pause_sync":
		db, _, err := Open(dbPath, Options{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: open failed: %v\n", err)
			os.Exit(1)
		}
		compactionHook = func(stage string) error {
			if stage == "after_sync" {
				fmt.Println("READY")
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan() // Wait for parent signal to proceed
			}
			return nil
		}
		if err := db.Compact(); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: compact failed: %v\n", err)
			os.Exit(1)
		}
		_ = db.Close()
		fmt.Println("COMPACT_DONE")
		os.Exit(0)

	case "kill_writing_tmp":
		db, _, err := Open(dbPath, Options{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: open failed: %v\n", err)
			os.Exit(1)
		}
		compactionHook = func(stage string) error {
			if stage == "writing_tmp" {
				fmt.Println("READY")
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan() // Wait until killed
			}
			return nil
		}
		_ = db.Compact()
		_ = db.Close()
		os.Exit(0)

	case "kill_after_sync":
		db, _, err := Open(dbPath, Options{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: open failed: %v\n", err)
			os.Exit(1)
		}
		compactionHook = func(stage string) error {
			if stage == "after_sync" {
				fmt.Println("READY")
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan() // Wait until killed
			}
			return nil
		}
		_ = db.Compact()
		_ = db.Close()
		os.Exit(0)

	case "kill_after_rename":
		db, _, err := Open(dbPath, Options{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: open failed: %v\n", err)
			os.Exit(1)
		}
		compactionHook = func(stage string) error {
			if stage == "after_rename" {
				fmt.Println("READY")
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan() // Wait until killed
			}
			return nil
		}
		_ = db.Compact()
		_ = db.Close()
		os.Exit(0)

	default:
		fmt.Fprintf(os.Stderr, "ERROR: unknown mode %s\n", mode)
		os.Exit(1)
	}
}

// spawnHelper spawns the current test binary configured for helper mode.
func spawnHelper(t *testing.T, mode, dbPath string) (*exec.Cmd, io.WriteCloser, *bufio.Reader) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable failed: %v", err)
	}

	cmd := exec.Command(exe, "-test.run=^TestHelperProcess$", "-test.v")
	cmd.Env = append(os.Environ(),
		"TEST_SUBPROCESS=1",
		"SUBPROCESS_MODE="+mode,
		"SUBPROCESS_DBPATH="+dbPath,
	)

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("StdinPipe failed: %v", err)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe failed: %v", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("cmd.Start failed: %v", err)
	}

	reader := bufio.NewReader(stdoutPipe)
	return cmd, stdinPipe, reader
}

// waitForToken reads lines from the helper process stdout until targetToken is found.
func waitForToken(t *testing.T, reader *bufio.Reader, targetToken string) {
	t.Helper()
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("failed waiting for token %q: %v", targetToken, err)
		}
		line = strings.TrimSpace(line)
		if line == targetToken {
			return
		}
		if strings.HasPrefix(line, "ERROR:") {
			t.Fatalf("subprocess error: %s", line)
		}
	}
}

// TestSubprocess_1_SecondProcessLockExclusion verifies that a second process cannot
// open the database while the first process holds it.
func TestSubprocess_1_SecondProcessLockExclusion(t *testing.T) {
	dir, err := os.MkdirTemp("", "kipdb_subproc_lock_*")
	if err != nil {
		t.Fatalf("MkdirTemp failed: %v", err)
	}
	defer os.RemoveAll(dir)
	dbPath := filepath.Join(dir, "locked.kip")

	// Prepopulate database with initial data
	db0, _, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Initial Open failed: %v", err)
	}
	if err := db0.Put([]byte("key1"), []byte("val1")); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	_ = db0.Close()

	// 1. Spawn child process to hold the database
	cmd, stdinPipe, stdoutReader := spawnHelper(t, "hold_db", dbPath)
	waitForToken(t, stdoutReader, "READY")

	// 2. Second process (parent test) attempts to open the same DB: must return ErrBusy
	_, _, err = Open(dbPath, Options{})
	if !errors.Is(err, ErrBusy) {
		t.Fatalf("expected ErrBusy when opening database held by another process, got: %v", err)
	}

	// 3. Gracefully release child process
	fmt.Fprintln(stdinPipe, "QUIT")
	_ = stdinPipe.Close()
	if err := cmd.Wait(); err != nil {
		t.Fatalf("child process exit error: %v", err)
	}

	// 4. Now parent process can open the database cleanly
	db1, _, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Open after child release failed: %v", err)
	}
	defer db1.Close()

	val, err := db1.Get([]byte("key1"))
	if err != nil || string(val) != "val1" {
		t.Fatalf("expected val1, got %q (err=%v)", val, err)
	}
}

// TestSubprocess_2_LockContinuityDuringCompaction verifies that a second process
// cannot open the database while the owner process is paused inside compaction,
// even when the primary data file handle is momentarily closed before rename.
func TestSubprocess_2_LockContinuityDuringCompaction(t *testing.T) {
	dir, err := os.MkdirTemp("", "kipdb_subproc_compact_lock_*")
	if err != nil {
		t.Fatalf("MkdirTemp failed: %v", err)
	}
	defer os.RemoveAll(dir)
	dbPath := filepath.Join(dir, "compact_lock.kip")

	// Prepopulate database with live and deleted keys
	db0, _, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Initial Open failed: %v", err)
	}
	for i := 0; i < 20; i++ {
		_ = db0.Put([]byte(fmt.Sprintf("k%02d", i)), []byte(fmt.Sprintf("v%02d", i)))
	}
	for i := 0; i < 5; i++ {
		_ = db0.Delete([]byte(fmt.Sprintf("k%02d", i)))
	}
	_ = db0.Close()

	// 1. Spawn child that pauses inside compaction after tmpFile is synced and handles closed
	cmd, stdinPipe, stdoutReader := spawnHelper(t, "compaction_pause_sync", dbPath)
	waitForToken(t, stdoutReader, "READY")

	// 2. Attempt to open from second process while child is paused inside compaction:
	// The process lock (<path>.lock) is held continuously, so ErrBusy must be returned!
	_, _, err = Open(dbPath, Options{})
	if !errors.Is(err, ErrBusy) {
		t.Fatalf("expected ErrBusy while compaction is paused before rename, got: %v", err)
	}

	// 3. Allow child to complete compaction
	fmt.Fprintln(stdinPipe, "CONTINUE")
	_ = stdinPipe.Close()
	waitForToken(t, stdoutReader, "COMPACT_DONE")
	if err := cmd.Wait(); err != nil {
		t.Fatalf("child process exit error: %v", err)
	}

	// 4. Verify compacted database is intact and holds all remaining live keys
	db1, info, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Open after compaction failed: %v", err)
	}
	defer db1.Close()

	if info.LiveKeyCount != 15 {
		t.Fatalf("expected 15 live keys after compaction, got %d", info.LiveKeyCount)
	}
	for i := 5; i < 20; i++ {
		val, err := db1.Get([]byte(fmt.Sprintf("k%02d", i)))
		expected := fmt.Sprintf("v%02d", i)
		if err != nil || string(val) != expected {
			t.Fatalf("key k%02d mismatch: got %q, want %q", i, val, expected)
		}
	}
}

// TestSubprocess_3_KillOwnerReleasesLock verifies that forcefully killing the owner process
// releases the OS lock and allows a new process to reopen and recover the database.
func TestSubprocess_3_KillOwnerReleasesLock(t *testing.T) {
	dir, err := os.MkdirTemp("", "kipdb_subproc_kill_owner_*")
	if err != nil {
		t.Fatalf("MkdirTemp failed: %v", err)
	}
	defer os.RemoveAll(dir)
	dbPath := filepath.Join(dir, "owner_kill.kip")

	db0, _, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Initial Open failed: %v", err)
	}
	_ = db0.Put([]byte("persisted_key"), []byte("persisted_val"))
	_ = db0.Close()

	// 1. Spawn child process holding the DB
	cmd, _, stdoutReader := spawnHelper(t, "hold_db", dbPath)
	waitForToken(t, stdoutReader, "READY")

	// 2. Confirm second open is blocked
	_, _, err = Open(dbPath, Options{})
	if !errors.Is(err, ErrBusy) {
		t.Fatalf("expected ErrBusy, got: %v", err)
	}

	// 3. Forcefully kill the owner process
	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("Process.Kill failed: %v", err)
	}
	_ = cmd.Wait()

	// 4. Reopen from new process: OS lock must be fully released and data intact
	db1, _, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Open after owner kill failed: %v", err)
	}
	defer db1.Close()

	val, err := db1.Get([]byte("persisted_key"))
	if err != nil || string(val) != "persisted_val" {
		t.Fatalf("persisted_key missing or mismatch: got %q, err: %v", val, err)
	}

	// Verify database is functional and can accept new writes
	if err := db1.Put([]byte("new_after_kill"), []byte("survived")); err != nil {
		t.Fatalf("Put after owner kill failed: %v", err)
	}
	val2, err := db1.Get([]byte("new_after_kill"))
	if err != nil || string(val2) != "survived" {
		t.Fatalf("new_after_kill mismatch: got %q, err: %v", val2, err)
	}
}

// TestSubprocess_4_KillCompactionWhileWritingTmp verifies that killing the compaction child
// while writing the temporary file leaves the original database untouched and usable,
// preserving all previously committed data.
func TestSubprocess_4_KillCompactionWhileWritingTmp(t *testing.T) {
	dir, err := os.MkdirTemp("", "kipdb_subproc_kill_writing_*")
	if err != nil {
		t.Fatalf("MkdirTemp failed: %v", err)
	}
	defer os.RemoveAll(dir)
	dbPath := filepath.Join(dir, "kill_writing.kip")

	db0, _, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Initial Open failed: %v", err)
	}
	for i := 0; i < 30; i++ {
		_ = db0.Put([]byte(fmt.Sprintf("k%d", i)), []byte(fmt.Sprintf("val%d", i)))
	}
	_ = db0.Close()

	// 1. Spawn child and wait until it is actively in compaction writing tmp
	cmd, _, stdoutReader := spawnHelper(t, "kill_writing_tmp", dbPath)
	waitForToken(t, stdoutReader, "READY")

	// 2. Kill the child process mid-compaction
	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("Kill failed: %v", err)
	}
	_ = cmd.Wait()

	// 3. Reopen database: original database must be intact and all 30 keys present
	db1, info, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Open after kill during tmp write failed: %v", err)
	}
	defer db1.Close()

	if info.LiveKeyCount != 30 {
		t.Fatalf("expected 30 live keys, got %d", info.LiveKeyCount)
	}
	for i := 0; i < 30; i++ {
		val, err := db1.Get([]byte(fmt.Sprintf("k%d", i)))
		expected := fmt.Sprintf("val%d", i)
		if err != nil || string(val) != expected {
			t.Fatalf("key k%d mismatch: got %q, want %q", i, val, expected)
		}
	}

	// 4. Verify stray tmp file was cleaned up on Open
	tmpPath := dbPath + ".compact.tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Fatalf("expected stray tmp file %s to be removed on Open, but it still exists", tmpPath)
	}
}

// TestSubprocess_5_KillCompactionAfterSyncBeforeRename verifies that killing the child
// after the replacement file is complete and synced, but before rename, leaves the original
// database authoritative and intact, cleaning up the temporary file on restart.
func TestSubprocess_5_KillCompactionAfterSyncBeforeRename(t *testing.T) {
	dir, err := os.MkdirTemp("", "kipdb_subproc_kill_sync_*")
	if err != nil {
		t.Fatalf("MkdirTemp failed: %v", err)
	}
	defer os.RemoveAll(dir)
	dbPath := filepath.Join(dir, "kill_sync.kip")

	db0, _, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Initial Open failed: %v", err)
	}
	for i := 0; i < 20; i++ {
		_ = db0.Put([]byte(fmt.Sprintf("key_%d", i)), []byte(fmt.Sprintf("payload_%d", i)))
	}
	_ = db0.Close()

	// 1. Spawn child and wait until sync is complete but rename has NOT occurred
	cmd, _, stdoutReader := spawnHelper(t, "kill_after_sync", dbPath)
	waitForToken(t, stdoutReader, "READY")

	// Temporary file should exist and be fully synced
	tmpPath := dbPath + ".compact.tmp"
	if _, err := os.Stat(tmpPath); err != nil {
		t.Fatalf("expected tmp file %s to exist before rename, err: %v", tmpPath, err)
	}

	// 2. Kill child before rename
	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("Kill failed: %v", err)
	}
	_ = cmd.Wait()

	// 3. Reopen database: original authoritative file must be recovered and tmp cleaned up
	db1, info, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Open after kill before rename failed: %v", err)
	}
	defer db1.Close()

	if info.LiveKeyCount != 20 {
		t.Fatalf("expected 20 live keys, got %d", info.LiveKeyCount)
	}
	for i := 0; i < 20; i++ {
		val, err := db1.Get([]byte(fmt.Sprintf("key_%d", i)))
		expected := fmt.Sprintf("payload_%d", i)
		if err != nil || string(val) != expected {
			t.Fatalf("key_%d mismatch: got %q, want %q", i, val, expected)
		}
	}

	// Verify tmp file was removed
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Fatalf("expected tmp file to be cleaned up on Open, but it still exists")
	}
}

// TestSubprocess_6_KillCompactionAfterRenameBeforeReopen verifies that killing the child
// after rename has succeeded, but before reopen, leaves the newly installed compacted file
// authoritative and completely recovered with all live data present.
func TestSubprocess_6_KillCompactionAfterRenameBeforeReopen(t *testing.T) {
	dir, err := os.MkdirTemp("", "kipdb_subproc_kill_rename_*")
	if err != nil {
		t.Fatalf("MkdirTemp failed: %v", err)
	}
	defer os.RemoveAll(dir)
	dbPath := filepath.Join(dir, "kill_rename.kip")

	db0, _, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Initial Open failed: %v", err)
	}
	for i := 0; i < 40; i++ {
		_ = db0.Put([]byte(fmt.Sprintf("rk%02d", i)), []byte(fmt.Sprintf("original_%d", i)))
	}
	// Overwrite 10 keys and delete 10 keys
	for i := 0; i < 10; i++ {
		_ = db0.Put([]byte(fmt.Sprintf("rk%02d", i)), []byte(fmt.Sprintf("updated_%d", i)))
	}
	for i := 30; i < 40; i++ {
		_ = db0.Delete([]byte(fmt.Sprintf("rk%02d", i)))
	}
	_ = db0.Close()

	// 1. Spawn child and wait until rename has completed
	cmd, _, stdoutReader := spawnHelper(t, "kill_after_rename", dbPath)
	waitForToken(t, stdoutReader, "READY")

	// 2. Kill child after rename
	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("Kill failed: %v", err)
	}
	_ = cmd.Wait()

	// 3. Reopen database: the compacted file is now authoritative at dbPath!
	db1, info, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Open after kill post-rename failed: %v", err)
	}
	defer db1.Close()

	// Total live keys: 40 - 10 deletes = 30 live keys
	if info.LiveKeyCount != 30 {
		t.Fatalf("expected 30 live keys in compacted database, got %d", info.LiveKeyCount)
	}

	// Check updated keys
	for i := 0; i < 10; i++ {
		val, err := db1.Get([]byte(fmt.Sprintf("rk%02d", i)))
		expected := fmt.Sprintf("updated_%d", i)
		if err != nil || string(val) != expected {
			t.Fatalf("updated key rk%02d mismatch: got %q, want %q", i, val, expected)
		}
	}
	// Check un-updated keys
	for i := 10; i < 30; i++ {
		val, err := db1.Get([]byte(fmt.Sprintf("rk%02d", i)))
		expected := fmt.Sprintf("original_%d", i)
		if err != nil || string(val) != expected {
			t.Fatalf("original key rk%02d mismatch: got %q, want %q", i, val, expected)
		}
	}
	// Check deleted keys
	for i := 30; i < 40; i++ {
		_, err := db1.Get([]byte(fmt.Sprintf("rk%02d", i)))
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("expected ErrNotFound for deleted key rk%02d, got %v", i, err)
		}
	}

	// Verify database accepts new writes
	if err := db1.Put([]byte("new_post_compact"), []byte("success")); err != nil {
		t.Fatalf("Put post-compact failed: %v", err)
	}
	got, err := db1.Get([]byte("new_post_compact"))
	if err != nil || string(got) != "success" {
		t.Fatalf("new_post_compact mismatch: got %q, err: %v", got, err)
	}
}

// TestSubprocess_7_RenameFailureSafety verifies that if rename fails (e.g. sharing violation
// on Windows or injected filesystem error), the engine does not report success, safely reopens
// the authoritative file, and leaves the database completely usable without data loss.
func TestSubprocess_7_RenameFailureSafety(t *testing.T) {
	dir, err := os.MkdirTemp("", "kipdb_subproc_rename_fail_*")
	if err != nil {
		t.Fatalf("MkdirTemp failed: %v", err)
	}
	defer os.RemoveAll(dir)
	dbPath := filepath.Join(dir, "rename_fail.kip")

	db, _, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	if err := db.Put([]byte("keyA"), []byte("valA")); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if err := db.Put([]byte("keyB"), []byte("valB")); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Case A: Injected rename error
	compactionRenameFn = func(oldpath, newpath string) error {
		return fmt.Errorf("injected rename I/O failure (sharing violation simulation)")
	}
	defer func() {
		compactionRenameFn = os.Rename
	}()

	compactErr := db.Compact()
	if compactErr == nil {
		t.Fatalf("expected Compact() to fail when rename fails, but got nil")
	}
	if !strings.Contains(compactErr.Error(), "injected rename I/O failure") {
		t.Fatalf("unexpected error message: %v", compactErr)
	}

	// Verify that the engine has recovered and reopened the original database file
	valA, err := db.Get([]byte("keyA"))
	if err != nil || string(valA) != "valA" {
		t.Fatalf("keyA lost or corrupted after failed rename: got %q, err: %v", valA, err)
	}
	valB, err := db.Get([]byte("keyB"))
	if err != nil || string(valB) != "valB" {
		t.Fatalf("keyB lost or corrupted after failed rename: got %q, err: %v", valB, err)
	}

	// Subsequent write succeeds on the restored descriptor
	if err := db.Put([]byte("keyC"), []byte("valC")); err != nil {
		t.Fatalf("Put failed on restored engine: %v", err)
	}
	valC, err := db.Get([]byte("keyC"))
	if err != nil || string(valC) != "valC" {
		t.Fatalf("keyC mismatch: got %q, err: %v", valC, err)
	}

	// Verify temporary file was deleted after rename failure
	tmpPath := dbPath + ".compact.tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Fatalf("temporary file %s was not cleaned up after rename failure", tmpPath)
	}
}

// TestSubprocess_7B_NativeWindowsSharingViolation triggers an actual native Windows
// sharing violation by keeping an open read handle on the destination file.
func TestSubprocess_7B_NativeWindowsSharingViolation(t *testing.T) {
	dir, err := os.MkdirTemp("", "kipdb_native_sharing_*")
	if err != nil {
		t.Fatalf("MkdirTemp failed: %v", err)
	}
	defer os.RemoveAll(dir)
	dbPath := filepath.Join(dir, "sharing.kip")

	db, _, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	if err := db.Put([]byte("stable_key"), []byte("stable_val")); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Open a raw file handle on the DB file without FILE_SHARE_DELETE
	// On Windows, this will cause MoveFileEx / os.Rename to fail with sharing violation
	blockerFile, err := os.Open(dbPath)
	if err != nil {
		t.Fatalf("os.Open failed: %v", err)
	}

	// Run compaction: rename will encounter sharing violation because blockerFile holds dbPath
	compactErr := db.Compact()
	_ = blockerFile.Close() // Release blocker immediately

	if compactErr == nil {
		t.Logf("Note: OS permitted rename over open handle (non-Windows or POSIX emulation)")
	} else {
		// On Windows, this error is guaranteed
		t.Logf("Observed expected Windows rename sharing error: %v", compactErr)
	}

	// In both cases, the database must remain fully usable and uncorrupted
	val, err := db.Get([]byte("stable_key"))
	if err != nil || string(val) != "stable_val" {
		t.Fatalf("stable_key corrupted or missing: got %q, err: %v", val, err)
	}

	// Subsequent write succeeds
	if err := db.Put([]byte("new_key"), []byte("new_val")); err != nil {
		t.Fatalf("Put failed after sharing violation test: %v", err)
	}
	valNew, err := db.Get([]byte("new_key"))
	if err != nil || string(valNew) != "new_val" {
		t.Fatalf("new_key mismatch: got %q, err: %v", valNew, err)
	}
}

// TestSubprocess_7C_SimulatedReopenFailurePoisonsHandle verifies that if rename fails
// AND the old file cannot be reopened, the handle is safely poisoned to prevent silent corruption.
func TestSubprocess_7C_SimulatedReopenFailurePoisonsHandle(t *testing.T) {
	dir, err := os.MkdirTemp("", "kipdb_subproc_poison_*")
	if err != nil {
		t.Fatalf("MkdirTemp failed: %v", err)
	}
	defer os.RemoveAll(dir)
	dbPath := filepath.Join(dir, "poison.kip")

	db, _, err := Open(dbPath, Options{})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	_ = db.Put([]byte("k"), []byte("v"))

	// Simulate catastrophic scenario where rename fails AND original file was deleted/unopenable
	compactionRenameFn = func(oldpath, newpath string) error {
		// Delete original file to make reopen fail
		_ = os.Remove(newpath)
		return fmt.Errorf("injected failure")
	}
	defer func() {
		compactionRenameFn = os.Rename
	}()

	err = db.Compact()
	if err == nil {
		t.Fatalf("expected Compact() error, got nil")
	}

	// Verify handle is marked poisoned
	putErr := db.Put([]byte("another"), []byte("val"))
	if !errors.Is(putErr, ErrPoisoned) {
		t.Fatalf("expected ErrPoisoned after unrecoverable rename failure, got: %v", putErr)
	}
}
