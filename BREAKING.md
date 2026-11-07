# TEAM INNOVEXA — FSYNC-OR-DIE
# BREAKING.MD — ADVERSARIAL FAILURE ANALYSIS & CRASH-SAFETY PROOF

**Project:** Fsync-or-Die (`kip`)  
**Tagline:** *“Don’t Trust the Process. Trust the Proof.”*  
**Repository:** [github.com/madhuharshag/Team-Innovexa](https://github.com/madhuharshag/Team-Innovexa)  
**Lead Author:** Member 1 — Tech Lead + Core Storage Engineer + Technical Verification Lead  
**Document Status:** Release Authoritative (`main`)  
**Audience:** Hackathon Technical Judges, Database Architects & Systems Engineers  

---

## 1. The Threat & Failure Model

A crash-resilient database must clearly delineate what failure modes it guarantees against versus what hazards reside outside its failure boundary.

```
+-------------------------------------------------------------------------------+
|                        VERIFIED WITHIN SYSTEM MODEL                           |
|  - Process Termination (kill -9, SIGKILL, Win32 TerminateProcess)             |
|  - Mid-Instruction CPU Halts & Out-Of-Memory (OOM) Kills                      |
|  - Incomplete / Torn Transaction Headers, Payloads, or Commit Footers        |
|  - Interrupted Atomic Batches (1 to 10,000 Operations)                        |
|  - Interrupted Online Compaction (before, during, and after rename)          |
|  - Repeated Crash & Interrupted Recovery Cycles                               |
|  - Conflicting Concurrent Process Access (Multi-Process Exclusion)            |
+-------------------------------------------------------------------------------+
                                        |
                                        v
+-------------------------------------------------------------------------------+
|                      ASSUMPTIONS & PHYSICAL BOUNDARIES                        |
|  - Storage hardware / drive controller honors fsync / FlushFileBuffers        |
|  - Underlying OS kernel filesystem maintains metadata consistency             |
|  - Non-volatile storage media does not suffer physical sector decay / bit rot|
+-------------------------------------------------------------------------------+
```

### Explicit Distinctions:
1. **Process Termination (`kill -9` / `TerminateProcess`):** The process dies abruptly at an arbitrary instruction. Operating system page caches remain alive; file descriptors are automatically closed by the kernel. **Fully covered and verified.**
2. **Operating System Crash / Kernel Panic:** The OS kernel halts immediately. Dirty pages in kernel memory that have not been flushed via `fsync` are lost. **Fully covered: our write path never acknowledges an operation before `SyncFile` completes.**
3. **Interrupted / Short File Writes:** Storage subsystems occasionally write fewer bytes than requested before stalling or returning an error. **Fully covered: `io.WriteAtFull` loops with short-write detection and traps zero-progress stalls.**
4. **Torn Writes (Block Fracturing):** If power fails while the drive is writing a physical 4KB sector, bytes may be half-written. **Fully covered: dual IEEE CRC32 checksums (`TxnCRC` and `FooterCRC`) detect torn records down to a single flipped bit.**
5. **Physical Power Loss (Pulling the Plug):** If the host loses AC power, all volatile RAM disappears. If the drive controller possesses an unbuffered write cache that lies about flush completion, data can be lost. **Honest statement: software cannot prevent drive firmware deceit. Under any hardware that correctly implements write-barrier flush semantics, our engine guarantees durability.**

---

## 2. Transaction Failure Points (P1 through P7)

Every mutation executed in [`src/kv/put_delete.go`](file:///Users/madhuharsha/Core-Storage/Team-Innovexa/src/kv/put_delete.go) traverses 7 distinct execution stages. Here is the exact analysis of what occurs if a crash strikes at each point:

```
[Start] --> P1 --> [Write TxnHdr] --> P2 --> [Write Payload] --> P3 --> [Write Footer] --> P4 --> [SyncFile] --> P5 --> [Publish Index] --> P6 --> [Return ACK] --> P7
```

### P1: Crash Prior to Disk Write
- **Execution State:** Input has been validated, memory buffers constructed; no bytes written to disk.
- **On-Disk State:** The authoritative log file is unchanged; physical length equals `db.committedEnd`.
- **Recovery Observation:** Recovery scans to `committedEnd` and finishes cleanly.
- **Client Perception:** The client received no acknowledgement (connection severed).
- **Correctness:** **Preserved.** The operation never took effect.

### P2: Crash During Transaction Header Write
- **Execution State:** `WriteAtFull` is partially executed. Between 1 and 31 bytes of `TxnHeader` exist past `db.committedEnd`.
- **On-Disk State:** Authoritative file has partial bytes at EOF.
- **Recovery Observation:** Recovery reads offset `committedEnd`, sees $\text{fileSize} - \text{offset} < 32$, recognizes an incomplete header, and terminates the scan.
- **Recovery Action:** Truncates file back to `committedEnd`, syncs file, and restores clean state.
- **Client Perception:** Never acknowledged. Operation safely discarded.

### P3: Crash During Operation Payload Write
- **Execution State:** Full 32-byte `TxnHeader` was written; payload writing was interrupted.
- **On-Disk State:** Valid header followed by partial payload bytes; `CommitFooter` is absent.
- **Recovery Observation:** Recovery validates `"TXN1"`, but bounds check fails ($\text{fileSize} - \text{offset} < 32 + \text{PayloadLen} + 32$). The scan terminates.
- **Recovery Action:** Truncates file back to `committedEnd`.
- **Client Perception:** Never acknowledged. Operation safely discarded.

### P4: Crash During Commit Footer Write
- **Execution State:** Header and payload are fully written; `CommitFooter` write was interrupted (e.g. 10 of 32 bytes written).
- **On-Disk State:** Header, payload, and torn footer at EOF.
- **Recovery Observation:** Recovery reads partial footer. Magic `"COMMIT01"` or `FooterCRC` validation fails.
- **Recovery Action:** The transaction is recognized as uncommitted. File is truncated to `committedEnd`.
- **Client Perception:** Never acknowledged. Zero corruption of previous records.

### P5: Crash During Physical Synchronization (`SyncFile`)
- **Execution State:** `WriteAtFull` completed; bytes reside in OS dirty page buffers; drive has not finalized flush.
- **On-Disk State:** Dependent on how many dirty pages reached magnetic/flash cells before power cut.
- **Recovery Observation:** If dirty pages were lost, recovery encounters an incomplete tail or bad CRC and truncates to `committedEnd`. If all pages reached disk prior to crash, recovery validates the dual CRCs and recovers the transaction.
- **Client Perception:** Never acknowledged. The client received an error or severed connection.
- **Correctness:** Indeterminate in-flight operation; recovery safely settles either to clean roll-forward or clean truncation.

### P6: Crash After Successful `SyncFile`, Before Return / ACK
- **Execution State:** `SyncFile` returned `nil`. Data is 100% physically durable on non-volatile media. Process killed before emitting `ACK <key>` to stdout.
- **On-Disk State:** Contiguous, fully checksummed, durable transaction frame at EOF.
- **Recovery Observation:** Recovery validates header, payload, footer, and dual CRCs. Replays transaction into RAM index.
- **Client Perception:** Client saw an abrupt disconnection.
- **Correctness:** **Roll-Forward Preserved.** The database chose durability over data loss. Acknowledging client failure while persisting durable data is the classic distributed two-phase commit uncertainty boundary.

### P7: Crash After Acknowledgement (`ACK`) Returned
- **Execution State:** Success returned to client; CLI emitted `ACK <key>`. Process terminated subsequently.
- **On-Disk State:** Contiguous, fully synced transaction frame.
- **Recovery Observation:** Validated and replayed into RAM index with 100% fidelity.
- **Client Perception:** Operation confirmed.
- **Correctness:** **DURABLE.** Permanent zero-loss guarantee satisfied.

---

## 3. Atomic Batch Failure Scenarios

Atomic batches ($1 \le N \le 10,000$ operations) are encapsulated into a **single transaction frame**:
$$\text{TxnHeader (OpCount}=N) \parallel \text{All } N \text{ Operations} \parallel \text{CommitFooter}$$

### The All-or-Nothing Guarantee:
- **No Intermediate Commits:** The engine never writes intermediate commit markers between operations in a batch.
- **Single Checksum Unit:** `TxnCRC` covers the entire batch payload. If even one byte of operation 9,999 is missing or corrupted, the CRC check fails for the entire frame.
- **Recovery Replay Invariant:** Recovery decodes operations into a temporary buffer `updates`. Only after **all** $N$ operations are decoded without error and $\text{count(updates)} == \text{txnHeader.OpCount}$ does recovery apply them to the index via `db.index.ApplyUpdates()`.
- **Empirical Evidence:** In `TestDB_BatchAtomicityOnCrash` and `TestTorture_BatchAtomicity`, out-of-process workers killed mid-batch never leave a database with partial operations.

---

## 4. Recovery Interruption Scenarios

What happens if recovery itself crashes while repairing a damaged file?

1. **ReadOnly Mode:** `Open(path, Options{ReadOnly: true})` performs **zero file writes**. If an uncommitted torn tail is detected, it returns `ErrCorrupt` without modifying a single byte on disk.
2. **Read-Write Recovery:** Recovery reads the log sequentially. It modifies disk state **only at the very end**, when calling `TruncateFile(offset)` and `SyncFile()`.
3. **Crash During Truncation:** If a crash halts the system during `TruncateFile()`, the physical file remains either at its old length or is partially truncated.
4. **Subsequent Recovery Idempotence:** When the database is opened again, recovery repeats the exact same sequential scan from offset 32. Because valid transactions before `offset` were never touched, recovery arrives at the exact same valid offset and repeats the truncation cleanly. Recovery is **100% idempotent**.

---

## 5. Compaction Interruption Scenarios

Compaction replaces the log file without stopping concurrent read operations:

| Compaction Stage | Failure Event | Authoritative File (`<path>`) | Staging File (`<path>.compact.tmp`) | Recovery on Next `Open()` |
| :--- | :--- | :--- | :--- | :--- |
| **Stage 1: Streaming live keys** | Process killed (`kill -9`) | 100% Intact and authoritative | Incomplete / torn | Detects and removes `.compact.tmp`. Original file intact. Zero loss. |
| **Stage 2: Staging file sync** | Sync fails or process dies | 100% Intact and authoritative | Fully written, unsynced | Detects and removes `.compact.tmp`. Original file intact. Zero loss. |
| **Stage 3: Descriptor pre-close** | Process killed before rename | 100% Intact and authoritative | 100% Synced, unrenamed | Detects and removes `.compact.tmp`. Original file intact. Zero loss. |
| **Stage 4: Atomic OS Rename** | Crash during `os.Rename` | Atomically points to old OR new file | Inode swapped or removed | Atomic metadata operation in OS. Clean open on whichever inode survived. |
| **Stage 5: Post-Rename reopen** | Crash after rename before swap | Newly installed compacted file | Does not exist | Opens new file; 100% live keys recovered. |

---

## 6. Capacity Exhaustion & Scenario E5 Policy

When `MaxFileBytes` is configured, database capacity is strictly governed by [`src/kv/capacity.go`](file:///Users/madhuharsha/Core-Storage/Team-Innovexa/src/kv/capacity.go):

### The Deletion Paradox (Scenario E5):
- In an append-only database, `Delete(key)` does not free disk space; it writes an immutable `DeleteOp` tombstone record.
- Appending a tombstone requires $71 + \text{len(key)}$ bytes.
- **The Question:** *What happens when the database is at 100% capacity (`committedEnd >= maxFileBytes`) and a client calls `Delete(key)`?*

### The Implemented Policy (Option A — Status Quo):
1. **Preflight Rejection:** `capacity.CheckCapacity()` calculates the required tombstone size. If $\text{committedEnd} + \text{totalTxBytes} > \text{maxFileBytes}$, it returns `ErrFull`.
2. **Zero In-Memory Modification:** The key is **not** deleted from the RAM index. The database state remains completely consistent.
3. **Zero Disk Bytes Appended:** File size does not grow by a single byte past the configured limit.
4. **Resolution Path:** The operator must run `kip compact <path>` to reclaim space occupied by overwritten versions and previous tombstones before new deletions can be appended.

*(For full architectural evaluation of alternative Option B Headroom reservation, see [`E5_CAPACITY_DELETION_POLICY_REVIEW.md`](file:///Users/madhuharsha/Core-Storage/Team-Innovexa/E5_CAPACITY_DELETION_POLICY_REVIEW.md)).*

---

## 7. Concurrency & Process-Locking Failures

1. **Process Mutual Exclusion:** Process locking uses a dedicated lock file (`<path>.lock`).
   - On Windows: `LockFileEx` with `LOCKFILE_EXCLUSIVE_LOCK | LOCKFILE_FAIL_IMMEDIATELY`.
   - On POSIX: `syscall.Flock(fd, syscall.LOCK_EX | syscall.LOCK_NB)`.
2. **Collision Behavior:** A second process attempting to open an already-locked database immediately receives sentinel error `ErrBusy` (exit code 1).
3. **Kernel Cleanup on Process Termination:** When a process holding `<path>.lock` is terminated by `kill -9` or Windows task termination, the operating system kernel automatically closes all open file handles and releases the advisory lock. The database is instantly available for subsequent clean opening.

---

## 8. Master Crash-Point Proof Table

| Scenario ID | Failure Event | Code Location | On-Disk State Prior to Recovery | Recovery Action Executed | Supporting Test | Verification Status |
| :-: | :--- | :--- | :--- | :--- | :--- | :---: |
| **F4.1** | 1-byte torn write past valid commit | `src/kv/db.go#L485` | Valid DB + 1 junk byte at EOF | Truncates 1 byte; replays valid txns | `TestRecovery_TornTailOneByteTruncation` | 🟢 VERIFIED |
| **F4.2** | Truncated TxnHeader (16 of 32B) | `src/kv/db.go#L380` | Valid DB + 16 partial header bytes | Bounds check fails; truncates 16B | `TestRecovery_IncompleteTxnHeaderAtEOF` | 🟢 VERIFIED |
| **F4.3** | Truncated payload bytes | `src/kv/db.go#L405` | Valid DB + Hdr + partial payload | Payload bounds check fails; truncates | `TestRecovery_TruncatedPayloadAtEOF` | 🟢 VERIFIED |
| **F4.4** | Corrupted payload byte (CRC mismatch) | `src/kv/db.go#L430` | Valid DB + full txn with 1 bit flip | `TxnCRC` mismatch; truncates torn txn | `TestRecovery_CorruptPayloadCRC` | 🟢 VERIFIED |
| **F4.5** | Corrupted CommitFooter byte | `src/kv/db.go#L436` | Valid DB + full txn with bad footer | `FooterCRC` mismatch; truncates txn | `TestRecovery_CorruptFooterCRC` | 🟢 VERIFIED |
| **F4.6** | Mismatched TxID in footer | `src/kv/db.go#L424` | Valid DB + txn with mismatched TxID | Validation check fails; truncates txn | `TestRecovery_TxIDMismatchDetection` | 🟢 VERIFIED |
| **F4.7** | Declared vs Decoded OpCount mismatch | `src/kv/db.go#L463` | Valid DB + txn with tampered OpCount | Decoded count != OpCount; truncates | `TestDB_OpCountMismatchDetection` | 🟢 VERIFIED |
| **F2.1** | Crash during recovery scan | `src/kv/db.go#L371` | Unchanged authoritative file | Recovery restarts from offset 32 cleanly | `TestRecovery_RepeatedReopenCycles` | 🟢 VERIFIED |
| **C1.1** | Crash while writing compaction temp | `src/kv/compaction.go` | Original DB intact; `.compact.tmp` exists | `Open()` deletes `.compact.tmp`; loads DB | `TestCompaction_InterruptedBeforeRename` | 🟢 VERIFIED |
| **C1.2** | Compaction atomic rename failure | `src/kv/compaction.go#L254` | Staged temp file; original untouched | Reopens original; deletes temp file | `TestCompaction_RenameFailureRollback` | 🟢 VERIFIED |
| **E1.1** | Write exceeding MaxFileBytes | `src/kv/capacity.go` | Database at capacity limit | Rejects with `ErrFull`; 0 bytes written | `TestCapacity_RejectsOverCapacityWrites` | 🟢 VERIFIED |
| **E5.1** | Delete at 100% capacity | `src/kv/put_delete.go` | Database at capacity limit | Rejects with `ErrFull`; index untouched | `TestCapacity_DeleteAt100Percent` | 🟢 VERIFIED |
| **G1.1** | Concurrent process collision | `src/kv/lock_windows.go` | Active `<path>.lock` held | Rejects second open with `ErrBusy` | `TestDB_SecondOpenReturnsErrBusy` | 🟢 VERIFIED |
| **G1.2** | Process kill with active lock | Kernel process table | `<path>.lock` descriptor orphaned | Kernel frees lock; next open succeeds | `TestSubprocess_LockReleasedOnKill` | 🟢 VERIFIED |
| **F1.1** | Random kill during multi-op stress | `torture/runner.go` | Arbitrary crash point (P1–P7) | Reopens, repairs, and passes Oracle check | `TestTorture_RandomCrashWorkload` | 🟢 VERIFIED |

---

## 9. Known Failure Boundaries & Honest Disclosures

To prevent misleading claims to judging panels:
1. **Drive Firmware Caching:** While `fsync` / `FlushFileBuffers` issues the physical cache flush command, consumer SSDs or virtualization layers that ignore flush barriers can experience data loss during an abrupt DC power cut.
2. **Read-Only Corruption Handling:** When a file with an uncommitted torn tail is opened with `ReadOnly: true`, the engine **does not mutate or truncate the file**. It returns `ErrCorrupt`. To repair the file, it must be opened in read-write mode.
3. **Capacity Full Recovery:** Compaction reclaims space only when the database contains dead or superseded records. If a database is filled with 100% unique live keys, compaction cannot reduce its size below `MaxFileBytes`.

---

*Document approved by Member 1 (Tech Lead, Team Innovexa).*
