# Fsync-or-Die — Member 1 Technical Handoff
**Role:** Member 1 — Tech Lead + Core Storage Engineer  
**Branch:** `feat/storage-core`  
**Package:** `github.com/madhuharshag/Team-Innovexa/src/kv` (`kv`)  
**Status:** IMPLEMENTED AND EXECUTED (Darwin arm64 — 30 top-level unit/integration tests PASS); CROSS-COMPILED (Linux amd64/arm64, Windows amd64); RUNTIME PENDING WSL2 VERIFICATION.

---

## 1. Overview & Ownership Boundaries

Member 1 owns the database filing system foundation:
- **On-Disk File Format & Binary Codec:** Little-endian serialization for File Header, Transaction Header, Operation records, and Commit Footer with dual CRC32-IEEE checksums (`format.go`).
- **Shared Public API & Error Taxonomy:** `Engine` interface, `Op`, `Kind`, `Options`, `OpenInfo`, `Stats`, and canonical error taxonomy (`contract.go`).
- **Low-Level Positioned I/O:** `ReadAtFull`, `WriteAtFull`, zero-progress detection, safe `TruncateFile`, and `SyncFile` error propagation (`io.go`).
- **In-Memory RAM Index:** Scale-oriented `Index` mapping byte-safe keys to `{ValueOffset, ValueLength}`, ordered pending-transaction publication, and index cloning (`index.go`).
- **Database Handle & Mutex Discipline:** Single global `sync.Mutex` on `DB`, lifecycle management (`Open`, `Close`, `Stats`, `Check`, `Scan`), process file locking (`lock_darwin.go`, `lock_linux.go`), and preflight capacity enforcement (`capacity.go`).
- **Entry Points:** Public `Get`, `Put`, `Delete`, and `Batch` delegating to the unified transaction path (`get.go`, `put_delete.go`).

---

## 2. On-Disk Binary Format (Little-Endian)

### File Header (32 bytes at offset 0)
```
Offset  Size  Field
0       8     Magic: "KIPDB001" (ASCII)
8       4     Version: 1 (uint32)
12      4     Header Size: 32 (uint32)
16      8     Max File Bytes: uint64 (0 = unlimited)
24      4     Flags: uint32 (0)
28      4     CRC32-IEEE over bytes [0..27]
```

### Transaction Header (32 bytes at start of each transaction)
```
Offset  Size  Field
0       4     Magic: "TXN1" (ASCII)
4       2     Version: 1 (uint16)
6       2     Flags: uint16 (0)
8       8     TxID: uint64
16      4     OpCount: uint32 (1..10000)
20      4     Reserved: uint32 (0)
24      8     PayloadLen: uint64 (bytes)
```

### Operation Layout (inside Transaction Payload)
```
[Kind: 1B][KeyLen: 2B][ValLen: 4B][Key bytes][Val bytes]
```
- `Kind`: `1` = `PutOp`, `2` = `DeleteOp`.
- `KeyLen`: 1 to 1,024 bytes.
- `ValLen`: 0 to 1,048,576 bytes (1 MB).
  - For `DeleteOp`: `ValLen` **must be 0**.
  - For `PutOp`: `ValLen` may be 0 (empty value, distinct from deletion).
- Encoded size per operation: `7 + len(key) + len(value)` bytes.

### Commit Footer (32 bytes at end of transaction)
```
Offset  Size  Field
0       8     Magic: "COMMIT01" (ASCII)
8       8     Matching TxID: uint64
16      8     TotalTxBytes: uint64 (64 + PayloadLen)
24      4     TxnCRC: uint32 over (TxnHeader 32B + Payload bytes)
28      4     FooterCRC: uint32 over Footer bytes [0..27]
```

---

## 3. Integration Guide for Member 2 (Transactions & Recovery)

### Common Transaction Path Invariants:
1. **Single Append Protocol:** `Put`, `Delete`, and `Batch` share the exact same transaction envelope:
   $$\text{Total Transaction Bytes} = 64 + \sum (7 + \text{len(key)} + \text{len(val)})$$
2. **Mutex Calling Convention:** Internal transaction commit helpers (`commitBatchLocked`) are invoked with `db.mu` **already held**. **Do NOT re-acquire `db.mu`** inside internal transaction methods.
3. **No Intermediate Index Publication:** Never apply updates to `db.index` until `SyncFile(db.file)` has successfully returned.
4. **Crash Semantics:** A complete valid transaction (including valid matching footer and dual CRCs) found during recovery is valid history even if the prior process crashed before returning success to the caller or before the CLI printed ACK. Being killed during Sync does not by itself mean the transaction is absent; if footer bytes and valid dual CRCs reached disk before termination, recovery restores the transaction. Incomplete final transactions are cleanly truncated and discarded.
5. **Tail Repair:** When recovery encounters an incomplete or torn tail, `db.file` is truncated to `committedEnd` and synced before new writes are accepted. If in read-only mode, the handle is poisoned.
6. **Compaction Protocol (Coordinated Ownership):**
   Member 2 owns `Compact()`. For the tested Linux/WSL2 target, compaction must follow this exact sequence:
   1. Acquire `db.mu.Lock()` **before** choosing snapshot state, and hold it through the entire snapshot build and publication (no reads or writes may enter during compaction; keep stable ownership lock held).
   2. Verify handle state (`checkStateLocked()`).
   3. Create a same-directory temporary file (`path + ".compact.tmp"`).
   4. Write complete immutable file header (`NewFileHeader`).
   5. Copy current live values into valid bounded snapshot transactions.
   6. Build replacement index with replacement-file offsets.
   7. Finish and `SyncFile(tmpFile)`.
   8. Rename temporary directly over authoritative pathname (`os.Rename`).
   9. Synchronize parent directory according to documented supported-platform policy.
   10. Retain the already-open temporary descriptor as the new active descriptor (avoiding reopen failure window), or re-open consistently.
   11. Switch active descriptor, index (`db.index.ReplaceWith`), committed end, and next TxID under the lock.
   12. Close old descriptor after successful publication.
   13. Unlock `db.mu`.
   *(Do NOT close old descriptor before rename. Never delete authoritative pathname first. Never append through old descriptor after publication. If rename succeeds and a later step fails, reconcile active state with the published file and do not delete the newly published database).*

---

## 4. Integration Guide for Member 3 (Torture Harness)

1. **Independent Oracle:** Do not import `src/kv` internal structs; model state exclusively through external CLI requests and observed outputs.
2. **Termination Points:** Test process termination (`SIGKILL`) during active writes, during sync, and during crash recovery.
3. **Correctness Criteria:**
   - Every acknowledged operation (`ACK <id>`) must survive recovery.
   - In-flight operations may either be completely present or completely absent. **No partial batches or torn values are permitted.**
4. **Request/ACK vs Commit Ordering:** Request/ACK ordering cannot be assumed to equal commit ordering under concurrency.
5. **Binary Data Boundaries:** Exercise keys with null bytes (`\x00`), high bytes (`\xFF`), max-length keys (1,024 bytes), empty values (`""`), and values up to 1 MB.

---

## 5. Integration Guide for Member 4 (CLI & Serve)

### Public API Usage:
```go
import "github.com/madhuharshag/Team-Innovexa/src/kv"

// Open database
engine, info, err := kv.Open("data.kip", kv.Options{
    MaxFileBytes: 100 * 1024 * 1024,
})
if err != nil {
    // Handle error
}
defer engine.Close()

// Recovery duration is exposed directly:
fmt.Printf("Recovered %d txns in %d ns\n", info.RecoveredTransactions, info.RecoveryDurationNs)

// Single Put
err = engine.Put([]byte("key"), []byte("value"))
if err == nil {
    // Emit ACK only after nil error
    fmt.Printf("ACK %d\n", opID)
}

// Single Get
val, err := engine.Get([]byte("key"))
if errors.Is(err, kv.ErrNotFound) {
    // Standalone CLI get command output:
    fmt.Println("NOTFOUND")
} else if err != nil {
    // I/O or database error
} else {
    // Found value
    fmt.Println(string(val))
}

// Atomic Batch
err = engine.Batch([]kv.Op{
    {Kind: kv.PutOp, Key: []byte("k1"), Value: []byte("v1")},
    {Kind: kv.DeleteOp, Key: []byte("k2")},
})
```

### Critical CLI Rules:
- **ACK Rule:** Print exactly `ACK <op-id>\n` replacing `<op-id>` with the real submitted request ID. Do not print an empty ACK. Emit ACK **only** after `engine.Put` or `engine.Batch` returns `nil`.
- **Missing Get Output:** Standalone `get <key>` prints the exact value followed by newline on success, or exactly `NOTFOUND\n` on missing key.
- **No Automatic Retry on Uncertain:** If an operation returns `kv.ErrUncertain`, the CLI must **not** automatically retry, as the transaction may have already reached disk.

---

## 6. Capacity & Full-Cap Deletion Gap

- **Tombstone Overhead:** An append-only delete tombstone consumes:
  $$\text{Tombstone Bytes} = 64 + 7 + \text{keyLength} \ge 72 \text{ bytes}$$
- **Full-Cap Gap:** If the log reaches 100% capacity ($remaining < 64 + 7 + \text{keyLength}$), an append-only `Delete` will be rejected with `ErrFull`.
- **Proposed Coordinated Fallback:** Build a separate complete post-operation snapshot, sync it, and publish it through replacement, preserving the original authoritative file until publication.
- **Status:** **INCOMPLETE / PENDING MEMBER 2 COORDINATION**.

---

## 7. Supported Platforms Policy

- **macOS (Darwin ARM64 / Apple Silicon):** Supported for local development via `lock_darwin.go` using `syscall.Flock`.
- **Linux (x86_64 / ARM64, Ubuntu under WSL2):** Primary integration and demonstration target via `lock_linux.go` using `syscall.Flock`.
- **Native Windows:** **EXPLICITLY UNSUPPORTED**. Native Windows execution without POSIX file replacement and flock semantics is rejected with `ErrConfig`.

---

## 8. Verified Test & Benchmark Details

- **Test Suite:** Exactly 30 top-level test functions (`go test -list '^Test' ./src/kv`), all passing under `go test -count=1 -v ./src/kv`.
- **Sync Failure Verification:** `TestDB_InjectedSyncFailureOnTransactionPath` injects a sync error on the real transaction call path via `rawDB.syncSeam`. Mutation returns `ErrUncertain` (never nil/success), poisons handle (`db.poisoned`), and rejects subsequent operations with `ErrPoisoned`.
- **Benchmark Arithmetic:**
  - `BenchmarkDB_Put`: Single fixed key (9B) and value (19B). Encoded transaction size is exactly $64 + 7 + 9 + 19 = 99$ bytes.
  - `BenchmarkDB_Batch100`: 100 operations. Keys `k0`..`k9` (2B) + `k10`..`k99` (3B); values `v0`..`v9` (2B) + `v10`..`v99` (3B). Total transaction size is exactly $64 + 10 \times (7+2+2) + 90 \times (7+3+3) = 64 + 110 + 1170 = 1344$ bytes.
  - Measured on Apple M4 (macOS Darwin arm64) with `File.Sync` enabled. Local microbenchmarks do not prove physical SSD power-loss safety or 1M-key recovery speed.
- **Timing Measurement:** `OpenInfo.RecoveryDurationNs` measures the elapsed wall-clock nanoseconds from entering `Open()` through file opening, header validation, transaction recovery replay, tail repair, and index population.
- **Scalability Status:** The in-memory index stores 64-bit offsets and 32-bit lengths rather than values. Full dataset scaling to 1,000,000 keys is supported by design, but timed recovery under 2 seconds remains unverified until measured on the judging laptop.
