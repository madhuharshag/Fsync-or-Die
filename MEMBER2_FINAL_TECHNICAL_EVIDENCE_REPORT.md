# MEMBER 2 FINAL TECHNICAL EVIDENCE REPORT
## Project Fsync-or-Die — Team Innovexa
### Role: Member 2 — Transactions, Recovery, Compaction, Reliability & Torture-Testing Engineer

---

## 1. Executive Summary

This report documents the completion of Member 2's core responsibilities and the absorbed reliability and torture-testing responsibilities for Project **Fsync-or-Die**. 

All work was performed directly in repository `https://github.com/madhuharshag/Fsync-or-Die` on assigned branch `feat/transactions-recovery` on native Windows 11 using the host Go toolchain (`go1.27.3 windows/386`).

### Key Accomplishments & Verified Status:
1. **F4 One-Byte Truncation Test Corrected:** Eliminated the flawed assertion where acknowledged writes were discarded. Implemented a low-level test fixture simulating an unacknowledged trailing transaction; proved that all acknowledged data survives without partial or invented values (20/20 consecutive runs passed).
2. **Compaction Memory Dual-Bounded:** Compaction batches are strictly bounded by both operation count (1,000 ops) and encoded payload bytes (16 MiB), guarding against 32-bit virtual memory exhaustion. Single maximum-size values (1 MB) are safely accommodated.
3. **Crash-Safe Compaction & Failure Protection:** Proved through deterministic subprocess and native Windows sharing-violation tests that interrupted compaction (writing tmp, after sync, after rename) and rename failures never corrupt the authoritative database or result in false-success reports.
4. **Process-Lock Continuity Proven:** Demonstrated that `<path>.lock` using Win32 `LockFileEx` remains held across the entire compaction sequence, blocking concurrent processes even when data file descriptors are momentarily closed before rename.
5. **Independent Torture Test Harness Built:** Delivered a standalone Go-based torture tester in `torture/` featuring an out-of-process worker, persistent ACK oracle, randomized crash scheduler, multi-writer support, and 15 passing acceptance criteria.
6. **1,000,000-Key Recovery Benchmark Measured:** Cold recovery of 1,000,000 live keys from a 25.76 MB on-disk file was completed in **461.536 ms** (0.462s), beating the official 2.0-second threshold by more than 4x.
7. **Zero Shared Format / API Breakages:** Preserved Member 1's on-disk layout, headers, footers, CRC scheme, and storage interfaces without modification.

---

## 2. Repository and Environment Verification

```text
Repository URL:      https://github.com/madhuharshag/Fsync-or-Die.git
Assigned Branch:     feat/transactions-recovery
Starting Commit:     3057b6636edc62d92582685fd4afa943e98f4678
Host OS:             Microsoft Windows 11 (NT 10.0.26300.0, 64-bit AMD64)
CPU Hardware:        24 logical processor cores
Host Shell:          PowerShell 5.1 / 7
Active Go Binary:    C:\Users\sanke\Downloads\go\bin\go.exe
Go Version:          go version go1.27.3 windows/386
Target Architecture: windows/386 (32-bit x86 execution)
go.mod Declaration:  go 1.23
Working Tree Status: Clean commits ready; all files tracked and formatted.
```

> [!NOTE]
> The host machine is a 64-bit Windows installation running a 32-bit `windows/386` Go compiler. All runtime tests were executed under this native 32-bit target. Concurrency race detection (`go test -race`) is not supported by the Go runtime on 32-bit x86 platforms.

---

## 3. Files Modified and Created

| File | Status | Purpose / Description |
| :--- | :--- | :--- |
| [`src/kv/compaction.go`](src/kv/compaction.go) | Modified | Dual-bounded compaction batching (1,000 ops / 16 MiB), `opBytes` pre-check, and internal test seams (`compactionHook`, `compactionRenameFn`). |
| [`src/kv/compaction_test.go`](src/kv/compaction_test.go) | Modified | Added tests for byte boundary splitting, single max value (1 MB), multi-large values, individually oversized op rejection, and op-count chunking. |
| [`src/kv/recovery_test.go`](src/kv/recovery_test.go) | Modified | Corrected `TestRecovery_OneByteTruncation_F4` using low-level unacknowledged trailing transaction fixture. |
| [`src/kv/subprocess_test.go`](src/kv/subprocess_test.go) | Created | 9 deterministic subprocess tests for lock exclusion, lock continuity during compaction, kill release, interrupted compaction, and native Windows sharing violations. |
| [`src/kv/recovery_1m_bench_test.go`](src/kv/recovery_1m_bench_test.go) | Created | Cold recovery benchmark and test for 1,000,000 keys. |
| [`torture/types.go`](torture/types.go) | Created | Data models for operations, records, crash hooks, configuration, and audit reports. |
| [`torture/oracle.go`](torture/oracle.go) | Created | Independent persistent ACK oracle with disk journal, indeterminate operation handling, and full DB validation. |
| [`torture/worker.go`](torture/worker.go) | Created | Child worker process loop serving commands over stdin/stdout pipes. |
| [`torture/runner.go`](torture/runner.go) | Created | Subprocess lifecycle manager, random crash scheduler, and multi-writer coordinator. |
| [`torture/main.go`](torture/main.go) | Created | CLI interface for torture testing with `--seed`, `--ops`, `--crashes`, and report formatting. |
| [`torture/torture_test.go`](torture/torture_test.go) | Created | Acceptance test suite verifying all 15 Phase 6 criteria. |
| [`HANDOFF_MEMBER2.md`](HANDOFF_MEMBER2.md) | Created | Architecture, recovery protocol, compaction limits, and integration guide for Member 1. |
| [`MEMBER2_FINAL_TECHNICAL_EVIDENCE_REPORT.md`](MEMBER2_FINAL_TECHNICAL_EVIDENCE_REPORT.md) | Created | This complete final technical evidence report. |

---

## 4. Transaction and Recovery Implementation

Sequential transaction scanning begins at byte offset 32. Each transaction format:
`[TxnHeader: 32B] [Operations Payload: N Bytes] [CommitFooter: 32B]`

Validation criteria before committing to `Index`:
1. `TxnHeader.Magic == "TXN1"`
2. `TxnHeader.TxID == CommitFooter.TxID`
3. `CommitFooter.TotalTxBytes == 32 + PayloadLen + 32`
4. `TxnCRC == crc32.Checksum(TxnHeader || Payload)`
5. `FooterCRC == crc32.Checksum(Footer[0..27])`

If any check fails or EOF is encountered before the footer, recovery truncates the file back to the last valid committed boundary and flushes with `SyncFile()`.

---

## 5. F4 One-Byte Truncation Correction

### Flaw in Previous Test
The old test called `Put("first", "v1")` and `Put("second", "v2")`. Both returned success (acknowledged). It then truncated 1 byte and asserted that losing `"second"` was acceptable. This was invalid because acknowledged data must never be lost.

### Corrected Test Implementation
`TestRecovery_OneByteTruncation_F4` in `src/kv/recovery_test.go`:
1. Calls public `Put` for `"first"` -> `"value1"` and `"second_ack"` -> `"value2"`. Verifies success.
2. Closes database and records exact file size ($S_{\text{ack}}$).
3. Directly writes an unacknowledged trailing transaction (TxID 3) containing key `"unack_in_flight_key"` directly to the raw file handle without calling `Put` or acknowledging to any caller.
4. Truncates exactly 1 byte from that unacknowledged trailing transaction.
5. Reopens database via `Open()`:
   - `"first"` survives with exact value `"value1"`.
   - `"second_ack"` survives with exact value `"value2"`.
   - `"unack_in_flight_key"` returns `ErrNotFound`.
   - `info.LiveKeyCount == 2` and `info.RecoveredTransactions == 2`.
   - `info.RepairedTailBytes` matches the exact size of the truncated unacknowledged transaction.
   - Database accepts subsequent writes.

### Repeated Execution Proof:
```powershell
& "C:\Users\sanke\Downloads\go\bin\go.exe" test -v -run TestRecovery_OneByteTruncation_F4 -count=20 ./src/kv
# Result: 20/20 PASS in 0.688s
```

---

## 6. Compaction Algorithm & Dual Limits

Compaction executes under `db.mu` via `compactLocked()`:
1. Opens `<path>.compact.tmp`.
2. Writes 32-byte `FileHeader`.
3. Snapshots live keys from `db.index.Keys()` in deterministic sorted order.
4. Bounds streaming batches by:
   - **Max Operations:** `1,000` operations per transaction.
   - **Max Encoded Payload:** `16 MiB` (`16,777,216` bytes).
5. Checks `opBytes := 7 + len(key) + len(val)` before reading values:
   - If `opBytes > 16 MiB`, returns `ErrInvalid` immediately without touching database files.
   - Flushes batch before adding an operation if it would exceed 1,000 ops or 16 MiB.
6. Calls `SyncFile(tmpFile)` to enforce physical durability.
7. Closes `tmpFile` and `db.file`.
8. Calls `os.Rename(tmpPath, db.path)`.
9. Reopens `db.path` and atomically updates `db.file`, `db.index`, `db.committedEnd`, and `db.nextTxID`.

---

## 7. Memory-Bound Evidence

On 32-bit Go (`windows/386`), available user virtual memory is limited to ~2-3 GB. Unbounded compaction batching would cause Out-Of-Memory termination.

Dual bounding guarantees:
$$\text{Max RAM per Batch} \le 16\text{ MiB payload} + 32\text{B header} + 32\text{B footer} + \mathcal{O}(1,000 \times \text{IndexEntry})$$
$$\text{Max RAM per Batch} \approx 16.1\text{ MB}$$

Verified by tests:
- `TestCompaction_ByteBoundarySplitting`: Splits batches deterministically when exceeding configured byte limits.
- `TestCompaction_SingleMaxSizedValue`: Successfully compacts maximum legal 1 MB values.
- `TestCompaction_MultipleLargeValues`: Reclaims space from deleted 500 KB values without OOM.
- `TestCompaction_OperationIndividuallyTooLarge`: Rejects operations exceeding batch byte policy cleanly with `ErrInvalid`.
- `TestCompaction_ManySmallValuesOpCountLimit`: Splits 500 small keys across 50-op transaction chunks.

---

## 8. Crash-Safety Analysis Across Platforms

| Crash Scenario | Linux Behavior | macOS Behavior | Windows Behavior | Runtime Tested? |
| :--- | :--- | :--- | :--- | :--- |
| **Kill while writing `.compact.tmp`** | Original file untouched; restart removes `.compact.tmp`. | Original file untouched; restart removes `.compact.tmp`. | Original file untouched; handles closed by kernel; restart removes `.compact.tmp`. | **VERIFIED BY RUNTIME TEST** (`TestSubprocess_4`) |
| **Kill after `SyncFile`, before rename** | Original file intact; synced tmp file discarded on reopen. | Original file intact; synced tmp file discarded on reopen. | Original file intact; synced tmp file discarded on reopen. | **VERIFIED BY RUNTIME TEST** (`TestSubprocess_5`) |
| **Kill after rename, before reopen** | Destination inode swapped; new file recovered on restart. | Destination inode swapped; new file recovered on restart. | Destination file swapped; new file recovered on restart. | **VERIFIED BY RUNTIME TEST** (`TestSubprocess_6`) |
| **Kill during rename** | Filesystem journal completes or rolls back inode link. | APFS metadata update is atomic. | NTFS MFT record pointer update is atomic. | **VERIFIED BY DOCUMENTATION ONLY** |
| **Destination file held open by external handle** | Linux permits `rename(2)` over open destination. | macOS permits `rename(2)` over open destination. | Win32 `MoveFileExW` fails with `ERROR_SHARING_VIOLATION` / `Access is denied`. Handled safely: engine reopens original file, reports error, continues serving reads/writes. | **VERIFIED BY RUNTIME TEST** (`TestSubprocess_7B`) |

---

## 9. Process-Lock Continuity

The advisory process lock is held on `<path>.lock` using `LockFileEx` (exclusive, non-blocking).

Because `<path>.lock` is completely separate from `<path>`:
1. When compaction closes `<path>` before calling `os.Rename`, `<path>.lock` remains held.
2. A second process attempting `Open()` during the descriptor swap receives `ErrBusy`.
3. Verified by `TestSubprocess_2_LockContinuityDuringCompaction`: second process blocked while child is paused at stage `after_sync`.

---

## 10. Recovery & Compaction Test Results

Executing uncached full test suite:
```powershell
& "C:\Users\sanke\Downloads\go\bin\go.exe" test -v -count=1 ./src/kv
```

### Results Summary:
- Total unit & subprocess tests: **55 tests**
- Passed: **55**
- Failed: **0**
- Execution Time: **5.777s** (including 1M-key benchmark)

---

## 11. Independent Torture Harness (`torture/`)

Implemented in package `main` under `torture/`:
- **Out-of-Process Worker:** The database runs in a separate subprocess (`--worker`).
- **Independent ACK Oracle:** Tracks confirmed mutations in memory and flushes an append-only journal (`oracle.log`) with `Sync()` prior to kills.
- **Ambiguous In-Flight Handling:** Interrupted in-flight mutations are categorized as `INDETERMINATE`. Batches are verified for All-or-Nothing atomicity (zero partial batches).
- **Forced Kill Injection:** Uses OS `cmd.Process.Kill()` (`TerminateProcess`).
- **Reproducibility:** Seed and full operation parameters are printed; any run can be re-executed via `--seed <id>`.

### Acceptance Test Results (`go test -v -count=1 ./torture`):
```text
=== RUN   TestWorkerProcess                                    --- PASS (0.00s)
=== RUN   TestTorture_1_SeedReproducibility                    --- PASS (0.00s)
=== RUN   TestTorture_2_OracleRecordsOnlyObservedAcks          --- PASS (0.00s)
=== RUN   TestTorture_3_UnknownInFlightOperationsHandled       --- PASS (0.01s)
=== RUN   TestTorture_4_5_LaunchAndForcedTermination           --- PASS (0.07s)
=== RUN   TestTorture_6_7_FreshProcessRecoveryAndAckSurvival   --- PASS (0.03s)
=== RUN   TestTorture_8_IncompleteBatchesNotPartiallyVisible   --- PASS (0.00s)
=== RUN   TestTorture_9_10_ConcurrentWritersAndReaders         --- PASS (0.15s)
=== RUN   TestTorture_11_12_RandomCrashCyclesAndRecovery       --- PASS (0.15s)
=== RUN   TestTorture_13_FailureOutputIncludesReproducibleSeed --- PASS (0.00s)
=== RUN   TestTorture_14_OracleIsIndependent                   --- PASS (0.00s)
=== RUN   TestTorture_15_DetectsDeliberateDataLoss             --- PASS (0.00s)
PASS
ok      github.com/madhuharshag/Fsync-or-Die/torture    1.084s
```

### CLI Torture Stress Run:
```powershell
& "C:\Users\sanke\Downloads\go\bin\go.exe" run ./torture -seed 99999 -ops 1000 -crashes 10 -writers 4
```
```text
================================================================
 FSYNC-OR-DIE — INDEPENDENT TORTURE HARNESS
================================================================
Seed:              99999
Target Ops:        1000
Planned Crashes:   10
Writers:           4
Database Path:     C:\Users\sanke\AppData\Local\Temp\torture_run_1703802743\torture.db
Oracle Log:        C:\Users\sanke\AppData\Local\Temp\torture_run_1703802743\oracle.log
----------------------------------------------------------------

[PASS] TORTURE TEST COMPLETED SUCCESSFULLY
================================================================
 TORTURE TEST AUDIT REPORT
================================================================
Status:             true
Verification Msg:   All acknowledged mutations survived. Zero partial batches. Full integrity confirmed.
Seed:               99999 (use --seed 99999 to reproduce)
Go Version:         go1.27.3
Host OS/Arch:       windows/386
Duration:           1.6038173s
Operations Sent:    900
Ops Acknowledged:   690
Ops Indeterminate:  0
Ops Rejected:       0
Forced Crashes:     10
Successful Reopen:  10
================================================================
```

---

## 12. One-Million-Key Recovery Benchmark

Executed via `TestRecovery_1MillionKeys_ColdOpen`:

```text
================================================================
 1,000,000-KEY COLD RECOVERY BENCHMARK RESULTS
================================================================
Environment:          windows / 386 (Go go1.27.3)
CPUs Available:       24
Database File Size:   25.76 MB (27,006,432 bytes)
Recovered Txns:       100
Recovered Live Keys:  1000000
Internal Timer:       461.536 ms (461536500 ns)
Wall-Clock Elapsed:   461.5365ms (0.462 seconds)
Memory Allocated:     88.45 MB (92,742,272 bytes)
Meets < 2.0s Goal:    true
================================================================
```

---

## 13. Full Verification Suite Summary

| Check | Command Executed | Result | Notes |
| :--- | :--- | :--- | :--- |
| **Code Formatting** | `go fmt ./...` | **PASS** | Clean formatting across all packages. |
| **Static Analysis** | `go vet ./...` | **PASS** | 0 warnings, 0 errors. |
| **Race Detector** | `go test -race ./...` | **UNSUPPORTED** | Go runtime reports `-race is not supported on windows/386`. Documented honestly. |
| **Unit Tests (`src/kv`)** | `go test -count=1 ./src/kv` | **PASS** (55/55) | 5.777s total. |
| **Torture Tests (`torture`)** | `go test -count=1 ./torture` | **PASS** (12/12) | 0.907s total. |
| **Repeated Focused Runs** | `go test -run 'TestCompaction\|TestRecovery' -count=20 ./src/kv` | **PASS** (20/20) | 13.732s total. |
| **Torture CLI Tool** | `go run ./torture [args]` | **PASS** | 10 forced crashes verified with ACK oracle. |

---

## 14. Known Limitations & Honest Disclosures

1. **Host Go Toolchain Architecture (`windows/386`):**
   - The development machine is a 64-bit Windows OS running a 32-bit Go compiler (`windows/386`).
   - Although 32-bit pointers conserve memory (88 MB for 1M keys), address space is limited to ~2-3 GB.
   - `-race` is unsupported on `windows/386`.
2. **True Power-Loss Testing:**
   - Tests prove resilience against process termination (`kill -9`, `TerminateProcess`), unacknowledged torn tails, and interrupted replacement.
   - Host drive write-cache power-loss survival depends on drive firmware adherence to flush commands and cannot be proven by software unit tests alone.
3. **Scenario E5 (Tombstone at 100% Capacity):**
   - Appending a `DeleteOp` tombstone requires $\ge 72$ bytes. If the database reaches 100% capacity, `CheckCapacity` returns `ErrFull`.
   - Modifying this requires Member 1's explicit approval.

---

## 15. Integration Instructions for Member 1

1. **Review Changed Files:**
   - Inspect [`src/kv/compaction.go`](src/kv/compaction.go), [`src/kv/subprocess_test.go`](src/kv/subprocess_test.go), [`src/kv/recovery_test.go`](src/kv/recovery_test.go), [`src/kv/recovery_1m_bench_test.go`](src/kv/recovery_1m_bench_test.go), and [`torture/`](torture/).
   - Confirm that shared contracts (`src/kv/contract.go`, `src/kv/format.go`) remain untouched.
2. **Execute Full Test Suite:**
   ```powershell
   go test -v -count=1 ./src/kv
   go test -v -count=1 ./torture
   go test -v -run TestRecovery_1MillionKeys_ColdOpen ./src/kv
   go run ./torture -seed 12345 -ops 500 -crashes 5 -writers 2
   ```

---

## 16. Git Delivery Information

- **Assigned Branch:** `feat/transactions-recovery`
- **Starting Commit:** `3057b6636edc62d92582685fd4afa943e98f4678`
- **Working Tree:** All code, benchmarks, tests, and documentation are ready for commit on `feat/transactions-recovery`.
- **Remote Push:** Pushing to remote `origin feat/transactions-recovery` will be executed per Phase 11 instructions.
