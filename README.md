# Team Innovexa — Fsync-or-Die (`kip`)

[![Go Version](https://img.shields.io/badge/Go-1.23%2B-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Correctness](https://img.shields.io/badge/Correctness-Crash--Proof-success)](#7-data-durability-model)
[![Verification](https://img.shields.io/badge/Torture%20Campaign-1000%2F1000%20PASS-brightgreen)](#11-independent-torture-testing)
[![Release](https://img.shields.io/badge/Release-main%20(v1.0.0)-blue)](#18-technical-documentation-index)

> **“Don’t Trust the Process. Trust the Proof.”**

An embedded, single-process, crash-resilient key-value storage engine and command-line interface written in Go for the **"Kill It and Prove It"** 24-Hour Engineering Hackathon.

---

## 1. Project Identity & Team
- **Project Name:** Fsync-or-Die (`kip`)
- **Team Name:** Team Innovexa
- **Repository:** [github.com/madhuharshag/Team-Innovexa](https://github.com/madhuharshag/Team-Innovexa)
- **Primary Language:** Go 1.23+ (Standard Library Only; Zero Third-Party Storage Dependencies)
- **Target Environments:** Native Windows 11 (AMD64), Linux (Ubuntu / WSL2), macOS (Darwin ARM64)

---

## 2. Problem Statement
Traditional databases trade durability for write throughput by buffering mutations in operating system page caches. If power cuts or the kernel crashes, un-flushed writes disappear. Conversely, naive systems that issue uncontrolled disk flushes suffer from block fracturing, torn writes, and painfully slow recovery loops.

In a true adversarial testing environment where processes are killed abruptly (`kill -9`, `TerminateProcess`) at arbitrary microsecond intervals, databases frequently suffer from corrupted metadata, fractured records, or unrecoverable files.

---

## 3. What the Database Does
**Fsync-or-Die (`kip`)** provides high-performance, single-node embedded key-value storage with mathematical crash durability:
- Durably stores, retrieves, and deletes arbitrary binary keys (up to 1,024 bytes) and values (up to 1,048,576 bytes / 1 MB).
- Atomically commits multi-operation batches (up to 10,000 operations) with all-or-nothing rollback guarantees.
- Recovers 1,000,000 live keys into memory in **under 0.3 seconds** (291.83 ms median on Apple Silicon NVMe; 461.54 ms on Windows 11 NVMe).
- Self-heals uncommitted torn writes at end-of-file automatically on restart without administrative intervention.
- Reclaims dead space online through dual-bounded streaming compaction.

---

## 4. Key Technical Features
1. **Append-Only Write-Ahead Log:** Active data is never overwritten in place. Mutations are serialized to the end of a single authoritative file (`<path>.kip`).
2. **Dual IEEE CRC32 Checksums:** Every transaction is protected by two distinct checksums: `TxnCRC` (protecting transaction header and payload) and `FooterCRC` (protecting commit footer fields).
3. **Unconditional Physical Synchronization:** Success is returned and acknowledgements are emitted **only after** `File.Sync` (`fsync` / `FlushFileBuffers`) durably flushes non-volatile media.
4. **Scale-Ready 12-Byte In-Memory Index:** Keys map to compact 12-byte structs storing 64-bit file offsets and 32-bit lengths. Values remain on disk. Full dataset scaling to 1,000,000 keys requires only ~50 MB RAM.
5. **Advisory Process Mutual Exclusion:** Exclusive process locking via native Win32 `LockFileEx` (Windows) and POSIX `flock` (Linux/macOS) on `<path>.lock`. Second processes receive immediate `ErrBusy`.
6. **Dual-Bounded Compaction:** Streams live keys into a temporary replacement file in bounded chunks (1,000 ops / 16 MiB payload), protecting 32-bit address spaces.

---

## 5. Architecture Overview

```
+-------------------------------------------------------------------------------+
|                        CLI / Client Layer (cmd/kip)                          |
+-------------------------------------------------------------------------------+
                                        |
                                        v
+-------------------------------------------------------------------------------+
|                       Engine Contract (src/kv/contract.go)                   |
|   Put | Get | Delete | Batch | Check | Scan | Stats | Compact | Close       |
+-------------------------------------------------------------------------------+
       |                      |                         |
       v                      v                         v
+--------------+     +------------------+     +--------------------+
| Process Lock |     | In-Memory Index  |     |  Transaction Pipe  |
| <path>.lock  |     | 12B per Live Key |     | 10-Step Durability |
+--------------+     +------------------+     +--------------------+
                              |                         |
                              |   Single Authoritative  |
                              +--------> File <---------+
                                    (<path>.kip)
```

For complete architectural specifications, byte layouts, and mathematical proofs, see [`ARCHITECTURE.md`](ARCHITECTURE.md) and editable Mermaid source files in [`diagrams/`](diagrams/).

---

## 6. On-Disk Binary Layout (Little-Endian)
The authoritative database state resides in a single binary file (`<path>.kip`):

```
+-----------------------------------------------------------------------+
| File Header (32 Bytes): Magic "KIPDB001", Version, MaxFileBytes, CRC  |
+-----------------------------------------------------------------------+
| Transaction 1: Header (32B) -> Payload [Ops] -> Commit Footer (32B)   |
+-----------------------------------------------------------------------+
| Transaction 2: Header (32B) -> Payload [Ops] -> Commit Footer (32B)   |
+-----------------------------------------------------------------------+
| ...                                                                   |
+-----------------------------------------------------------------------+
```

- **TxnHeader (32B):** Magic `"TXN1"`, Version (1), Flags, TxID (uint64), OpCount (uint32), PayloadLen (uint64).
- **Operations:** Compact little-endian records: `[Kind: 1B][KeyLen: 2B][ValLen: 4B][Key][Value]`.
- **CommitFooter (32B):** Magic `"COMMIT01"`, TxID, TotalTxBytes, TxnCRC (IEEE CRC32), FooterCRC.

---

## 7. Data Durability Model
Our write path enforces a strict, verifiable 10-step sequence:
1. Input validation (key $\le 1024\text{B}$, val $\le 1\text{MB}$, batch $\le 10000$ ops).
2. Capacity preflight (`capacity.CheckCapacity`).
3. Construct `TxnHeader` (magic `"TXN1"`).
4. Encode operation payload.
5. Construct `CommitFooter` and compute dual CRC32 checksums.
6. Positioned append (`io.WriteAtFull`).
7. **Unconditional physical synchronization (`io.SyncFile` calling `fsync` / `FlushFileBuffers`).**
8. Atomic in-memory index publication (`index.ApplyUpdates`).
9. Transaction accounting (`committedEnd`, `nextTxID`, `txCount`).
10. Return `nil` to caller -> CLI emits `ACK <key>` / `ACK <count>`.

**Durability Rule:** An operation is **never acknowledged** until step 7 has returned successfully from non-volatile storage.

---

## 8. Atomic Batch Support
Multi-operation batches (up to 10,000 operations) commit as a single indivisible transaction frame:
- The entire batch shares a single `TxnHeader` and a single `CommitFooter`.
- Dual CRC32 covers the entire batch payload.
- If a crash occurs mid-batch, recovery encounters an incomplete frame or CRC mismatch, discards the uncommitted tail, and publishes **0%** of the batch.
- If the batch completes `SyncFile`, recovery validates the dual CRCs and applies **100%** of operations to the index in exact order.
- Partial batch visibility is mathematically impossible.

---

## 9. Crash Recovery & Self-Healing
On startup (`kv.Open`):
1. Validates immutable 32-byte `FileHeader`.
2. Scans log sequentially from offset 32, verifying `TxnHeader`, `PayloadLen`, `CommitFooter`, and dual CRCs.
3. If an unacknowledged torn write or truncated tail is encountered at EOF, recovery truncates the file back to the last valid `CommitFooter` and calls `SyncFile`.
4. Rebuilds the RAM index with live keys only.
5. Replays 1,000,000 keys in **0.29 seconds** on macOS ARM64 and **0.46 seconds** on Windows 11.

---

## 10. Crash-Safe Online Compaction
Reclaims disk space occupied by deleted keys (tombstones) and overwritten historical values:
- Holds database mutex; live key snapshot taken.
- Streams live keys to `<path>.compact.tmp` in dual-bounded batches (1,000 ops / 16 MiB).
- Calls `SyncFile` on replacement file.
- Atomically replaces authoritative file via OS rename (`os.Rename`).
- Swaps internal file descriptor and RAM index.
- If interrupted at any stage, original database remains untouched; leftover tmp files are cleaned up automatically on restart (Scenario C5).

---

## 11. Independent Torture Testing
The project includes a comprehensive, out-of-process crash-torture harness under [`torture/`](torture/):
- **Worker Subprocess:** Runs database in an isolated child process via `kip serve` over standard I/O pipes.
- **Persistent JSON Oracle:** Independently journals all sent requests and confirmed ACKs.
- **Asynchronous Crash Injection:** Kills the worker abruptly using OS termination signals (`TerminateProcess` on Windows, `SIGKILL` on POSIX) at arbitrary microsecond intervals.
- **Oracle Verification:** Reopens the database and verifies that 100% of acknowledged writes survive with exact bytes, and batches remain atomic.
- **1,000-Case Campaign:** Executed 1,000 randomized failure test cases across all official challenge categories with **1000/1000 PASS** ([`artifacts/1000_campaign/`](artifacts/1000_campaign/)).

---

## 12. Command-Line Reference

Build the executable:
```bash
go build -o kip ./cmd/kip
```

### The 8 Mandatory Challenge Commands:
```bash
# 1. Open or create database, run crash recovery, report telemetry
./kip open <db-path> [--max-bytes N]

# 2. Durably write key-value pair (prints ACK <key> on confirmed fsync)
./kip put <db-path> <key> <value>

# 3. Retrieve value (prints raw value, or NOTFOUND with exit code 2)
./kip get <db-path> <key>

# 4. Durably delete key (prints ACK <key>)
./kip del <db-path> <key>

# 5. Atomically commit batch from file (JSON array or line-based format)
./kip batch <db-path> <file>

# 6. Verify full physical file integrity, transaction boundaries, and dual CRCs
./kip check <db-path>

# 7. Print all live key-value pairs sorted lexicographically by key
./kip dump <db-path>

# 8. Interactive stdin/stdout IPC mode used by crash torture harnesses
./kip serve <db-path> [--max-bytes N]
```

### Administrative Commands:
```bash
# Display live engine telemetry (LiveKeys, FileSize, MaxCapacity, LastTxID)
./kip stats <db-path>

# Trigger crash-safe atomic space reclamation (compaction)
./kip compact <db-path>

# Display CLI reference manual
./kip help
```

For full schemas, exit codes, and examples, see [`docs/CLI_USAGE.md`](docs/CLI_USAGE.md).

---

## 13. Build Instructions
- Requires Go `1.23` or higher.
- Standard library only; no external Cgo or database dependencies.

```bash
# Build standalone binary
go build -o kip ./cmd/kip

# Cross-compile for Linux (from Windows or macOS)
GOOS=linux GOARCH=amd64 go build -o kip-linux ./cmd/kip

# Cross-compile for Windows (from Linux or macOS)
GOOS=windows GOARCH=amd64 go build -o kip.exe ./cmd/kip
```

---

## 14. Test Instructions

```bash
# 1. Run all unit and integration tests
go test -v -count=1 ./src/kv

# 2. Run CLI command and argument validation tests
go test -v -count=1 ./cmd/kip/...

# 3. Run independent crash torture acceptance tests
go test -v -count=1 ./torture/...

# 4. Run full project test suite
go test -count=1 ./...

# 5. Run race detector across packages (64-bit platforms)
go test -race -count=1 ./cmd/kip/...
go test -race -count=1 ./src/kv

# 6. Static analysis
go vet ./...
```

---

## 15. Demonstration Instructions
A complete, 5-minute judge demonstration script using an isolated disaster-relief inventory database is provided in [`docs/DEMO.md`](docs/DEMO.md).

On Linux / WSL2, run the automated verification script:
```bash
chmod +x scripts/wsl2_verify.sh
./scripts/wsl2_verify.sh
```

---

## 16. Platform Verification Matrix

| Platform | OS / Kernel | Go Version | Concurrency Locking | Storage & Compaction | Verified Status |
| :--- | :--- | :--- | :--- | :--- | :---: |
| **Native Windows 11** | AMD64 (NT 10.0.26300) | `go1.23.6` (64-bit) & `go1.27.3` (32-bit) | Win32 `LockFileEx` | NTFS append-only log + atomic rename | 🟢 **PASS** |
| **Linux (Ubuntu / WSL2)** | Linux 5.15 / 6.x | `go1.23.2` | POSIX `flock` | Ext4 append-only log + atomic rename | 🟢 **PASS** |
| **macOS (Darwin ARM64)** | macOS 15.3 (Apple M4) | `go1.23.2` | POSIX `flock` | APFS append-only log + atomic rename | 🟢 **PASS** |

---

## 17. Known Limitations & Failure Boundaries
1. **Physical Power-Loss Protection:** While our write path invokes physical `fsync` (`FlushFileBuffers`), survival under catastrophic DC power interruption requires drive hardware to honor flush barriers rather than caching writes in volatile DRAM.
2. **Deletion at 100% Capacity (Scenario E5):** In an append-only log, deletions append a tombstone record requiring $71 + \text{len(key)}$ bytes. When the file reaches `MaxFileBytes`, deletions return `ErrFull`. Compaction must be triggered to reclaim space.
3. **Recovery Scan Bounds:** Recovery scans from offset 32 to `committedEnd`. While 1,000,000 keys recover in under 0.3s, databases with 100,000,000+ keys will benefit from checkpoint snapshots.

---

## 18. Technical Documentation Index

| Document | Description |
| :--- | :--- |
| [`ARCHITECTURE.md`](ARCHITECTURE.md) | Complete system architecture, write path, recovery, compaction, 32-row proof table, and 3 guarantee proofs |
| [`BREAKING.md`](BREAKING.md) | Threat failure model, P1–P7 transaction failure analysis, crash-point proof table, and E5 capacity analysis |
| [`JUDGING_CHECKLIST.md`](JUDGING_CHECKLIST.md) | Master traceability matrix mapping all official challenge criteria, scenarios, and tests |
| [`PROMPTS.md`](PROMPTS.md) | Chronological, unedited archive of all development and engineering prompts |
| [`FINAL_INTEGRATION_REPORT.md`](FINAL_INTEGRATION_REPORT.md) | Master release integration report signed off by Tech Lead |
| [`docs/CLI_USAGE.md`](docs/CLI_USAGE.md) | Detailed CLI command manual, syntax, JSON schemas, exit codes, and IPC serve protocol |
| [`docs/DEMO.md`](docs/DEMO.md) | 5-minute interactive judge demonstration walkthrough |
| [`diagrams/`](diagrams/) | Editable Mermaid source files for architecture, write path, recovery, compaction, and torture |
| [`HANDOFF_MEMBER1.md`](HANDOFF_MEMBER1.md) | Member 1: Core storage foundation and 1M recovery benchmark handoff |
| [`HANDOFF_MEMBER2.md`](HANDOFF_MEMBER2.md) | Member 2: Recovery repair, dual-bounded compaction, and Win32 locking handoff |
| [`HANDOFF_MEMBER3.md`](HANDOFF_MEMBER3.md) | Member 3: Torture harness, concurrent multi-writer, and oracle hardening handoff |
| [`HANDOFF_MEMBER4.md`](HANDOFF_MEMBER4.md) | Member 4: CLI, argument validation, and safety verification handoff |
