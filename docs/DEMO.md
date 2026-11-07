# Team Innovexa — 5-Minute Judge Demonstration Script
**Project:** Fsync-or-Die (`kip`)  
**Scenario:** Emergency Disaster-Relief Warehouse Inventory Tracking  
**Target:** Live 15-Minute Judge Evaluation (Section 11)

---

## 1. Setup & Build Verification (Minute 0 – 1)

Verify clean build without external dependencies or internet access:

```bash
# Build the single CLI binary
go build -o kip ./cmd/kip
```
*(On Windows PowerShell, use `go run ./cmd/kip` or `./kip.exe`)*

**Expected output:** Zero compilation errors, single clean binary created.

---

## 2. Fresh Database Creation & Telemetry (Minute 1 – 2)

Initialize an authoritative relief warehouse inventory file with a 50 MB capacity ceiling:

```powershell
go run ./cmd/kip open ./warehouse.kip --max-bytes 50000000
```

**Expected output:**
```text
OPENED: ./warehouse.kip (recreated=true, recovered_txns=0, live_keys=0, recovery_ms=3.125)
```
*Notice: Brand-new database initialized with 32-byte immutable header and CRC32-IEEE checksum.*

---

## 3. Durable Puts & Reads (Minute 2 – 3)

Deliver emergency supply shipments. Each `ACK` guarantees a physical disk sync:

```powershell
go run ./cmd/kip put ./warehouse.kip rice "500 bags"
go run ./cmd/kip put ./warehouse.kip water "1200 liters"
go run ./cmd/kip put ./warehouse.kip medical_kits "80 sets"
```

**Expected output:**
```text
ACK rice
ACK water
ACK medical_kits
```

Verify reading an existing record:
```powershell
go run ./cmd/kip get ./warehouse.kip rice
```
**Expected output (Exit code 0):**
```text
500 bags
```

Verify missing key lookup:
```powershell
go run ./cmd/kip get ./warehouse.kip blankets
```
**Expected output (Exit code 2):**
```text
NOTFOUND
```

---

## 4. Atomic Batch & Idempotent Deletion (Minute 3 – 4)

Execute an atomic relief shipment batch containing both additions and inventory removals:

```powershell
# Create batch operation file
@'
[
  {"op": "put", "key": "blankets", "value": "350 units"},
  {"op": "put", "key": "tents", "value": "50 large"},
  {"op": "del", "key": "medical_kits"}
]
'@ | Set-Content -Path "shipment.json"

go run ./cmd/kip batch ./warehouse.kip shipment.json
```

**Expected output:**
```text
ACK 3
```

Verify that `medical_kits` was removed and new items were committed:
```powershell
go run ./cmd/kip get ./warehouse.kip medical_kits
# Outputs: NOTFOUND (exit code 2)

go run ./cmd/kip dump ./warehouse.kip
```

**Expected output (sorted lexicographically):**
```text
blankets = 350 units
rice = 500 bags
tents = 50 large
water = 1200 liters
```

---

## 5. Storage Compaction & Full Integrity Verification (Minute 4 – 5)

Display file size and operational counters:
```powershell
go run ./cmd/kip stats ./warehouse.kip
```

Run crash-safe compaction to reclaim dead records and deletion tombstones:
```powershell
go run ./cmd/kip compact ./warehouse.kip
```
**Expected output:**
```text
OK: Compaction completed (new size: 284 bytes)
```

Run the exhaustive integrity checker scanning all transaction boundaries and dual CRCs:
```powershell
go run ./cmd/kip check ./warehouse.kip
```
**Expected output:**
```text
OK: Database integrity verified (1 transactions)
```

---

## 6. Crash Resilience & Torture Harness Proof (Judge Demonstration)

Prove that abrupt process termination (`TerminateProcess` / `SIGKILL`) never causes data loss:

```powershell
# Run 10 rapid randomized crash cycles against the database
go run ./torture -crashes 10 -ops 100 -seed 12345
```

**Expected output:**
```text
======================================================================
  FSYNC-OR-DIE INDEPENDENT RELIABILITY & TORTURE HARNESS
======================================================================
Configuration:
  Mode:       random
  Seed:       12345
  Total Ops:  100
  Crashes:    10
  Writers:    2
...
PASS: 10/10 crash cycles verified cleanly. 0 invariant violations.
```

---

## 7. Clean-Up

To reset demo artifacts:
```powershell
Remove-Item warehouse.kip*, shipment.json -Force -ErrorAction SilentlyContinue
```

