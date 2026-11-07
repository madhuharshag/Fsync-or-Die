# Fsync-or-Die CLI (`kip`) Usage Guide & Reference Manual
**Project:** Fsync-or-Die  
**Team:** Innovexa  
**Binary:** `kip` / `kip.exe`  
**Package:** `cmd/kip`

---

## 1. Overview & Exit Codes

The `kip` CLI provides the official interface to the embedded crash-resilient key-value engine. It is designed for both human operators and automated testing tools (such as Member 3's crash torture injector).

### Standard Streams
- **Standard Output (`stdout`):** Reserved exclusively for confirmed acknowledgements (`ACK <key>`, `ACK <count>`), retrieved data values, scan dumps, and machine-readable IPC responses.
- **Standard Error (`stderr`):** All diagnostic messages, usage help on error, and failure reports are written to standard error with the prefix `error: <description>`.

### Exit Codes
| Exit Code | Constant | Meaning | Typical Scenarios |
| :--- | :--- | :--- | :--- |
| **`0`** | `ExitSuccess` | Operation completed and durably synchronized | Successful put, get of existing key, batch commit, clean open |
| **`1`** | `ExitError` | General operational or execution failure | Invalid arguments, capacity exceeded (`ErrFull`), corrupted file, locked database (`ErrBusy`), indeterminate sync (`ErrUncertain`) |
| **`2`** | `ExitNotFound` | Key not found | Running `kip get <db> <key>` on a key that does not exist or was deleted |

---

## 2. Command Reference

### 2.1 `open` — Open / Initialize & Replay Recovery
Opens or creates the database file, applies recovery if resuming after an interrupted session, repairs torn tails if present, and prints telemetry.

```bash
kip open <db-path> [--max-bytes N]
```

**Arguments:**
- `<db-path>`: Path to `.kip` database file.
- `--max-bytes N` (Optional): Maximum allowable file size in bytes (e.g. `50000000`). Setting to `0` means unlimited.

**Output:**
```text
OPENED: ./warehouse.kip (recreated=false, recovered_txns=42, live_keys=350, recovery_ms=0.485)
```

---

### 2.2 `put` — Durably Store a Key-Value Pair
Writes or updates a record. The CLI emits `ACK` **only after** physical `fsync` (`FlushFileBuffers`) has succeeded.

```bash
kip put <db-path> <key> <value>
```

**Limits:**
- Key length: 1 to 1,024 bytes.
- Value length: 0 to 1,048,576 bytes (1 MB). Empty values (`""`) are permitted and distinct from deletions.

**Example:**
```bash
kip put ./warehouse.kip emergency_rations 500kg
# Output:
ACK emergency_rations
```

---

### 2.3 `get` — Retrieve a Stored Record
Retrieves the value associated with `<key>`.

```bash
kip get <db-path> <key>
```

**Behavior:**
- **Key exists:** Value string is written directly to `stdout` followed by newline. Exit code is `0`.
- **Key missing or deleted:** Prints `NOTFOUND` to `stdout`. Exit code is `2`.
- **Engine error / lock failure:** Prints error to `stderr`. Exit code is `1`.

**Example:**
```bash
kip get ./warehouse.kip emergency_rations
# Output (Exit Code 0):
500kg

kip get ./warehouse.kip nonexistent_item
# Output (Exit Code 2):
NOTFOUND
```

---

### 2.4 `del` — Durably Delete a Key
Appends a deletion tombstone. Deleting a key that does not exist succeeds idempotently.

```bash
kip del <db-path> <key>
```

**Example:**
```bash
kip del ./warehouse.kip emergency_rations
# Output:
ACK emergency_rations
```

---

### 2.5 `batch` — Atomically Apply Multiple Operations
Executes up to 10,000 operations as an atomic unit. Either all operations commit and survive, or none do.

```bash
kip batch <db-path> <file>
```

The batch file can be in either **JSON Array format** or **Line-Delimited text format**.

#### A. JSON Array Format
Supports both numeric kinds (`1=put`, `2=del`) and human-readable action tags (`"op": "put"` / `"op": "del"`):

```json
[
  {"op": "put", "key": "relief_rice", "value": "1000kg"},
  {"op": "put", "key": "relief_water", "value": "2000L"},
  {"op": "del", "key": "expired_medicine"}
]
```

Or Member 3 wire schema:
```json
[
  {"kind": 1, "key": "relief_rice", "val": "1000kg"},
  {"kind": 1, "key": "relief_water", "val": "2000L"},
  {"kind": 2, "key": "expired_medicine"}
]
```

#### B. Line-Delimited Format
Per Section 6 of the hackathon specification, text files with one operation per line are supported:
```text
# lines starting with '#' are ignored
put relief_tents 150
put relief_blankets 600
del damaged_stretchers
```

**Output:**
```bash
kip batch ./warehouse.kip ./ops.json
# Output:
ACK 3
```

---

### 2.6 `check` — File Integrity Verification
Performs a full sequential scan validating header magic, transaction headers, payload byte boundaries, commit footers, matching TxIDs, and dual CRC32-IEEE checksums.

```bash
kip check <db-path>
```

**Output on Valid Database:**
```text
OK: Database integrity verified (45 transactions)
```

**Output on Corrupted Database:**
```text
error: integrity check failed: kv: corrupt or torn record encountered: check: txn CRC mismatch at 1042
```

---

### 2.7 `dump` / `scan` — Sorted Inventory Dump
Traverses all active records in the in-memory RAM index and displays them sorted lexicographically by key.

```bash
kip dump <db-path>
```

**Output:**
```text
relief_blankets = 600
relief_rice = 1000kg
relief_tents = 150
```

If the database is empty:
```text
(empty database)
```

---

### 2.8 `stats` — Operational Telemetry
Reports snapshot metrics from the database handle:

```bash
kip stats <db-path>
```

**Output:**
```text
LiveKeys: 3
FileSizeBytes: 4280
MaxFileBytes: 50000000
LastTxID: 15
TxCount: 15
CompactionCount: 1
```

---

### 2.9 `compact` — Crash-Safe Storage Compaction
Reclaims disk space consumed by superseded updates and deletion tombstones by streaming live keys into a temporary replacement file and atomically swapping pointers.

```bash
kip compact <db-path>
```

**Output:**
```text
OK: Compaction completed (new size: 1048 bytes)
```

---

### 2.10 `serve` — Subprocess IPC Server for Torture Testing
Long-running mode communicating over `stdin` and `stdout`. Used directly by Member 3's independent crash torture harness (`torture/runner.go`) and judge stress tests.

```bash
kip serve <db-path> [--max-bytes N]
# or:
kip --worker
```

**Wire Protocol Specification:**

| Inbound Command (stdin) | Arguments | Response (stdout) |
| :--- | :--- | :--- |
| `OPEN <path> [max_bytes]` | Path and optional capacity | `READY\n` or `ERR 0 <msg>\n` |
| `PUT <opID> <b64key> <b64val>` | Request ID, base64 key, base64 val | `ACK <opID>\n` or `ERR <opID> <msg>\n` |
| `GET <opID> <b64key>` | Request ID, base64 key | `VAL <opID> <b64val>\n`, `NOT_FOUND <opID>\n`, or `ERR <opID> <msg>\n` |
| `DEL <opID> <b64key>` | Request ID, base64 key | `ACK <opID>\n` or `ERR <opID> <msg>\n` |
| `BATCH <opID> <b64json>` | Request ID, base64 encoded JSON array | `ACK <opID>\n` or `ERR <opID> <msg>\n` |
| `COMPACT [opID]` | Optional request ID | `ACK <opID>\n` or `ERR <opID> <msg>\n` |
| `CLOSE [opID]` | Optional request ID | `ACK <opID>\n` (releases locks & terminates) |
| `PING` | None | `PONG\n` |

All responses are immediately flushed via `os.Stdout.Sync()`.

---

## 3. Error Handling Principles

1. **Durability Guarantee:** The CLI guarantees that `ACK` is **never emitted** unless `db.Put()` or `db.Batch()` returns `nil`.
2. **Indeterminate State (`ErrUncertain`):** If a disk synchronization barrier fails after payload bytes are queued, the engine returns `kv.ErrUncertain`. The CLI outputs a descriptive warning to `stderr` explaining that the transaction's persistence state cannot be guaranteed and exits with code `1`. It never claims the write succeeded and never assumes it failed.
3. **Capacity Full (`ErrFull`):** When the configured `--max-bytes` boundary is reached, writes are safely rejected without modifying existing state.
4. **Lock Contention (`ErrBusy`):** If another process has opened the database, `acquireProcessLock` receives `ERROR_LOCK_VIOLATION` (or `EWOULDBLOCK`), causing `kip` to report `error: open failed: kv: database file is locked by another process` and exit cleanly with code `1`.

