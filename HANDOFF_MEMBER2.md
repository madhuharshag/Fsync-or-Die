# HANDOFF_MEMBER2.md
## Team Innovexa — Fsync-or-Die
### Member 2: Transactions, Recovery, Compaction, Reliability & Torture Testing

**Branch:** `feat/transactions-recovery`  
**Host Environment:** Native Windows 11 AMD64, Go `go1.27.3 windows/386`, PowerShell  
**Primary Integration Owner:** Member 1 (Core Storage & Format)  
**Status:** IMPLEMENTED AND TESTED (All 55 kv unit & subprocess tests PASS, all 12 torture acceptance tests PASS, CLI torture verified)

---

## 1. Scope & Ownership Summary

Following the emergency departure of the reliability teammate, Member 2 absorbed all crash-recovery, incomplete-tail repair, compaction, and reliability torture-testing responsibilities.

### Owned Components:
- **Transaction & Batch Commit Protocol:** Recovery parser and index builder ([`src/kv/db.go`](file:///d:/test/transaction%20and%20recovery%20engine/src/kv/db.go#L273-L410)).
- **Torn-Write & Tail Repair:** Header, payload, footer, and CRC validation with safe truncation ([`src/kv/recovery_test.go`](file:///d:/test/transaction%20and%20recovery%20engine/src/kv/recovery_test.go)).
- **Crash-Safe Compaction:** Dual-bounded batching (1,000 ops / 16 MiB payload), streaming snapshot, atomic swap, and rollback safety ([`src/kv/compaction.go`](file:///d:/test/transaction%20and%20recovery%20engine/src/kv/compaction.go)).
- **Process Lock Separation:** Dedicated advisory lock file (`<path>.lock`) using Win32 `LockFileEx` ([`src/kv/lock_windows.go`](file:///d:/test/transaction%20and%20recovery%20engine/src/kv/lock_windows.go)) maintaining mutual exclusion during file descriptor swaps.
- **Independent Torture Test Harness:** Standalone Go harness under [`torture/`](file:///d:/test/transaction%20and%20recovery%20engine/torture/) with an out-of-process worker, persistent ACK oracle, random crash injector, and zero dependence on internal DB state.
- **1,000,000-Key Cold Recovery Benchmark:** Real database recovery benchmark ([`src/kv/recovery_1m_bench_test.go`](file:///d:/test/transaction%20and%20recovery%20engine/src/kv/recovery_1m_bench_test.go)) achieving 0.462s cold recovery (< 2.0s goal).

---

## 2. Transaction & Batch Recovery Architecture

### Recovery Protocol (`recoverAndRepairLocked` in `src/kv/db.go`)
1. Recovery scans transactions sequentially starting at byte offset 32 (immediately following the immutable `FileHeader`).
2. For each transaction:
   - Validates 32-byte `TxnHeader` (magic `TXN1`, version, sequential `TxID`, payload length).
   - Reads `PayloadLen` bytes of packed operations.
   - Reads 32-byte `CommitFooter` (magic `COMMIT01`, matching `TxID`, matching `TotalTxBytes`).
   - Computes and verifies dual CRCs:
     - `TxnCRC`: CRC32-IEEE over `(TxnHeader || Payload)`.
     - `FooterCRC`: CRC32-IEEE over footer bytes `[0..27]`.
   - On full validation, operations are decoded and committed to the in-memory `Index`.
3. **Torn Tail Handling:**
   - If an uncommitted or partially written tail is detected past the last committed transaction (e.g. truncated header, partial payload, corrupt footer, mismatched CRC):
   - The loop terminates at the last committed byte boundary.
   - In read-write mode, the file is physically truncated to `offset` using `TruncateFile` and flushed with `SyncFile`.
   - `RepairedTailBytes` is reported in `OpenInfo`.
   - In read-only mode, the file is not modified, and the handle is marked `ErrPoisoned` to prevent corrupted operations.

### Corrected F4 One-Byte Truncation Scenario
- **Previous Flawed Assertion:** The prior test wrote two acknowledged transactions using the public `Put` API, truncated 1 byte, and accepted the loss of the second transaction. This violated the core durability contract: an acknowledged write must never be lost.
- **Corrected Implementation (`TestRecovery_OneByteTruncation_F4`):**
  1. Writes acknowledged key `"first"` -> `"value1"` and `"second_ack"` -> `"value2"` using public `Put` calls.
  2. Closes database and records authoritative file size.
  3. Uses a controlled low-level fixture to append an unacknowledged trailing transaction without calling `Put`.
  4. Truncates exactly 1 byte from that unacknowledged trailing transaction.
  5. Reopens database:
     - Both acknowledged keys (`"first"` and `"second_ack"`) survive with exact values.
     - The unacknowledged trailing mutation is completely discarded.
     - Zero partial or invented values appear in the index.
     - Reopened database accepts subsequent writes cleanly.

---

## 3. Compaction Algorithm & Memory Bounding

### Algorithm Lifecycle (`compactLocked` in `src/kv/compaction.go`)
1. Creates temporary replacement file `<path>.compact.tmp`.
2. Writes immutable 32-byte `FileHeader`.
3. Iterates live keys from `db.index.Keys()` in deterministic sorted order.
4. Streams live keys into the new file in bounded transactions constrained by:
   - **Operation Count Limit:** 1,000 operations per transaction (`compactionBatchOps`).
   - **Payload Byte Size Limit:** 16 MiB per transaction (`compactionBatchBytes = 16 * 1024 * 1024`).
5. For each key, calculates exact encoded operation size (`OpHeaderSize + len(key) + len(val)`) **before** reading or allocating value bytes:
   - If adding the op would exceed either limit, flushes the current transaction first.
   - If an individual op exceeds 16 MiB, compaction fails with `ErrInvalid` without altering the database.
   - Legal maximum values (up to 1 MB) are always admitted safely into an empty batch.
6. Calls `SyncFile(tmpFile)` to guarantee physical durability on disk.
7. Closes `tmpFile` and `db.file`.
8. Calls `os.Rename(tmpPath, db.path)` to atomically swap directory pointers.
9. Reopens `db.path` and atomically updates in-memory `Index`, `committedEnd`, and `nextTxID`.

### Rollback & Failure Protection
- If `os.Rename` fails (e.g. sharing violation on Windows):
  - Compaction reopens the original authoritative file `db.path`.
  - Removes the temporary file `_ = os.Remove(tmpPath)`.
  - Returns `fmt.Errorf("compaction: atomic rename failed: %w", err)`.
  - The database is **not poisoned** and continues serving reads and writes without data loss.
- If reopening also fails, `db.poisoned` is set to prevent silent corruption.

---

## 4. Process-Lock Continuity

- **Architecture:** Process-level mutual exclusion is maintained on `<path>.lock` rather than the data file descriptor `<path>`.
- **Platform Implementation:**
  - Windows: Win32 `LockFileEx` (`LOCKFILE_EXCLUSIVE_LOCK | LOCKFILE_FAIL_IMMEDIATELY`).
  - Linux / Darwin: `syscall.Flock` (`LOCK_EX | LOCK_NB`).
- **Continuity across Compaction:**
  - Compaction closes and reopens `<path>`, but **never closes `<path>.lock`**.
  - A second process calling `Open()` while compaction is paused before rename receives `ErrBusy` (proven by `TestSubprocess_2_LockContinuityDuringCompaction`).
  - Sudden process termination (`TerminateProcess` / `SIGKILL`) automatically releases the OS file lock, allowing restart recovery.

---

## 5. Platform Guarantee Matrix

| Platform | Atomic File Replacement | Open Handle Replacement | Process Lock Mechanism | Status |
| :--- | :--- | :--- | :--- | :--- |
| **Native Windows 11 (AMD64 / x86)** | Atomic metadata pointer swap via `MoveFileExW(MOVEFILE_REPLACE_EXISTING)` on same volume | Requires handles to old and new file to be closed first; otherwise returns `ERROR_SHARING_VIOLATION` or `ERROR_ACCESS_DENIED` | Win32 `LockFileEx` / `UnlockFileEx` on `<path>.lock` | **VERIFIED BY RUNTIME TEST** (55 unit/subprocess tests + torture harness) |
| **Linux (ext4/XFS)** | Atomic inode directory link replacement via `rename(2)` | Destination unlinked while open handles remain valid | `syscall.Flock` on `<path>.lock` | **VERIFIED BY DOCUMENTATION ONLY** |
| **macOS (APFS)** | Atomic replacement via `rename(2)` / `renamex_np` | Destination unlinked while open handles remain valid | `syscall.Flock` on `<path>.lock` | **VERIFIED BY DOCUMENTATION ONLY** |

---

## 6. Independent Torture Testing Harness (`torture/`)

A standalone reliability torture tester implemented in package `main` under `torture/`:

### Key Capabilities:
1. **Independent Persistent ACK Oracle (`Oracle`):**
   - Keeps live key-value state and delete tombstones in memory.
   - Synchronously flushes an append-only JSON journal (`oracle.log`) with `Sync()` before every crash point.
   - Validates after every restart that every acknowledged write survives, all deletes remain absent, no partial batches appear, and no invented keys exist.
2. **Ambiguous Operation Handling:**
   - In-flight mutations interrupted before an ACK is observed are classified as `INDETERMINATE`.
   - Single operations may be present or absent, but never corrupt.
   - Batches must be 100% applied or 0% applied (All-or-Nothing atomicity).
3. **Out-of-Process Execution & Forced Kills:**
   - Spawns the database engine as a child subprocess (`--worker`) with isolated RAM and descriptors.
   - Executes deliberate kills via OS process kill (`TerminateProcess`).
4. **Reproducibility:**
   - Runs accept `--seed <int64>`. Any run can be reproduced by passing the reported seed.

---

## 7. 1,000,000-Key Cold Recovery Benchmark

Executed on native Windows 11 (`windows/386` Go toolchain, 24 CPU cores):
- **Database Size:** 25.76 MB (27,006,432 bytes)
- **Transactions Recovered:** 100 transactions (10,000 keys per transaction)
- **Live Keys Recovered:** 1,000,000 keys
- **Cold Recovery Elapsed Time:** **461.536 ms** (0.462 seconds)
- **Challenge Goal (< 2.0s):** **PASSED** (more than 4x faster than required threshold)
- **RAM Allocated during Recovery:** 88.45 MB

---

## 8. Integration Instructions for Member 1

1. **Shared Contract / Format Files:**
   - No modifications were made to `src/kv/contract.go` or `src/kv/format.go`.
   - No modifications were made to the on-disk binary layout or transaction commit protocol.
2. **Open Design Decision (Scenario E5):**
   - When the file reaches 100% capacity (`committedEnd >= maxFileBytes`), deleting a key requires appending a 72-byte tombstone. Currently `CheckCapacity` returns `ErrFull`.
   - If Member 1 approves reserving a delete tombstone headroom (e.g. 64 KB) or allowing tombstones when full, Member 2 can integrate the change. Currently preserved unchanged per instructions.
3. **Running the Test Suite:**
   ```powershell
   go test -v -count=1 ./src/kv
   go test -v -count=1 ./torture
   go run ./torture -seed 12345 -ops 500 -crashes 5 -writers 2
   go test -v -run TestRecovery_1MillionKeys_ColdOpen ./src/kv
   ```
