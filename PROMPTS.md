# TEAM INNOVEXA — FSYNC-OR-DIE
# PROMPTS.MD — CHRONOLOGICAL UNEDITED PROMPT ARCHIVE

**Project:** Fsync-or-Die (`kip`)  
**Tagline:** *“Don’t Trust the Process. Trust the Proof.”*  
**Repository:** [github.com/madhuharshag/Team-Innovexa](https://github.com/madhuharshag/Team-Innovexa)  
**Authoritative Source:** Machine-logged session transcripts (`transcript_full.jsonl`)  
**Status:** 100% Verbatim & Chronologically Preserved  
**Date Range:** 2026-11-06T07:08:03Z to 2026-11-07T03:17:24Z  

---

> [!NOTE]
> **Provenance & Authenticity Notice:**  
> In compliance with the official challenge submission guidelines, every prompt in this document is preserved verbatim from the session logs recorded by Google Antigravity. No prompt text has been fabricated, backfilled, or retrospectively edited. Prompts are organized in exact chronological sequence.

---

## Table of Contents
1. [Prompt 1: Initial Repository & Branch Switch (2026-11-06 07:08 UTC)](#prompt-1-initial-repository--branch-switch)
2. [Prompt 2: Member 1 Phase 0/1 Architecture Confirmation (2026-11-06 08:44 UTC)](#prompt-2-member-1-phase-01-architecture-confirmation)
3. [Prompt 3: Go Toolchain Setup & Environment Discovery (2026-11-06 09:02 UTC)](#prompt-3-go-toolchain-setup--environment-discovery)
4. [Prompt 4: Member 1 Complete Core Implementation Mission (2026-11-06 09:09 UTC)](#prompt-4-member-1-complete-core-implementation-mission)
5. [Prompt 5: Toolchain Verification & Physical Execution Gate (2026-11-06 09:30 UTC)](#prompt-5-toolchain-verification--physical-execution-gate)
6. [Prompt 6: Member 1 Local Commit Authorization (2026-11-06 09:51 UTC)](#prompt-6-member-1-local-commit-authorization)
7. [Prompt 7: Push feat/storage-core to Origin (2026-11-06 09:57 UTC)](#prompt-7-push-featstorage-core-to-origin)
8. [Prompt 8: Member 1 Tech Lead Master Delivery & WSL2 Benchmark (2026-11-06 10:09 UTC)](#prompt-8-member-1-tech-lead-master-delivery--wsl2-benchmark)
9. [Prompt 9: 1-Million Key Cold Recovery Benchmark Authorization (2026-11-06 10:25 UTC)](#prompt-9-1-million-key-cold-recovery-benchmark-authorization)
10. [Prompt 10: Member 1 Final Completion & Integrity Checker Mission (2026-11-06 10:32 UTC)](#prompt-10-member-1-final-completion--integrity-checker-mission)
11. [Prompt 11: Member 2 Completion & Torture Delivery Report (2026-11-06 13:08 UTC)](#prompt-11-member-2-completion--torture-delivery-report)
12. [Prompt 12: Tech Lead Authorized Final Integration Reconciliation (2026-11-07 02:28 UTC)](#prompt-12-tech-lead-authorized-final-integration-reconciliation)
13. [Prompt 13: Final Judging Documentation Audit & Project Organization (2026-11-07 03:17 UTC)](#prompt-13-final-judging-documentation-audit--project-organization)

---

## Prompt 1: Initial Repository & Branch Switch
**Timestamp:** `2026-11-06T07:08:03Z`  
**Transcript Step Index:** `0`  

```text
I am a member of Team Innovexa, working on the project Fsync-or-Die.

My assigned GitHub branch is `feat/storage-core`.

Please switch my local project to this branch.

Follow these steps:
1. Locate the root folder of my cloned GitHub repository.
2. Check the current Git branch and working-tree status.
3. Fetch the latest remote branches using `git fetch origin`.
4. If `feat/storage-core` already exists locally, switch to it.
5. Otherwise, if it exists on GitHub, create a local tracking branch and switch to it.
6. Verify the active branch using `git branch --show-current`.

IMPORTANT:
- Do not create another repository or branch.
- Do not modify, delete or overwrite any files.
- Do not reset, clean or discard changes.
- Do not merge anything into main.
- If there are uncommitted changes, stop and explain before switching.
- If the branch does not exist remotely, stop and tell me.

Do not start coding yet. Confirm that I am on `feat/storage-core` and ready to work.
```

---

## Prompt 2: Member 1 Phase 0/1 Architecture Confirmation
**Timestamp:** `2026-11-06T08:44:27Z`  
**Transcript Step Index:** `33`  

```text
FSYNC-OR-DIE — MEMBER 1
ROLE: TECH LEAD + CORE STORAGE ENGINEER
BRANCH: feat/storage-core

PHASE 0: UNDERSTAND AND INSPECT
PHASE 1: CONFIRM THE STORAGE ARCHITECTURE

We will work interactively, starting with repository inspection and architecture confirmation. We will not create or change implementation files yet.

Your first deliverable is a contract: one agreed storage format, one public API, and clear ownership boundaries that let the other three members develop without inventing incompatible implementations.

I have reread the complete 19-page challenge. I also checked for an available GitHub connector; none is available in this session. Therefore, I have not inspected your repository, local branch, files, or Go installation. We must obtain that information before generating changes.

The architecture below carries forward our previous planning. It is a proposal pending confirmation against your actual repository—not a claim that your team has implemented or formally approved it.


======================================================================
PHASE 0 — UNDERSTAND BEFORE CODING
======================================================================

A. YOUR ROLE IN SIMPLE WORDS

You are building the database’s filing system.

Imagine a notebook:

- The file is the notebook.
- An encoded transaction is a new entry.
- The checksum helps identify an incomplete or damaged entry.
- The in-memory index tells you where each key’s latest value is stored.
- Synchronization is the step that must complete before you promise the entry is safe.
- Recovery, owned by Member 2, reconstructs the index after the process dies.

Your job is to ensure that every component agrees on what the bytes mean and who is allowed to change them.

YOUR OWNERSHIP

1. File-format specification and codec
   Coordinate with: Member 2.

2. Database state and in-memory index
   Coordinate with: Member 2.

3. Low-level file I/O helpers
   Coordinate with: Member 2.

4. Put/Get/Delete public entry points
   Coordinate with: Member 2’s common transaction path.

5. Input validation and size accounting
   Coordinate with: Member 2.

6. File ownership and locking foundation
   Coordinate with: Member 2.

7. Public API and error categories
   Coordinate with: Members 2, 3, and 4.

8. Storage correctness tests
   Coordinate with: Members 2 and 3.

9. Final architecture/code reviews
   Coordinate with: Entire team.

WHAT YOU MUST NOT IMPLEMENT INDEPENDENTLY

Do not build one append protocol for Put while Member 2 builds a different protocol for Batch.

The intended relationship is:

Put    ──┐
         ├── Common transaction writer
Delete ──┤       ↓
         │   Synchronization
Batch  ──┘       ↓
             Index publication

Likewise, do not create a second recovery scanner just to make your Open work. Open needs one coordinated initialization-and-recovery implementation.


----------------------------------------------------------------------
B. OFFICIAL REQUIREMENTS RELEVANT TO YOUR ROLE
----------------------------------------------------------------------

The page numbers below refer to the supplied “Kill It and Prove It” PDF.

1. SINGLE-PROCESS EMBEDDED KEY-VALUE DATABASE
   PDF page: 2.

   Meaning:
   This is a library/storage engine, not a web service.

   Your implementation:
   Public Go API and database handle.

   Verification:
   API and CLI inspection.

   Failure:
   Required behavior depends on a separate application or server.

2. PUT STORES OR OVERWRITES; SUCCESS IS DURABLE
   PDF page: 2.

   Meaning:
   Returning success is a promise.

   Your implementation:
   Put delegates to the durable transaction path.

   Verification:
   Kill immediately after success.

   Failure:
   The acknowledged effect disappears without a later valid operation.

3. GET RETURNS CURRENT VALUE OR NOT FOUND
   PDF page: 2.

   Meaning:
   Return complete values only.

   Your implementation:
   Index lookup and exact value read.

   Verification:
   Missing-key, overwrite, and binary-value tests.

   Failure:
   Wrong value, torn value, or incorrect not-found result.

4. DELETE IS DURABLE; DELETING A MISSING KEY IS NOT AN ERROR
   PDF page: 2.

   Meaning:
   Absence is a valid committed result.

   Your implementation:
   Delete operation/tombstone representation.

   Verification:
   Reopen and kill tests.

   Failure:
   Deleted data reappears incorrectly.

5. ATOMIC BATCHES OF PUTS AND DELETES
   PDF page: 2.

   Meaning:
   All batch operations act together.

   Your implementation:
   Compatible format and API for Member 2.

   Verification:
   Partial-write and kill tests.

   Failure:
   Normal reads or recovery expose a partial batch.

6. OPEN/CREATE/RECOVER
   PDF page: 2.

   Meaning:
   Existing files must not be accidentally erased.

   Your implementation:
   Coordinate opening and recovery.

   Verification:
   Reopen killed-run files.

   Failure:
   Files produced during valid database execution cannot reopen after a kill.

7. OPEN UNDER TWO SECONDS WITH ONE MILLION KEYS
   PDF page: 2.

   Meaning:
   Startup cost is a requirement.

   Your implementation:
   Efficient format, index representation, and benchmark support.

   Verification:
   Timed recovery on the judging laptop.

   Failure:
   Recovery exceeds the specified limit.

8. CLOSE FLUSHES AND CLOSES CLEANLY
   PDF page: 2.

   Meaning:
   Cleanup must work, but durability cannot wait for close.

   Your implementation:
   Serialized close and checked errors.

   Verification:
   Clean close and kill-during-close tests.

   Failure:
   Prior successful operations depend on close finishing.

9. KEYS ARE BYTE STRINGS, 1–1,024 BYTES
   PDF page: 2.

   Meaning:
   Keys are not restricted to printable text.

   Your implementation:
   Validation and byte-safe encoding.

   Verification:
   Boundary and binary-key tests.

   Failure:
   Empty or oversized keys accepted, or valid binary keys mishandled.

10. VALUES ARE BYTE STRINGS, 0–1 MB
    PDF page: 2.

    Meaning:
    Empty values are valid and are not deletion.

    Your implementation:
    Separate operation kind from value length.

    Verification:
    Empty, large, and binary-value tests.

    Failure:
    Empty put treated as delete, or supported values truncated.

11. AT LEAST ONE MILLION KEYS
    PDF page: 2.

    Meaning:
    The RAM index and file offsets must scale.

    Your implementation:
    Store offsets, not all values, in the index.

    Verification:
    Large-dataset tests.

    Failure:
    Index/file representation cannot handle the dataset.

12. UP TO 10,000 OPERATIONS PER BATCH
    PDF page: 2.

    Meaning:
    Avoid small artificial operation-count limits.

    Your implementation:
    Appropriate count fields, validation, and streaming-compatible codec.

    Verification:
    Large-batch tests.

    Failure:
    Valid batches rejected or partially processed.

13. AT LEAST TWO CONCURRENT WRITER THREADS/PROCESSES
    PDF page: 2.

    Meaning:
    Calls may overlap.

    Your implementation:
    Consistent mutex and shared-state rules.

    Verification:
    Concurrent library calls and torture workloads.

    Failure:
    Races, interleaved records, or corrupted state.

14. READS CANNOT OBSERVE HALF-APPLIED BATCHES
    PDF page: 2.

    Meaning:
    Readers must respect the publication lock.

    Your implementation:
    Lock reads as well as writes.

    Verification:
    Controlled reader/writer test.

    Failure:
    A get enters during partial batch publication.

15. EXACTLY ONE AUTHORITATIVE DATA FILE
    PDF page: 3.

    Meaning:
    Do not create an authoritative data file plus separate authoritative WAL.

    Your implementation:
    One transaction log is the database.

    Verification:
    Filesystem and architecture inspection.

    Failure:
    Authoritative state depends on multiple data files.

16. LOCK FILE AND COMPACTION TEMPORARY ARE ALLOWED
    PDF page: 3.

    Meaning:
    Auxiliary files are permitted for those purposes.

    Your implementation:
    Stable ownership lock; coordinate temporary files.

    Verification:
    Confirm authoritative state remains in one file.

17. FILE MUST SHRINK THROUGH CRASH-SAFE COMPACTION
    PDF page: 3.

    Meaning:
    The log cannot grow indefinitely.

    Your implementation:
    Format and descriptor/index design support replacement.

    Verification:
    Size reduction and compaction-kill tests.

    Failure:
    File does not shrink, or compaction loses/corrupts data.

18. CONFIGURABLE MAXIMUM FILE SIZE
    PDF page: 3.

    Meaning:
    Simulate a full disk.

    Your implementation:
    Exact encoded-size accounting and persisted-cap policy.

    Verification:
    Cap-boundary tests.

    Failure:
    File exceeds the configured limit or enforcement is inconsistent.

19. FULL-CAP FAILURE MUST PRESERVE READABILITY AND RECOVERY
    PDF page: 3.

    Meaning:
    Rejected writes must not damage existing data.

    Your implementation:
    Preflight checks and checked I/O errors.

    Verification:
    E1–E5 tests.

    Failure:
    Rejected writes partially modify state or make the database unopenable.

These are mandatory challenge behaviors, not optional improvements.


GOVERNING CRASH RULES

The PDF requires the following after recovery:

- Successful operations are preserved in a valid recovered history.
- In-flight operations are fully included or fully absent.
- Batches are never partially included.
- Values are never invented, torn, mixed, or associated with the wrong key.
- Recovery must tolerate being killed itself.

PDF pages: 4–5.

“Preserved in history” does not mean an old value can never be overwritten.

Example:

put rice 500 → success
put rice 450 → success

Recovering rice = 450 is correct.


FAILURE-MODEL BOUNDARY

The required crash is unexpected process termination, including during:

- Write.
- Synchronization.
- Rename.
- Recovery.

The OS and disk remain running. OS crashes and laptop power loss are not required.

PDF page: 5.

We must not advertise power-loss safety based only on process-kill tests.


RESTRICTIONS YOU MUST RESPECT

- Standard library is permitted.
- Storage engines, persistence libraries, WAL/B-tree/LSM implementations, and database wrappers are banned.
- Existing database source must not be copied.
- Event work must have meaningful Git history.
- The implementation must build and run offline.
- AI prompts must be logged.
- No AI or internet during the demo.

PDF pages: 3–4, 11–12, 15, and 17.


----------------------------------------------------------------------
C. MANDATORY VERSUS OPTIONAL DESIGN CHOICES
----------------------------------------------------------------------

The PDF specifies behavior, not a required implementation format.

MANDATORY

- Durable successful writes.
- Atomic batch behavior.
- Concurrent callers.
- Crash-safe compaction.
- Configurable file-size cap.

OUR RECOMMENDED DESIGN

- Append-only log.
- CRC32-IEEE.
- One global mutex.
- RAM key-to-offset index.
- Transaction IDs.

OPTIONAL OR DEFERRED

- Persisted checkpoint/index:
  Add only if measurement shows it is needed.

- High-throughput or lock-free writes:
  Not a scoring priority.

OUTSIDE THE REQUESTED PRODUCT

- Frontend.
- Dashboard.
- Network API.

The PDF explicitly permits a simple append-only log and says a global lock is acceptable.

PDF pages: 2 and 18.


----------------------------------------------------------------------
D. REPOSITORY INSPECTION — DO THIS BEFORE MODIFYING FILES
----------------------------------------------------------------------

Run these commands inside your existing local clone using Antigravity’s terminal.

These commands inspect state. They do not switch branches, initialize a module, edit files, commit, or merge.


1. REPOSITORY AND BRANCH

git rev-parse --show-toplevel
git branch --show-current
git status --short
git branch --list
git log -5 --oneline

Expected branch:

feat/storage-core

If another branch appears, stop. Do not switch automatically until we understand whether you have uncommitted work.


2. EXISTING TRACKED FILES

git ls-files

Also include untracked filenames:

git ls-files --others --exclude-standard

Do not paste secrets, tokens, .env contents, or private credentials.


3. EXISTING SHARED INTERFACE AND DESIGN FILES

In Antigravity’s file explorer, look for:

go.mod
src/kv/contract.go
src/kv/db.go
ARCHITECTURE.md
README.md

Also check for an existing alternative layout such as:

internal/engine/
internal/storage/
internal/record/

Paste the contents of:

- go.mod, if present.
- Existing engine/API type definitions.
- Existing file-format definitions.
- Any already-approved architecture section.

If those files do not exist, say so.

Do not create them yet.


4. ENVIRONMENT

go version
go env GOOS GOARCH GOMOD GOTOOLCHAIN

Also state:

- Your operating system.
- The intended judging laptop’s operating system.
- Whether the repository/database test directory is on local storage or a network/cloud-synced folder.

If Go is not installed, share the error. Installation belongs to Phase 2.


WHY THIS INSPECTION IS NECESSARY

Our previous report suggested src/kv/.

Your current prompt offers an internal/... structure as another possibility.

Neither should override existing working team code without inspection and agreement.


======================================================================
PHASE 1 — ARCHITECTURE AND PREPARATION
======================================================================

E. RECOMMENDED CORE ARCHITECTURE

ALTERNATIVE 1: APPEND-ONLY LOG + RAM INDEX

How it works:
Append new transactions. The index points to latest values.

Main complication:
Recovery scan and compaction.

Decision:
Recommended.


ALTERNATIVE 2: MUTABLE PAGE-BASED STORAGE

How it works:
Update fixed-size pages containing records or tree nodes.

Main complication:
Torn page updates, allocation, journaling, and root publication.

Decision:
Too much new machinery.


ALTERNATIVE 3: COPY-ON-WRITE PAGES

How it works:
Write replacement pages and publish a new root.

Main complication:
Root safety and free-space reclamation.

Decision:
Avoid for this event.


ALTERNATIVE 4: WHOLE-FILE REPLACEMENT FOR EVERY WRITE

How it works:
Rebuild the complete database on every mutation.

Main complication:
Poor scaling.

Decision:
Not the ordinary write path.


ALTERNATIVE 5: APPEND LOG PLUS PERSISTED CHECKPOINT

How it works:
Load an index snapshot, then replay the tail.

Main complication:
Checkpoint publication and validation.

Decision:
Consider only after benchmarking.


WHY THE APPEND-ONLY DESIGN FITS

An unfinished transaction sits after already committed data:

[valid transaction]
[valid transaction]
[unfinished transaction]

The unfinished append does not require rewriting earlier records.

The index makes reads direct:

"inventory/rice" → offset 1842, length 6

Compaction later rewrites only the latest live values into a new file.


DISADVANTAGES WE MUST ACKNOWLEDGE

- Recovery work depends on log bytes, not only live-key count.
- Obsolete versions consume space until compaction.
- The index consumes RAM for every live key.
- Compaction temporarily needs extra storage.
- Holding one lock during compaction blocks operations.

These are acceptable starting trade-offs.

The two-second startup requirement must still be measured, not assumed.


----------------------------------------------------------------------
F. ARCHITECTURE DIAGRAMS
----------------------------------------------------------------------

OVERALL OWNERSHIP — MERMAID

flowchart TD
    CLI[Member 4: CLI and serve workers]
    API[Shared public API]
    DB[Member 1: DB state and global mutex]
    FORMAT[Member 1: format and checked I/O]
    INDEX[Member 1: RAM key-to-offset index]
    TX[Member 2: transaction commit]
    REC[Member 2: recovery]
    COMP[Member 2: compaction]
    FILE[(One authoritative file)]
    HARNESS[Member 3: independent torture process]

    CLI --> API
    API --> DB
    DB --> TX
    DB --> INDEX
    TX --> FORMAT
    FORMAT --> FILE
    REC --> FORMAT
    REC --> INDEX
    COMP --> FORMAT
    COMP --> INDEX
    HARNESS -->|stdin/stdout and force-kill| CLI


WRITE AND DELETE FLOW — MERMAID

flowchart TD
    PUT[Put key,value] --> SINGLE[One-operation transaction]
    DEL[Delete key] --> SINGLE
    BATCH[Batch operations] --> MUTEX[Acquire DB mutex]
    SINGLE --> MUTEX
    MUTEX --> VALIDATE[Validate entire request]
    VALIDATE --> CAP[Check exact encoded size and capacity]
    CAP --> WRITE[Write transaction header and operations]
    WRITE --> FOOTER[Write commit footer]
    FOOTER --> SYNC[Sync file and check result]
    SYNC --> INDEX[Publish all index changes]
    INDEX --> RETURN[Release mutex and return success]
    RETURN --> ACK[CLI may emit ACK]


READ FLOW — MERMAID

flowchart TD
    GET[Get key] --> LOCK[Acquire same DB mutex]
    LOCK --> LOOKUP[Lookup key in index]
    LOOKUP -->|missing| NF[Return ErrNotFound]
    LOOKUP -->|present| READ[Read exact length at stored value offset]
    READ --> COPY[Return owned value bytes]
    NF --> UNLOCK[Release mutex]
    COPY --> UNLOCK

Reads initially hold the mutex through the file read.

This prevents compaction from changing the file descriptor while a read uses offsets associated with the previous file.


----------------------------------------------------------------------
G. PROPOSED STORAGE FORMAT — CARRY FORWARD, DO NOT REINVENT
----------------------------------------------------------------------

The following is the existing planning proposal.

Before implementation, we must compare it with Member 2’s current work.

All integers use little-endian encoding.


FILE HEADER: 32 BYTES

Offset  Bytes  Field
0       8      Magic: KIPDB001
8       4      Format version: 1
12      4      Header size: 32
16      8      Maximum file size; zero means unlimited
24      4      Flags, initially zero
28      4      CRC32-IEEE of bytes 0–27


TRANSACTION HEADER: 32 BYTES

Offset  Bytes  Field
0       4      Magic: TXN1
4       2      Transaction version
6       2      Flags
8       8      Transaction ID
16      4      Operation count
20      4      Reserved
24      8      Payload byte length


OPERATION

[1][key length:2][value length:4][key bytes][value bytes]

Rules:

- Kind 1: put.
- Kind 2: delete.
- Delete value length: zero.
- Empty put value: permitted.
- Batch operations apply in list order.

A delete is a tombstone: a log operation saying “this key is absent now.”

It is not an empty value.


COMMIT FOOTER: 32 BYTES

Offset  Bytes  Field
0       8      Magic: COMMIT01
8       8      Matching transaction ID
16      8      Total transaction length
24      4      CRC32-IEEE of transaction header and payload
28      4      CRC32-IEEE of footer bytes 0–27


EXAMPLE

A put of key rice and value 500:

Transaction header:
    32 bytes

Operation:
    kind          1 byte   = put
    key length    2 bytes  = 4
    value length  4 bytes  = 3
    key           4 bytes  = rice
    value         3 bytes  = 500

Commit footer:
    32 bytes

Total:

32 + 1 + 2 + 4 + 4 + 3 + 32 = 78 bytes


VALIDATION RULES

A transaction is eligible for replay only if:

1. Header magic, version, and flags are valid.
2. Length arithmetic does not overflow.
3. Operation count is permitted.
4. Every operation is structurally valid.
5. Payload ends exactly where specified.
6. Footer is complete.
7. Transaction IDs match.
8. Total length matches.
9. Checksums match.

A footer’s presence alone is not enough.


EOF VERSUS INCOMPLETE RECORD

- EOF exactly at a validated committed boundary:
  Normal end.

- EOF partway through a header, payload, or footer:
  Incomplete tail.

- Unsupported version or invalid immutable header:
  Explicit error.

- Arbitrary middle corruption:
  Not automatically permission to delete all later history.

The brief excludes random middle-of-file bit rot from required testing.

PDF page: 8.


FORWARD COMPATIBILITY

For the first version:

- Reject unsupported versions.
- Reject unknown flag bits and operation types.
- Do not guess how to skip unfamiliar transactions.
- Change version deliberately when incompatible encoding changes.

That is simpler and safer than pretending to support unknown future formats.


----------------------------------------------------------------------
H. PROPOSED FOLDER STRUCTURE AND OWNERSHIP
----------------------------------------------------------------------

Retain this layout only if repository inspection shows it is compatible with existing work:

src/
└── kv/
    ├── contract.go          Member 1; shared review
    ├── db.go                Member 1
    ├── format.go            Member 1; Member 2 reviews
    ├── io.go                Member 1
    ├── capacity.go          Member 1; Member 2 coordinates fallback
    ├── lock_linux.go        Member 1
    ├── transaction.go       Member 2
    ├── recovery.go          Member 2
    └── compact.go           Member 2

src/cli/                    Member 4
src/cmd/kvdb/               Member 4
torture/                    Member 3
tests/                      Coordinated integration tests
diagrams/                   Member 4 with technical review


WHY NOT CREATE SIX INTERNAL PACKAGES IMMEDIATELY?

DB, index publication, recovery, transaction commit, and compaction share tightly coupled invariants.

Splitting them prematurely can create:

- Circular imports.
- Unnecessary exported fields.
- Duplicate state ownership.
- Complex plumbing before correctness works.

A single small kv package with separate owned files is a reasonable starting point.

Do not move existing code merely to match this diagram.


----------------------------------------------------------------------
I. SHARED API TO VERIFY AGAINST THE REPOSITORY
----------------------------------------------------------------------

Our previous report proposed these signatures:

func Open(path string, opts Options) (Engine, OpenInfo, error)

type Engine interface {
    Put(key, value []byte) error
    Get(key []byte) ([]byte, error)
    Delete(key []byte) error
    Batch(ops []Op) error

    Compact() error
    Check() error
    Scan(fn func(key, value []byte) error) error
    Stats() Stats

    Close() error
}

And these operation kinds:

type Kind uint8

const (
    PutOp    Kind = 1
    DeleteOp Kind = 2
)

type Op struct {
    Kind  Kind
    Key   []byte
    Value []byte
}

These are previously proposed interfaces, not verified repository definitions.

We will not generate replacements until we see the actual contract.


SEMANTICS TO APPROVE

Successful mutation:
    nil error means durable under the challenge model.

Missing get:
    ErrNotFound.

Missing delete:
    Success.

Duplicate keys in batch:
    Apply in list order.

Empty value:
    Present value of zero bytes.

Returned get bytes:
    Owned by caller.

Inputs:
    Caller must not mutate until operation returns.

Thread safety:
    Public methods serialize through DB mutex.

Failed configured-cap preflight:
    Definitely rejected; no transaction appended.

Uncertain I/O outcome:
    Explicit error; no false success; handle may require reopening.

Scan callback:
    Must not reenter DB.

Format ownership:
    Member 1 coordinates; Member 2 reviews transaction changes.


PROPOSED INTERNAL STATE

Conceptually:

- Mutex.
- Current file descriptor.
- Stable ownership-lock descriptor.
- Database pathname.
- Key-to-offset index.
- Committed end offset.
- Next transaction ID.
- Configured cap.
- Closed state.
- Poisoned/error state.

We will decide exact Go fields after seeing existing files.


----------------------------------------------------------------------
J. INTEGRATION DEPENDENCIES AND CONFLICTS TO RESOLVE
----------------------------------------------------------------------

MEMBER 2: TRANSACTIONS AND RECOVERY

You need agreement on:

- Header/footer bytes and CRC coverage.
- Which function holds the mutex.
- Who implements Open.
- How pending transaction metadata becomes index entries.
- Who repairs failed appends.
- What happens after failed synchronization.
- How compaction switches descriptor and index.
- How full-cap space-reducing operations work.

Critical rule:

A public method acquires the mutex once.
Internal functions documented as already locked must not acquire it again.

Otherwise a call such as Put → Batch can deadlock if both independently lock the same mutex.


MEMBER 3: TORTURE TESTING

Provide:

- Stable executable invocation through Member 4.
- Explicit success/error semantics.
- Stable request IDs.
- Real concurrent engine invocation.
- Clear distinction between request ID and internal transaction ID.

Member 3 must independently model expected state, not reuse your index or recovery implementation.


MEMBER 4: CLI

Provide:

- Actual import path.
- Exact opener signature.
- Public API and error categories.
- OpenInfo/statistics semantics.
- Stable scan/check behavior.

The engine returns errors.
The CLI prints responses.

The engine should not print ACKs itself.


INTEGRATION RISKS AND PREVENTION

Risk:
Put and Batch use different formats.

Prevention:
One common transaction writer.


Risk:
Two implementations of Open.

Prevention:
Assign one coordinated integration path.


Risk:
CLI reads private index fields.

Prevention:
Public scan/stats interface.


Risk:
Compaction replaces file but offsets remain old.

Prevention:
Treat descriptor/index publication as one locked transition.


Risk:
Real I/O failure reported as clean rejection.

Prevention:
Explicit uncertain/poisoned state.


Risk:
Member 2 changes CRC coverage silently.

Prevention:
Format review required.


Risk:
Existing layout differs from planning.

Prevention:
Inspect before editing.


----------------------------------------------------------------------
K. IMPORTANT UNRESOLVED REQUIREMENTS
----------------------------------------------------------------------

Ask organizers or record explicit assumptions:

1. F4, PDF page 8:

   Is the removed byte guaranteed to belong to an unacknowledged tail?

   A checksum cannot recreate destroyed acknowledged bytes.

2. Does max_bytes count the temporary compaction file together with the old file, or only authoritative file size?

3. Is an initialization temporary allowed?

   The text explicitly permits compaction temporaries but does not separately discuss initial publication.

4. What exact bytes does “1 MB” mean?

   Supporting 1,048,576 bytes is a conservative proposed choice, not an official quotation.

5. Are 10,000 maximum-size values expected in one batch?

   Do not impose an undocumented combined payload limit.

6. What key/value sizes and history are used for the million-key recovery test?

These do not block pure codec work once format and API are approved, but they affect later edge-case guarantees.


----------------------------------------------------------------------
L. ACCEPTANCE CRITERIA FOR THESE TWO PHASES
----------------------------------------------------------------------

PHASE 0 IS COMPLETE WHEN:

- Repository and branch are inspected.
- Existing files and API are identified.
- Your OS and intended judging OS are known.
- You understand durable success versus in-flight uncertainty.
- Member 1/Member 2 boundaries are explicit.
- Outstanding organizer questions are recorded.


PHASE 1 IS COMPLETE WHEN:

- You approve or correct the proposed architecture.
- Actual repository API is compared with the planned API.
- Format is agreed with Member 2.
- Folder/file ownership is agreed.
- First implementation slice is selected.


INTENDED FIRST CODING SLICE

After approval and environment setup:

Shared contract, if absent and approved
    ↓
Pure binary format constants/codec
    ↓
Codec unit tests
    ↓
Checked low-level I/O helpers and tests

We should not begin with a “successful” Put that cannot yet reopen correctly.

Pure format/I/O foundations let Member 2 develop recovery against real, tested bytes without introducing misleading persistence claims.


======================================================================
CURRENT STATUS AND REQUIRED REPLY
======================================================================

Official PDF reviewed:
    Complete.

Relevant requirements extracted:
    Complete.

Repository inspected:
    Pending your output.

Branch verified:
    Pending.

Architecture approved by you/team:
    Pending.

Source files changed:
    None.

Go tests/builds executed:
    None.

Storage implementation complete:
    No.


REPLY WITH:

1. Repository, branch, and file-list command outputs.

2. Output from:

   go version
   go env GOOS GOARCH GOMOD GOTOOLCHAIN

3. Existing go.mod and API/format files, or confirmation they do not exist.

4. Your OS and the judging laptop’s OS.

5. Whether you approve the proposed architecture, and whether Member 2 has already started a different format or API.

We will resolve any mismatch before proceeding to Phase 2 and the first manageable implementation step.
```

---

## Prompt 3: Go Toolchain Setup & Environment Discovery
**Timestamp:** `2026-11-06T09:02:38Z`  
**Transcript Step Index:** `77`  

```text
# TEAM INNOVEXA — AUTOMATIC GO INSTALLATION AND CROSS-PLATFORM SETUP

We have four team members:
- Two use macOS.
- Two use Windows.
- The main implementation is handled by Member 1 (macOS) and one Windows teammate.
- The Windows teammate's laptop is our official hackathon demonstration machine.

Our project is Fsync-or-Die, written in Go.

## YOUR TASK: INSTALL GO AUTOMATICALLY

Do not merely tell me how to install Go. Detect the current computer's operating system and architecture, download the correct official Go installer using the terminal, install it, configure PATH if necessary, and verify the installation.

### Step 1 — Inspect the environment

Identify:
- Operating system: macOS or Windows.
- CPU architecture: ARM64 or AMD64.
- Whether Go is already installed.
- Existing Go version and installation location.
- Whether Homebrew or winget is available.
- Whether administrator permissions are required.

Do not replace a working Go installation unnecessarily.

### Step 2 — Download Go from the official source

Use only the official Go distribution:

https://go.dev/dl/

Select the latest stable release available when installation is performed.

Download the correct installer:
- macOS Apple Silicon: official ARM64 PKG.
- macOS Intel: official AMD64 PKG.
- Windows 64-bit Intel/AMD: official AMD64 MSI.
- Windows ARM: official ARM64 MSI.

Use terminal commands:
- macOS: curl and the official PKG installer.
- Windows: PowerShell and the official MSI installer.

Verify the downloaded installer's SHA-256 checksum against the checksum published on the official Go download page before installing.

Do not use unofficial download sites or unverified scripts.

### Step 3 — Install automatically

Perform the installation from the terminal.

If administrator privileges are needed:
- macOS: request sudo authorization.
- Windows: request an elevated administrator/UAC installation.

Never bypass administrator permissions.

Do not delete existing installations or modify unrelated environment settings.

### Step 4 — Verify Go

After installation, run:

```bash
go version
go env GOOS GOARCH GOMOD GOTOOLCHAIN GOPATH
```

On Windows, use PowerShell if necessary.

Confirm that the Go executable is accessible from the terminal. Refresh the terminal environment if required.

### Step 5 — Team version consistency

All four computers must use the same Go release for reproducible builds.

Record the exact installed Go version. Tell us which version the other three computers should install.

Our Windows demonstration laptop is the final acceptance machine. All project code must build and run correctly there.

### Step 6 — Cross-platform engineering rules

Our code will be developed on macOS and Windows, but demonstrated on Windows.

Therefore:
- Do not assume Linux-only behavior.
- Avoid Unix-only system calls in shared code.
- Handle file paths correctly on both platforms.
- Verify Windows-specific file locking and atomic file replacement behavior.
- Use OS-specific Go files and build tags when necessary.
- Do not claim cross-platform support without testing it.
- Keep dependencies within challenge restrictions.

### Step 7 — Protect the repository

Before and after installation, check:

```bash
git status --short
git branch --show-current
```

Do not change project source files, create go.mod, initialize packages, or start coding as part of this installation task.

We will handle architecture approval and implementation separately.

## FINAL REPORT

Tell me:
1. Detected operating system and architecture.
2. Whether Go was already installed.
3. Exact download and installation performed.
4. Exact Go version installed.
5. Actual verification command outputs.
6. Whether administrator permission was required.
7. Any errors or unresolved issues.
8. What exact Go version the other team members should use.

Do not claim success if installation or verification failed.

Start now. Perform the installation on this computer using the terminal. PHASE 0/1 REVIEW — PLATFORM CORRECTION AND CONDITIONAL APPROVAL

I am Member 1, the Tech Lead and Core Storage Engineer.

Our team has two Mac computers and two Windows computers.

I develop on the inspected Apple Silicon Mac. The live demonstration will happen on a Windows teammate’s laptop.

Do not assume that native macOS development means macOS is our judging target.

PLATFORM DECISION

Our preferred primary integration and demonstration environment is:

- Windows host.
- Ubuntu running under WSL2.
- Linux Go toolchain installed inside Ubuntu.
- Repository, database files, compaction temporaries, and torture artifacts stored inside the WSL Linux filesystem under /home/..., not /mnt/c/....
- Database executable and torture executable both built and run inside WSL.

This is our proposed target, pending verification that the laptop can run WSL2 and organizer confirmation that this demonstration setup is acceptable.

Do not install or configure another teammate’s machine automatically.

If WSL2 is unavailable or not permitted, stop and report the consequences before we choose a native Windows implementation. Do not silently reuse Unix atomic-rename and locking assumptions on Windows.

ARCHITECTURE APPROVAL

I approve the following design direction, subject to checking the official PDF and obtaining Member 2’s confirmation of the format/API:

- Append-only transaction log as the single authoritative database file.
- RAM key-to-offset index.
- One global database mutex.
- Common transaction mechanism for Put, Delete, and Batch.
- The previously proposed little-endian file/transaction/operation/footer layout.
- Exact full-transaction size accounting.
- Pure codec and tests as the first implementation slice.
- No persisted checkpoint initially.

CORRECTIONS TO YOUR REVIEW

1. Change “Approved” to “Proposed” or “Confirmed by Tech Lead” as appropriate. Do not claim Member 2 approved anything without their confirmation.

2. Do not describe temporary-file plus rename as universally crash-safe or portable.

   Platform-specific publication, synchronization, descriptor switching, and error handling must be designed and tested.

   Native Windows os.Rename must not be treated as having the same atomicity guarantee as Unix rename.

3. Do not claim existing committed data cannot be corrupted as an implementation result. This remains a design argument until code and tests support it.

4. Describe CRC coverage exactly:

   - File-header CRC over file-header bytes 0–27.
   - Transaction CRC over transaction header plus payload, stored in footer.
   - Footer CRC over footer bytes 0–27.

   The transaction header does not have a separate header-only CRC in this format.

5. Replace “100% test coverage” as an acceptance promise with explicit tests for byte layout, bounds, invalid fields, CRC failures, incomplete structures, and integer overflow.

6. Remote branches at the same initial commit do not establish that other members have no uncommitted or unpushed work. Ask them before freezing shared files.

7. Do not initialize go.mod with an arbitrary example module name.

   Confirm one exact shared module path before creating it.

OFFICIAL PDF

The PDF is available in our planning conversation but was not found in this workspace.

I will attach or copy the official PDF into this Antigravity session/workspace.

Until then:

- Do not claim to have independently read the PDF.
- Do not replace it with a generated summary.
- You may continue non-modifying environment inspection and identify blockers.

MAC ENVIRONMENT SETUP

My Go command is currently unavailable.

I choose the official macOS Apple Silicon Go installer for the agreed team Go version.

Do not install automatically.

Give me the official installation instructions, then wait for me to install it and share:

go version
go env GOOS GOARCH GOMOD GOTOOLCHAIN

Do not create go.mod or implementation files until:
- Go is verified.
- The actual shared module path is agreed.
- The contract and binary format are confirmed with Member 2.
- The first implementation slice is approved.

FIRST IMPLEMENTATION SLICE

After those gates are satisfied:

1. Create the agreed shared contract only if it is absent.
2. Implement pure format constants and codec.
3. Add explicit codec tests.
4. Run gofmt and affected tests.
5. Report actual output.
6. Stop for review before adding database file mutation.

Keep this slice platform-neutral.

Do not add Linux-only locking directly into portable source files. Future OS-specific implementations must be isolated through appropriate filenames/build constraints and explicit support policy.

Do not add a fake no-op locking implementation for an unsupported platform.

Do not implement transactions, recovery, compaction, or the CLI on behalf of the other members without coordination.

NEXT RESPONSE

Give me:
- Corrected Phase 0/1 status.
- Mac Go installation instructions.
- Exact outstanding decisions.
- A small first-slice plan.

Then stop. Do not generate or modify implementation files yet.
```

---

## Prompt 4: Member 1 Complete Core Implementation Mission
**Timestamp:** `2026-11-06T09:09:39Z`  
**Transcript Step Index:** `101`  

```text
FSYNC-OR-DIE — MEMBER 1 COMPLETE IMPLEMENTATION MISSION

TEAM: Team Innovexa
PROJECT: Fsync-or-Die
TAGLINE: “Don’t Trust the Process. Trust the Proof.”
ROLE: Member 1 — Tech Lead + Core Storage Engineer
ASSIGNED BRANCH: feat/storage-core

Act as a principal Go storage-engine engineer, database architect, reliability engineer, and practical programming mentor.

I approve proceeding with implementation of my assigned module. Do not stop after another planning report.

Your goal is to leave a real, readable, tested Member 1 implementation in this repository, together with precise integration instructions that I can give Members 2, 3, and 4.

Work incrementally internally, but continue through all feasible Member 1 milestones without asking me to approve each ordinary local coding step.

IMPORTANT:
No engineer or AI can honestly guarantee “10/10 accuracy,” zero bugs, or universal crash safety. Do not promise those things. Earn confidence through explicit invariants, executed tests, error-path testing, and truthful evidence.

If a real blocker prevents safe implementation, identify it precisely. Do not invent missing facts or hide blockers behind a claim of completion.


======================================================================
1. MY AUTHORIZATION AND ITS LIMITS
======================================================================

I authorize you to:

- Inspect the current repository.
- Read the official challenge PDF and existing project documents.
- Create and modify files belonging to Member 1 on feat/storage-core.
- Create the agreed root go.mod if it does not exist.
- Implement and test my storage foundation.
- Run local formatting, builds, tests, vet, race checks, and benchmarks where supported.
- Create local technical handoff documents in my ownership.
- Fix issues introduced by your own changes.
- Continue from one successful local implementation milestone to the next.

This authorization does NOT permit you to:

- Overwrite another member’s work.
- Modify another member’s implementation without coordination.
- Switch away from feat/storage-core without asking.
- Reset, clean, or discard existing work.
- Force-push.
- Merge branches.
- Commit or push automatically.
- Install system software with administrator privileges.
- Download or install unverified executables.
- Add external dependencies without explaining the need and obtaining approval.
- Weaken correctness to make tests pass.
- Create a fake production engine or pretend missing recovery exists.
- Claim tests passed when they were not executed.

For ordinary local Member 1 file edits, do not repeatedly ask “may I proceed?” I have approved that work.

For destructive actions, system installation, shared API conflicts, other members’ files, commits, pushes, or merges, stop and request the specific authorization required.


======================================================================
2. SOURCE OF TRUTH AND CURRENT CONTEXT
======================================================================

Priority order:

1. Official “Kill It and Prove It” PDF.
2. Explicitly approved shared architecture and interface.
3. Actual repository contents and agreed ownership.
4. Planning proposals below.

Previously reported workspace:

- Repository: /Users/madhuharsha/Core-Storage/Team-Innovexa
- Branch: feat/storage-core
- Initial commit: e68e909
- Tracked files: README.md only
- No go.mod
- No engine implementation
- Go previously unavailable on PATH
- Mac development machine: Apple Silicon
- Planned primary demo environment: Ubuntu under WSL2 on a Windows teammate’s laptop
- Demo project/data should be in the Linux filesystem under /home/..., not a mounted Windows working directory

These are previous observations. Reinspect them; do not assume they remain current.

Do not repeat an unverified operating-system version, installer checksum, installed toolchain, or filesystem claim as an established fact.

The PDF was previously absent from the local workspace. Search again.

If the PDF is still unavailable:
- Report that clearly.
- Do not claim to have read it.
- Implement only work justified by the provided specification and approved format.
- Mark direct PDF verification pending.
- Do not claim full challenge compliance until the official document is actually checked.

If Go is missing:
- Check whether it exists outside the current PATH.
- Do not claim successful tests or builds.
- Do not run administrator installation automatically.
- Give exact, verified official installation instructions.
- Verify installer URLs/checksums from official sources before using or recommending them.
- Do not reuse unverified hashes from earlier generated text.
- Prepare appropriate source files if safe, but report executable verification as blocked.
- Once Go is available, continue and execute verification.


======================================================================
3. FIRST ACTION: INSPECT, THEN IMPLEMENT
======================================================================

Run and inspect:

git rev-parse --show-toplevel
git branch --show-current
git status --short
git branch --list
git log -5 --oneline
git ls-files
git ls-files --others --exclude-standard

Inspect:

go version
go env GOOS GOARCH GOMOD GOTOOLCHAIN

Read:

- README.md
- go.mod, if present
- Existing public contracts
- Existing storage format
- Existing engine/transaction/recovery code
- Existing tests
- Architecture and ownership documents

Check whether the official PDF is present.

Do not display secrets.

If the branch is not feat/storage-core, or existing changes conflict with your planned edits, stop before modifying those files.

Otherwise provide a brief inspection summary and immediately begin the first safe implementation slice. Do not spend the whole response repeating the plan.


======================================================================
4. MY ROLE — IMPLEMENT THIS, NOT ANOTHER MEMBER’S MODULE
======================================================================

MEMBER 1 OWNS

- Shared engine API coordination.
- Storage format and binary codec.
- Checked length/offset arithmetic.
- Key/value/operation validation.
- Checked file I/O primitives.
- RAM key-to-offset index.
- Database state and locking foundation.
- File lifecycle foundation, coordinated with recovery.
- Put/Get/Delete public entry points.
- Capacity accounting.
- Closed/poisoned/error state handling.
- Core storage tests.
- Member 1 integration handoff.

MEMBER 2 OWNS

- Atomic batch transaction execution.
- Commit pipeline.
- Recovery scan/replay.
- Torn-tail repair.
- Crash-safe compaction.
- Transaction/recovery/compaction tests.

MEMBER 3 OWNS

- Independent torture tool.
- Workload generation.
- Independent request/ACK oracle.
- Force-kill and restart campaigns.
- Seeds and failure artifacts.

MEMBER 4 OWNS

- CLI executable.
- Command parsing.
- Serve transport.
- Machine-readable output.
- CLI integration tests.
- Documentation coordination.

CRITICAL:
Put, Delete, and Batch must use one common transaction path.

Do not create:
- A special Put record format.
- A second transaction implementation.
- A duplicate recovery scanner.
- A production in-memory fake standing in for durability.
- A CLI inside my storage-core branch.

If a Member 2 dependency is not implemented:
- Define and document the smallest stable integration boundary.
- Use test doubles only in test files.
- Make missing production behavior explicit.
- Never return durable success for an operation that has not actually committed.

If safe completion of a public method depends on Member 2, implement the Member 1 portion and report the integration dependency accurately.


======================================================================
5. ARCHITECTURE TO IMPLEMENT
======================================================================

Use the existing approved architecture if present. Otherwise use this baseline:

- Single authoritative append-only transaction-log file.
- The log itself is the database.
- In-memory map from key bytes to current value offset and length.
- One global DB mutex.
- Put/Delete/Batch share one transaction envelope.
- Complete transaction footer.
- CRC32-IEEE validation.
- Final synchronization before mutation success.
- Recovery validates complete transactions before applying them.
- Compaction publishes a complete replacement.
- No persisted index/checkpoint initially.

Do not prematurely split tightly coupled state into many packages.

Prefer a small src/kv package with clearly owned files, unless the repository already uses another approved structure.

Use the actual repository layout rather than renaming working files merely to match a suggested diagram.


======================================================================
6. SHARED MODULE PATH
======================================================================

Use the existing go.mod if present.

If absent:

1. Inspect the Git remote without printing credentials.
2. If the actual canonical repository path is clear, use the corresponding agreed GitHub module path.
3. If no approved canonical module path can be established, use the previously proposed local module path:

   innovexa/fsync-or-die

4. Record the choice clearly in the handoff.
5. Do not create nested Go modules.

Use the agreed Go toolchain version after verifying its availability and the team’s actual decision.

Do not silently trigger a toolchain download during offline verification.

Do not claim the go directive is an exact installed-version lock.


======================================================================
7. EXACT BINARY FORMAT BASELINE
======================================================================

All integers: little-endian.

Use explicit field encoding with encoding/binary.
Do not serialize native Go struct memory or depend on padding.

FILE HEADER — 32 BYTES

Offset  Size  Field
0       8     Magic: KIPDB001
8       4     Format version: 1
12      4     Header length: 32
16      8     Maximum file bytes; zero means unlimited
24      4     Flags; initially zero
28      4     CRC32-IEEE over bytes 0–27

TRANSACTION HEADER — 32 BYTES

Offset  Size  Field
0       4     Magic: TXN1
4       2     Transaction version: 1
6       2     Flags: zero
8       8     Transaction ID
16      4     Operation count
20      4     Reserved: zero
24      8     Payload byte length

OPERATION

[1][key length:2][value length:4][key][value]

Kind 1: Put.
Kind 2: Delete.

Delete requires zero value length.
Put with zero value length is valid.
Duplicate keys apply in operation-list order.

COMMIT FOOTER — 32 BYTES

Offset  Size  Field
0       8     Magic: COMMIT01
8       8     Matching transaction ID
16      8     Total transaction bytes
24      4     CRC32-IEEE over transaction header plus payload
28      4     CRC32-IEEE over footer bytes 0–27

TRANSACTION SIZE

64 + sum(7 + keyBytes + valueBytes)

CRC COVERAGE MUST BE EXACT

- File header has its own CRC.
- Transaction header and payload share a CRC stored in footer.
- Transaction header does not have a standalone header-only CRC.
- Footer has its own CRC.

VALIDATION

- Correct magic.
- Supported version.
- Known flags and operation kinds.
- Reserved fields valid.
- Key limits.
- Value limits.
- Operation-count limits.
- Checked size arithmetic.
- Safe uint64/int64 conversions.
- Exact payload consumption.
- Complete footer.
- Matching IDs.
- Matching total length.
- Correct CRCs.

Do not accept a commit marker without validating the whole transaction.

Reject unsupported formats rather than guessing how to skip them.

Do not silently impose an undocumented small total batch-byte limit.


======================================================================
8. STABLE PUBLIC CONTRACT
======================================================================

Inspect existing definitions first.

Previously proposed public shape:

func Open(path string, opts Options) (Engine, OpenInfo, error)

type Engine interface {
    Put(key, value []byte) error
    Get(key []byte) ([]byte, error)
    Delete(key []byte) error
    Batch(ops []Op) error

    Compact() error
    Check() error
    Scan(fn func(key, value []byte) error) error
    Stats() Stats

    Close() error
}

type Kind uint8

const (
    PutOp Kind = 1
    DeleteOp Kind = 2
)

type Op struct {
    Kind Kind
    Key []byte
    Value []byte
}

Previously proposed errors:

ErrNotFound
ErrInvalid
ErrFull
ErrClosed
ErrBusy
ErrCorrupt
ErrVersion
ErrConfig
ErrUncertain
ErrPoisoned

Do not silently rename these to ErrFullCap or ErrCorruptRecord if the team is already using the original names.

Use errors.Is-compatible wrapping.

Define and document Options, OpenInfo, and Stats consistently with the existing approved contract.

If no approved definitions exist, use the previous full shared-contract proposal where available; avoid inventing another competing API.

SEMANTICS

- nil mutation error means durable success.
- Missing Get returns ErrNotFound.
- Missing Delete succeeds.
- Empty Put value remains present.
- Caller may mutate inputs only after method returns.
- Get returns caller-owned bytes.
- Batch list order determines repeated-key behavior.
- Public methods are thread-safe.
- Scan callback must not reenter DB.
- Failed cap preflight means definitely absent.
- Uncertain I/O must not be represented as a definite rejection.
- Closing/poisoning semantics must be explicit.

Document which extensions are needed by CLI and compaction rather than pretending they are six additional official required library operations.


======================================================================
9. REQUIRED MEMBER 1 IMPLEMENTATION MILESTONES
======================================================================

MILESTONE A — CONTRACT AND PURE CODEC

Implement:

- Public shared types and errors, if absent.
- Fixed layout constants.
- File-header encoding/decoding.
- Transaction-header encoding/decoding.
- Operation metadata encoding/decoding.
- Commit-footer encoding/decoding.
- Exact CRC coverage.
- Checked encoded-size calculation.
- Validation helpers.

Use APIs that support Member 2 streaming large payloads.

Avoid requiring every transaction to become one giant byte slice.

Add meaningful tests before proceeding.


MILESTONE B — CHECKED FILE I/O

Implement testable helpers for:

- Full positioned writes.
- Exact positioned reads.
- Zero-progress detection.
- Partial-progress errors.
- Offset overflow.
- File synchronization error propagation.
- Safe truncation interface where needed by Member 2.

Do not use WriteAt on an O_APPEND-opened file.

Use a small testable I/O seam instead of an unnecessarily elaborate virtual filesystem.

Test doubles belong in tests.


MILESTONE C — RAM INDEX

Implement:

- Byte-safe keys.
- Key-to-offset/length entries.
- Put/overwrite metadata application.
- Delete metadata application.
- Ordered pending-transaction publication support.
- Stable scan support as agreed.
- Caller-owned returned data.

Do not store all values in the map.

Document mutex ownership.


MILESTONE D — DB STATE AND OWNERSHIP

Implement the coordinated DB foundation:

- File descriptor.
- Stable process-ownership lock descriptor.
- Path.
- Index.
- Committed end.
- Next transaction ID.
- Configured capacity.
- Closed state.
- Poisoned state.
- One mutex.

Clearly define public locking methods versus internal already-locked helpers.

Do not double-lock through Put → Batch.

Implement platform-sensitive ownership code only with explicit supported-platform policy.

The primary integration target is Linux under WSL2.
My local Mac may need a separate Darwin implementation for local engine tests.

Use platform-specific files/build constraints correctly.
Do not assume a filename ending _unix.go automatically selects Unix platforms.
Use explicit supported OS suffixes or valid build constraints.

Do not provide a fake successful no-op lock on an unsupported OS.

Do not claim native Windows support without implementing and testing it.


MILESTONE E — FILE LIFECYCLE FOUNDATION

Coordinate with Member 2:

- Safe open flags.
- Header initialization.
- Existing-header validation.
- Stored-cap policy.
- Recovery invocation boundary.
- Serialized close.
- Error-preserving cleanup.

Never use truncating creation to open an existing database.

Interrupted initialization must have a defined recovery policy.

Do not label an Open implementation complete if it can read only cleanly closed databases while crash recovery is still missing.


MILESTONE F — PUBLIC GET / PUT / DELETE INTEGRATION

Get:
- Lock.
- Check state.
- Lookup index.
- Read exact value bytes.
- Return owned data.
- Preserve real I/O errors.

Put/Delete:
- Validate appropriately.
- Delegate to the common transaction path.
- Do not independently serialize/commit a second format.
- Never report success before the coordinated commit implementation completes.

If Member 2’s production commit path is missing, clearly expose the integration boundary and do not fake durability.


MILESTONE G — CAPACITY AND ERROR STATE

Implement:

- Exact encoded-byte accounting.
- Cap includes header and all transaction metadata.
- Checked subtraction/comparison.
- Clear ErrFull semantics.
- Poisoned-handle rejection.
- Uncertain outcome classification support.

Coordinate the full-file delete/reducing replacement case with Member 2.

Do not pretend append-only tombstones require no free bytes.


MILESTONE H — TESTS, BENCHMARKS, AND HANDOFF

Complete my owned tests and benchmarks.

Create a Member 1 handoff document describing actual:

- Exported symbols.
- File-format fields.
- Internal integration functions.
- Mutex rules.
- I/O interfaces.
- Error meanings.
- Pending Member 2 dependencies.
- CLI integration expectations.
- Tested platforms.
- Unverified behavior.

Do not mark dependency-owned features as completed by me.


======================================================================
10. DURABILITY AND FAILURE INVARIANTS
======================================================================

For the agreed common transaction path:

1. Lock.
2. Validate all operations.
3. Calculate exact size.
4. Check capacity.
5. Save transaction start.
6. Write header.
7. Stream payload.
8. Write complete footer.
9. Final Sync succeeds.
10. Publish complete index changes.
11. Advance state.
12. Return success.
13. CLI may ACK.

Confirm this ordering with Member 2.

If code uses buffering, flush user-space buffers before File.Sync.

A complete transaction may recover even if no ACK was observed.

On real Sync failure after footer:
- Do not ACK.
- Do not claim definite absence.
- Prevent unsafe continuation.
- Preserve uncertainty explicitly.

On failed append:
- Do not update committed index.
- Coordinate tail rollback/repair.
- Do not append new transactions behind an unresolved tail.

Compaction:
- New offsets must accompany new descriptor.
- Never append through old descriptor after replacement.
- Rename success followed by another error is not the same as no publication.
- Cleanup must not delete an already published authoritative file.

Do not claim power-loss safety based on SIGKILL testing.


======================================================================
11. TEST REQUIREMENTS
======================================================================

CODEC TESTS

- Exact field offsets and widths.
- Little-endian bytes.
- Known expected byte vectors, not only round trips.
- Empty and binary values.
- Minimum and maximum keys.
- Maximum supported value.
- Invalid operation kinds.
- Delete with nonzero value length.
- Invalid version/flags/reserved fields.
- Header/payload/footer truncation.
- CRC mutation tests.
- ID and length mismatch.
- Integer overflow and conversion bounds.
- Duplicate-key operation ordering where relevant.

I/O TESTS

- Partial progress.
- Zero progress.
- Error before writing.
- Error after partial writing.
- Unexpected EOF.
- Invalid offsets.
- Sync error propagation.

INDEX/API TESTS

- Put/Get metadata.
- Overwrite.
- Delete.
- Missing key.
- Empty value differs from deletion.
- Binary key/value.
- No borrowed mutable result.
- Closed/poisoned state.
- Concurrent callers.
- Capacity boundary.

INTEGRATION TESTS WHEN MEMBER 2 IS AVAILABLE

- Clean reopen.
- Crash-produced tail.
- Final Sync before ACK.
- Atomic batch publication.
- Recovery rebuilds offsets.
- Compaction descriptor/index switch.
- Full-cap failure leaves previous state valid.

Do not call a clean reopen test a force-kill test.

Do not reuse engine replay logic as the independent torture oracle.


======================================================================
12. TEST EXECUTION AND EVIDENCE
======================================================================

Run appropriate commands as code becomes available:

gofmt on only the files you own/change
go test -count=1 -v ./src/kv
go vet ./...
go test -race -count=1 ./src/kv
go test -run '^$' -bench . -benchmem ./src/kv

Adapt paths to actual repository layout.

Run a broader go test ./... once integrated packages exist.

If a command cannot run:
- Say why.
- Distinguish environment failure from code failure.
- Do not present an expected result as observed output.

If a test fails:
- Diagnose.
- Fix the relevant issue.
- Rerun.
- Retain useful evidence.

Do not delete meaningful tests or relax assertions simply to obtain green output.

Coverage percentage is supplemental, not proof.

No invented throughput, memory, or recovery numbers.


======================================================================
13. PARALLEL TEAM SUPPORT — IMPORTANT
======================================================================

I need to guide the other members.

As soon as the contract and codec are stable, provide a concise copy-paste handoff for:

MEMBER 2
- Exact import path and types.
- Codec calls.
- CRC coverage.
- Pending metadata representation.
- Mutex rules.
- Append and recovery integration points.
- Expected error semantics.
- Who owns Open/Batch/Compact.

MEMBER 3
- What is implemented versus pending.
- Engine/CLI success semantics.
- Required binary-data cases.
- Failure injection support if available.
- No requirement to import private storage structs.

MEMBER 4
- Actual opener and Engine API.
- Error mappings.
- OpenInfo/Stats.
- Scan/Check/Close behavior.
- No ACK before nil mutation result.
- Missing/uncertain errors distinguished.

Do not invent CLI or torture commands that are not implemented.

Provide these handoffs while completing my module, not only after the whole team finishes.


======================================================================
14. GIT AND DOCUMENTATION RULES
======================================================================

- Stay on feat/storage-core.
- Preserve existing work.
- Keep changes scoped.
- Do not format or modify unrelated files.
- Do not commit/push/merge without explicit instruction.
- Prepare suggested small commit messages.
- Show final diff summary.
- Keep prompts/evidence truthful.

Do not overwrite Member 4’s README/ARCHITECTURE/BREAKING work.

If those shared documents do not exist, provide proposed sections in a Member 1-owned handoff file rather than claiming the final team documents are complete.

Record actual observations, not fictional failures or tests.


======================================================================
15. OUTPUT STYLE WHILE WORKING
======================================================================

Do not stop after a plan.

For each milestone:
- Briefly explain the purpose.
- Identify files.
- Make approved local Member 1 changes.
- Explain important code.
- Execute relevant tests if possible.
- Report actual results.
- Continue to the next safe milestone.

Do not repeatedly paste the entire project into chat if you can edit the actual workspace. Show useful summaries and critical code excerpts, with exact file paths.

If you cannot edit the workspace, say so and provide complete files in manageable groups, with exact paths and execution instructions.

Never claim a file was created if you only displayed its contents.

Stop only for:
- Missing required system software that needs my action.
- Wrong branch or conflicting existing work.
- Unresolvable shared API/ownership conflict.
- A required external decision.
- A dangerous/destructive action.
- A blocker preventing truthful or safe continuation.


======================================================================
16. FINAL DELIVERY REPORT
======================================================================

At the end, provide:

1. Changed/new files.
2. Actual implemented capabilities.
3. Actual test commands and results.
4. Build/vet/race status.
5. Benchmarks actually executed.
6. Supported/tested platforms.
7. Features pending Member 2.
8. Features pending CLI/torture integration.
9. Known limitations and bugs.
10. Exact reproduction commands.
11. Member 2 handoff.
12. Member 3 handoff.
13. Member 4 handoff.
14. Suggested commit messages.
15. Suggested PR description.
16. Clear completed/incomplete checklist.

Use these statuses:

IMPLEMENTED AND TESTED
IMPLEMENTED BUT UNVERIFIED
BLOCKED BY ENVIRONMENT
PENDING MEMBER 2
PENDING TEAM INTEGRATION
KNOWN FAILING

Do not label the whole storage engine “complete” if production transaction execution, recovery, or required integration remains missing.


======================================================================
17. BEGIN NOW
======================================================================

Reinspect the repository and toolchain.

Then implement every safe, authorized Member 1 component through the milestones above, with real tests and clear integration boundaries.

Do not ask for another blanket approval.
Do not make unsupported “will work without errors” promises.
Do not take over other members’ modules.

My priority is a small, correct, understandable storage foundation that the team can integrate and defend during judging.

START WITH THE ACTUAL WORKSPACE NOW.
```

---

## Prompt 5: Toolchain Verification & Physical Execution Gate
**Timestamp:** `2026-11-06T09:30:49Z`  
**Transcript Step Index:** `208`  

```text
FSYNC-OR-DIE — STOP REPORT LOOP; COMPILE AND VERIFY ACTUAL CODE

I am Member 1 on feat/storage-core.

Your corrected report is useful, but it is not verification. Do not produce another “delivery complete” report while Go is unavailable.

The next objective is actual compilation and executed tests, followed by a precise handoff to Member 2.

Do not implement more optional architecture, group commit, hooks, or native Windows support.

Do not commit, push, merge, discard work, or modify another member’s files.


1. VERIFY THE TOOLCHAIN FIRST

Run:

command -v go
go version

If not found, check:

/usr/local/go/bin/go version

If that executable exists, use it or set PATH for the current terminal session:

export PATH="/usr/local/go/bin:$PATH"

Then run:

go version
go env GOOS GOARCH GOMOD GOTOOLCHAIN

If Go is genuinely absent:
- Stop pretending executable verification is possible.
- Tell me installation is the only immediate external blocker.
- Direct me to the official macOS ARM64 package on go.dev.
- Do not recommend an old patch merely because go.mod has a lower minimum version.
- Do not repeat unverified hashes.
- Do not use Markdown link syntax inside shell commands.
- Do not run sudo automatically.
- Continue only the targeted source inspection below while waiting.


2. PRIORITY SOURCE CHECK: SyncWrites

Inspect actual uses of Options.SyncWrites.

Do not assume false is safe.

Our public mutation contract requires durable success for every successful Put, Delete, and Batch.

Therefore:
- Options{} must not silently disable synchronization.
- A test or benchmark must not disable synchronization while presenting its results as durable-write performance.
- Check every branch that can return nil after mutation.

Preferred simple production policy:
- Every nonempty successful mutation uses the required final file synchronization.
- Do not expose an unsafe durability-off mode in the challenge engine.

If changing/removing the exported field affects other members:
- Identify the exact change.
- Coordinate it explicitly.
- Do not silently break their callers.
- At minimum, do not let false bypass the durability guarantee.

Add an injected-I/O test proving that:
- Default options reach final Sync.
- Sync error produces no successful mutation result.
- The handle cannot continue unsafely after an uncertain failure.


3. PRIORITY SOURCE CHECK: CAP CONFIGURATION

The current field:

MaxFileBytes uint64

cannot by itself distinguish:
- “No cap override supplied.”
- “Explicitly request unlimited capacity.”

Inspect existing behavior.

Required practical result:
- Reopening through a normal CLI command must preserve the stored cap.
- Options{} must not accidentally reset an existing database to unlimited.
- An explicitly conflicting cap must have documented behavior.
- Validate conversions before using uint64 values as int64 file offsets.

Use the simplest agreed policy. Avoid changing the API unnecessarily.

Add tests:
- Create with small cap.
- Close.
- Reopen with default options.
- Confirm cap still applies.
- Confirm exact-fit and over-cap behavior.


4. PRIORITY SOURCE CHECK: RECOVERY

Verify the implementation, not the report:

- Complete valid transactions are replayed even if prior ACK was not observed.
- No index updates before full transaction validation.
- Incomplete tail is repaired before another append.
- Repair errors prevent unsafe further writes.
- Arbitrary unsupported/corrupt structure is not silently treated as permission to erase valid history.
- ReadOnly mutation attempts are rejected before touching the file.
- ReadOnly behavior does not become another undocumented production mode that undermines normal recovery.

Do not say a caller “received ErrUncertain” when the process was killed before it could return. Distinguish:
- Actual returned Sync error.
- Abrupt termination with no response.


5. PRIORITY SOURCE CHECK: OPEN AND CLOSE

Verify:
- Existing database is never truncated by opening.
- Interrupted initialization has an explicit policy.
- Failed Open releases file descriptors and ownership locks.
- Close is serialized.
- Close errors are preserved.
- Operations after Close return the correct error.
- A poisoned handle does not accept mutations.


6. EXECUTE TESTS ONCE GO IS AVAILABLE

Stay on feat/storage-core.

Format only owned files.

Run:

go test -count=1 -v ./src/kv

Fix compilation errors first, then failing tests.

Then run:

go vet ./...
go test -race -count=1 ./src/kv
go test -count=1 ./...

If a command is blocked by environment, report the actual error.

Do not delete tests or weaken assertions to obtain PASS.

Do not repeat unverified coverage or scalability claims.


7. FREEZE ONE ACTUAL CONTRACT

After compilation, provide the actual contents/signatures for:

- Module path.
- Open.
- Options.
- OpenInfo.
- Engine.
- Op and Kind.
- Errors.
- Scan callback rules.
- Close behavior.

OpenInfo currently lacks a recovery-duration field.

Coordinate a single timing policy with Member 4:
- Either engine exposes measured open/recovery duration,
- Or CLI measures a clearly documented Open interval.

Do not claim the required CLI timing output is already implemented.


8. HAND OFF EXISTING TRANSACTION/RECOVERY CODE TO MEMBER 2

You have already written commitBatchLocked and recoverAndRepairLocked.

Do not ask Member 2 to independently build a second transaction writer or recovery scanner.

Produce exact source-level handoff:
- Existing functions and their signatures.
- Which require DB mutex held.
- State fields they modify.
- Commit ordering.
- Parser validation.
- Tail-repair behavior.
- Remaining weaknesses.
- Tests already executed.
- Member 2’s remaining ownership.

Member 2 should review and extend the existing coherent path, with coordination, rather than create incompatible duplicates.

Compaction and full-cap reducing replacement remain incomplete until actually implemented and tested.


9. TEST ON THE DEMO ENVIRONMENT

After local Mac tests pass, give exact instructions for the Windows teammate to run inside WSL2 Ubuntu:

go version
go env GOOS GOARCH
go test -count=1 -v ./src/kv
go vet ./...
go test -race -count=1 ./src/kv

The source must first be transferred through our agreed reviewed Git workflow.

Do not claim Linux runtime verification from Mac tests or cross-compilation.

Do not execute a Linux test binary on macOS.


10. FINAL RESPONSE REQUIREMENTS

Return only useful evidence:

A. Actual Go version/environment.
B. Actual compilation errors found and fixed.
C. Actual test commands and output summaries.
D. SyncWrites/default durability result.
E. Stored-cap reopen result.
F. Remaining failing or unverified behavior.
G. Exact Member 2 handoff.
H. Whether code is ready for a reviewed integration PR.

If Go is still unavailable, state:
“EXECUTION BLOCKED: install Go before further verification.”

Do not label the engine complete, crash-safe, or fully tested based on source inspection alone.

Start with toolchain verification now.
```

---

## Prompt 6: Member 1 Local Commit Authorization
**Timestamp:** `2026-11-06T09:51:38Z`  
**Transcript Step Index:** `334`  

```text
FSYNC-OR-DIE — AUTHORIZE MEMBER 1 COMMIT AND PREPARE TEAM HANDOFF

I approve staging and committing the reviewed Member 1 foundation to:

feat/storage-core

This approval is for a LOCAL COMMIT ONLY.

Do not push, merge, force-push, switch branches, or alter other members’ work.

Do not start another broad audit or architecture rewrite. Finish the narrow corrections below, execute the verification gate, commit the scoped files, and give me the handoffs.


1. CONFIRM CURRENT STATE

Run:

git branch --show-current
git status --short

Proceed only if the branch is feat/storage-core.

Inspect the current source and file list again before staging.

Stop if you find conflicting changes, another member’s files, or unexpected staged content.


2. CORRECT REPORTING ERRORS

The pasted test log lists 30 top-level tests, not 19.

Verify using:

go test -list '^Test' ./src/kv

Update HANDOFF_MEMBER1.md and the proposed PR description to match actual results.

Do not describe an injected function error as an “injected physical sync failure.”

Use:

“Injected Sync error on the real transaction call path.”

Do not claim hardware failure was physically reproduced.

Correct benchmark size claims using actual source and the format:

TransactionBytes = 64 + sum(7 + keyLength + valueLength)

For the stated 9-byte key and 19-byte value, the transaction is 99 bytes, not 78.

For Batch100, inspect actual generated keys and values. k0..k99 and v0..v99 do not all have two-byte lengths.

Do not rerun benchmarks merely to correct arithmetic unless the implementation changed.

Remove unsupported “exact timing” language. Describe precisely which interval RecoveryDurationNs measures.

Keep these limitations explicit:
- Mac runtime tests reported/executed locally.
- Linux cross-compilation is not Linux runtime verification.
- Native Windows is unsupported.
- Compaction incomplete.
- Full-cap reducing replacement incomplete.
- Force-kill campaign incomplete.
- Million-key recovery benchmark incomplete.


3. FINALIZE COMPACTION HANDOFF

Ensure HANDOFF_MEMBER1.md includes:

- Hold db.mu before snapshot selection and through publication.
- Preserve old authoritative file while building temporary.
- Write complete valid snapshot and replacement index.
- Sync temporary before replacement.
- Rename directly over authoritative pathname.
- Retain the already-open temporary descriptor for active use.
- Include parent-directory synchronization in the agreed Linux publication protocol.
- Switch descriptor, index, committed end, and transaction metadata consistently.
- Close old descriptor after successful switch.
- Correctly handle errors occurring after rename succeeded.
- Never delete a published authoritative file during cleanup.
- Never append through old descriptor after publication.

Mark platform-specific publication behavior as a design to implement and verify, not completed compaction.

Do not implement Member 2’s remaining work in this step.


4. RUN THE COMMIT GATE

Ensure the local Go executable is on PATH.

Run:

go test -count=1 -v ./src/kv
go vet ./...
go test -race -count=1 ./src/kv
go test -count=1 ./...

Check formatting of owned Go files and correct only those files if needed.

Report actual command results.

If a test fails, fix the relevant Member 1 defect and rerun.

Do not proceed to commit if the gate fails or execution is blocked.

Do not weaken tests to obtain PASS.


5. REVIEW AND STAGE ONLY THE SCOPED FILES

Expected allowed files:

go.mod
HANDOFF_MEMBER1.md
src/kv/contract.go
src/kv/format.go
src/kv/format_test.go
src/kv/io.go
src/kv/io_test.go
src/kv/index.go
src/kv/index_test.go
src/kv/capacity.go
src/kv/capacity_test.go
src/kv/lock_darwin.go
src/kv/lock_linux.go
src/kv/lock_windows.go
src/kv/db.go
src/kv/db_test.go
src/kv/get.go
src/kv/put_delete.go
src/kv/bench_test.go

Stage existing approved files explicitly.

Do not use git add . or git add -A.

Do not stage:
- Database files.
- Lock files.
- Installers.
- Compiled binaries.
- Credentials.
- Unrelated changes.
- Another member’s files.

After staging, inspect:

git diff --cached --stat
git diff --cached --check
git diff --cached

If git diff --cached --check reports issues, fix them before committing.


6. CREATE THIS LOCAL COMMIT

If verification succeeds and the staged diff is correct, commit with:

feat(storage-core): add tested storage foundation and integration contract

Do not claim the commit includes complete compaction, full-cap recovery, or verified process-crash correctness.

If Git identity is missing, stop and ask me for the identity to use. Do not invent it.

Do not bypass hooks if a hook fails.


7. RETURN LOCAL COMMIT EVIDENCE

After committing, run:

git log -1 --oneline
git status --short

Report:
- Actual commit hash.
- Actual committed file list.
- Whether working tree is clean.
- Verification commands and results.
- Remaining limitations.

Do not say the team has received the code yet. A local commit is not a push.


8. GIVE ME THREE SHORT COPY-PASTE HANDOFFS

MEMBER 2 — TRANSACTIONS, RECOVERY, COMPACTION

Include:
- Exact module/import path.
- Existing commitBatchLocked and recoverAndRepairLocked signatures.
- Which files already contain transaction/recovery logic.
- One format and one commit path; no competing implementation.
- Mutex preconditions.
- Compaction implementation ownership.
- Full-cap reducing snapshot ownership.
- Actual tests and remaining crash/performance verification.
- Need to coordinate existing Compact method location before adding another method declaration.

MEMBER 3 — TORTURE TESTING

Include:
- Foundation commit hash.
- Current engine status.
- Real CLI/serve integration dependency.
- Independent submitted-request and ACK logs.
- Valid unresolved operations may recover completely.
- Concurrent request/response ordering must be handled correctly.
- No imports of private replay/index logic.
- Required targeted modes and retained failure artifacts.

MEMBER 4 — CLI AND DOCUMENTATION

Include:
- Foundation commit hash.
- Exact current Open signature.
- Actual Options/OpenInfo/Stats/Engine definitions or precise source references.
- Error categories.
- Missing standalone get prints NOTFOUND.
- ACK only after nil durable mutation result.
- Recovery timing semantics.
- Scan callback/ordering rules.
- WSL2 Linux demonstration target.
- Need to agree serve framing with Member 3.


9. PREPARE, BUT DO NOT EXECUTE, SHARING STEPS

Show me the exact proposed push command for feat/storage-core.

Do not run it yet.

Prepare a PR description with:
- Implemented scope.
- Actual executed test evidence.
- Platform verification boundaries.
- Pending compaction/full-cap/crash/scale work.

The eventual integration path remains:

local commit
→ push feat/storage-core after my approval
→ reviewed PR into main
→ members update their feature branches from integrated main
→ WSL2 runtime verification and team integration

Do not merge automatically.


FINAL RESPONSE

Keep it practical:

A. Corrections made.
B. Actual verification results.
C. Local commit hash and working-tree status.
D. Three teammate handoffs.
E. Proposed push command.
F. Remaining blockers.

Proceed with the authorized local work now.
```

---

## Prompt 7: Push feat/storage-core to Origin
**Timestamp:** `2026-11-06T09:57:05Z`  
**Transcript Step Index:** `359`  

```text
FSYNC-OR-DIE — SHARE MEMBER 1 FOUNDATION AND UNBLOCK THE TEAM

My Member 1 foundation is reported committed locally as:

0a4f44a feat(storage-core): add tested storage foundation and integration contract

I authorize pushing the verified feat/storage-core branch to its existing origin repository after the checks below.

I also authorize opening a pull request into main if authenticated GitHub tooling is already available.

DO NOT merge the PR.
DO NOT force-push.
DO NOT delete any branches.
DO NOT switch branches.
DO NOT modify another member’s work.
DO NOT install GitHub tooling automatically.
DO NOT initiate another broad architecture rewrite.


1. VERIFY THE LOCAL COMMIT AND DESTINATION

Run:

git branch --show-current
git status --short
git log -1 --oneline
git remote -v

Verify:
- Current branch is feat/storage-core.
- Expected foundation commit is present.
- Working tree has no unexpected changes.
- Origin is our intended Team Innovexa repository.

Do not print credentials embedded in remote URLs.

If branch, commit, or destination differs unexpectedly, stop and explain before pushing.


2. FIX ONLY REAL HANDOFF ERRORS

Inspect HANDOFF_MEMBER1.md.

Correct these statements if they exist in the actual file:

A. Standalone get:
   Success prints the actual value followed by a newline.
   Missing key prints exactly:

   NOTFOUND\n

   The textual CLI behavior is separate from the binary-safe extension agreed with Member 3.

B. Serve success:
   Print exactly:

   ACK <op-id>\n

   Replace <op-id> with the real submitted request ID.
   Do not print an empty ACK.
   ACK only after the engine returns durable success.

C. Recovery:
   Complete valid transactions may recover without an observed ACK.
   Incomplete final transactions are rejected/repaired.
   Being killed during Sync does not by itself mean the transaction is absent.

D. Compaction handoff:
   Preserve the complete protocol in HANDOFF_MEMBER1.md:
   - Hold db.mu before snapshot selection.
   - Keep it held through replacement publication.
   - Finish and Sync the replacement.
   - Rename directly over the authoritative pathname.
   - Include the agreed parent-directory synchronization step.
   - Switch active descriptor/index/boundary metadata consistently.
   - Close old descriptor after publication/state switch.
   - Handle post-rename errors without deleting published authority.

E. Scalability:
   “Index stores offsets rather than values” is implemented.
   “One million keys verified” remains unverified until measured.
   Do not describe design intent as an executed scale test.

If the actual file is already correct and only chat formatting was damaged, do not create a pointless edit.

If a documentation correction is needed:
- Modify only the owned handoff document.
- Run git diff --check.
- Stage that exact file.
- I authorize one local documentation commit:

  docs(storage-core): clarify CLI acknowledgements and recovery handoff

Do not amend or rewrite the existing foundation commit.


3. PUSH THE FEATURE BRANCH

After checks succeed, run:

git push -u origin feat/storage-core

If authentication fails:
- Show the useful error without secrets.
- Tell me what action is required.
- Do not change remotes or credentials automatically.
- Do not claim the push succeeded.

If push is rejected:
- Do not force.
- Inspect and explain the remote difference.
- Stop before any merge/rebase.

After success, verify:

git status -sb
git rev-parse HEAD
git ls-remote --heads origin feat/storage-core

Confirm the remote branch points to the expected current commit.


4. CREATE OR PREPARE THE PR

Base branch:
main

Head/compare branch:
feat/storage-core

Title:
feat(storage-core): add storage foundation and shared engine contract

Description must distinguish:

IMPLEMENTED AND LOCALLY TESTED
- Binary format/codec.
- Checked I/O.
- RAM key-to-offset index.
- Public engine contract.
- Current mutation/recovery foundation.
- Mac unit/integration tests.
- Vet/race evidence as actually executed.

COMPILES BUT NOT RUNTIME-VERIFIED
- Linux targets, according to actual cross-compilation evidence.

INCOMPLETE
- Compaction.
- Full-cap reducing replacement.
- Real process-kill campaigns.
- WSL2 runtime verification.
- Million-key recovery measurement.
- Final CLI/torture integration.

Native Windows runtime is unsupported.

If gh is already installed and authenticated, create the PR without merging.

Check for an existing PR first; do not create duplicates.

If GitHub tooling is unavailable, provide the browser steps and a ready-to-paste PR description. Do not install tools merely to open the PR.

Request Member 2 to review the transaction/recovery/format integration.

Keep all five team branches.


5. PRODUCE THREE CLEAN TEAM MESSAGES

Give me three separate, concise copy-paste messages with the actual pushed commit hash and PR reference.

MEMBER 2 MESSAGE

Explain:
- Read the actual shared contract and HANDOFF_MEMBER1.md.
- Existing transaction/recovery code already exists.
- Do not build a competing codec or scanner.
- Review existing commit/recovery correctness.
- Implement compaction and full-cap reducing replacement.
- Coordinate removing/moving the existing Compact stub before defining another method.
- Keep the same binary format and mutex rules.
- Review the PR before integration into main.

MEMBER 3 MESSAGE

Explain:
- Start the external harness and independent oracle now.
- Coordinate wire framing and request IDs with Member 4.
- Record submitted operations plus observed ACKs.
- Include all acknowledged operations in legal histories.
- Allow unresolved operations to be completely included or absent.
- Do not assume request order or ACK order equals commit order.
- Do not import private index/replay code.
- Target actual force-kills, not graceful shutdown.
- Preserve seed, trace, stderr, and failing files.

MEMBER 4 MESSAGE

Explain:
- Build against the actual contract, not prior chat summaries.
- Open returns the actual Engine/OpenInfo/error signature.
- Implement all eight required commands.
- get prints value or NOTFOUND.
- serve prints ACK <op-id> only after durable success.
- Handle uncertain errors without automatic mutation retry.
- Coordinate binary framing with Member 3.
- Open timing must follow documented measurement semantics.
- Primary demo runtime is Linux inside WSL2.


6. TEAM INTEGRATION POLICY

Do not tell teammates to blindly overwrite their branches.

They should:
- Preserve and commit their own work as appropriate.
- Review the foundation PR.
- After it is approved and merged, fetch integrated main and incorporate it into their assigned branches.
- Resolve conflicts deliberately.
- Avoid creating additional shared modules or competing APIs.

Before PR merge, they may inspect the pushed feature branch without abandoning their own work.

Do not merge main or any teammate branch automatically.


7. WSL2 VERIFICATION MESSAGE

Prepare this gate for the demo-laptop teammate after the code is available through the agreed integration path:

Inside Ubuntu under WSL2:
- Verify the repository lives under /home/...
- Verify uname -s reports Linux.
- Verify architecture.
- Verify actual Go version.
- Run:

  go test -count=1 -v ./src/kv
  go vet ./...
  go test -race -count=1 ./src/kv

Report actual results.

Do not call a Mac cross-compile a completed Linux runtime test.

Do not claim a complete database win-condition until crash, compaction, full-cap, and recovery-scale tests pass.


FINAL RESPONSE

Return:

1. Any narrow corrections made.
2. Actual local and remote commit hashes.
3. Push result.
4. PR URL, if actually created; otherwise browser instructions.
5. Three clean teammate messages.
6. Remaining integration gates.

Stop after sharing and handoff.
Do not merge automatically.
```

---

## Prompt 8: Member 1 Tech Lead Master Delivery & WSL2 Benchmark
**Timestamp:** `2026-11-06T10:09:43Z`  
**Transcript Step Index:** `412`  

```text
# FSYNC-OR-DIE — MEMBER 1 MASTER MISSION
## TECH LEAD + CORE STORAGE ENGINEER + INTEGRATION LEAD

You are my senior Go engineering partner, storage-systems architect, integration engineer and technical mentor.

**Team:** Team Innovexa  
**Project:** Fsync-or-Die  
**Language:** Go  
**My role:** Member 1 — Tech Lead + Core Storage Engineer + Integration Lead  
**My branch:** feat/storage-core  
**Repository:** Team Innovexa shared GitHub repository  
**Primary demo machine:** Designated Windows teammate's laptop.

## MY FINAL OBJECTIVE

Complete everything belonging to my role and prepare me to successfully lead all four members until the final hackathon demonstration.

I am responsible for:

1. Completing and verifying my core storage engine.
2. Ensuring Member 2's transactions and recovery integrate with my engine.
3. Ensuring Member 3's independent torture-testing tool correctly tests our database.
4. Ensuring Member 4's CLI works with our actual public API.
5. Coordinating integration, reviewing code, resolving conflicts and testing the complete system.
6. Preparing our final technical demonstration and evidence.

Do not just create reports. Inspect the real repository, perform authorized work, execute tests and fix genuine problems.

## PART 1 — COMPLETE MY CORE ENGINE

Inspect and verify my existing implementation:

- contract.go
- format.go
- io.go
- index.go
- capacity.go
- db.go
- get.go
- put_delete.go
- platform-specific locking files
- all existing tests
- HANDOFF_MEMBER1.md

Verify the actual code for:

- Binary encoding and decoding.
- File and transaction headers.
- Commit footer and CRC validation.
- Checked file I/O.
- RAM key-to-offset index.
- Get, Put and Delete.
- Capacity accounting.
- Database locking and lifecycle.
- Correct error handling.
- Thread safety.
- Integration boundaries with Member 2.

Fix bugs in my owned code, add missing tests and keep the API and binary format consistent.

Never claim that a feature is tested unless the test actually ran.

## PART 2 — BECOME THE TECHNICAL INTEGRATION LEAD

Establish one authoritative architecture and shared interface for all four members.

Review the actual code on all available branches:

- feat/storage-core
- feat/transactions-recovery
- feat/torture-harness
- feat/cli-docs

Keep main stable.

Do not invent missing implementations or assume other members have pushed their work.

Resolve and document:

- Public Engine API.
- Binary file format.
- Error types.
- Transaction protocol.
- Mutex ownership.
- File synchronization.
- Index publication.
- Recovery boundaries.
- Compaction protocol.
- CLI request/response contract.
- Torture harness request IDs and ACK handling.

There must be one transaction protocol, one recovery implementation and one authoritative database file.

## PART 3 — GUIDE MEMBER 2

Member 2 owns:

- Atomic transactions and Batch.
- Commit protocol.
- Recovery scanner and replay.
- Incomplete-tail repair.
- Repeated-crash-safe recovery.
- Crash-safe compaction.

Prepare a precise technical handoff for Member 2 using my actual source code.

Explain exactly which functions they should use, extend or replace.

Review their implementation for:
- Complete transaction validation.
- Atomic batch publication.
- Correct synchronization before success.
- Recovery after abrupt process termination.
- Repeated recovery crashes.
- Compaction correctness.
- Full-cap deletion and space reclamation.
- Consistent descriptor and index switching.

Do not implement a competing recovery scanner or silently overwrite their work.

Give me clear review instructions and acceptance tests so I can guide Member 2.

## PART 4 — GUIDE MEMBER 3

Member 3 owns the independent torture-testing system.

Prepare their exact testing contract.

Verify that their tool:
- Maintains an independent oracle.
- Records submitted operations and actual ACKs.
- Uses reproducible seeds.
- Starts and forcibly terminates the actual database process.
- Restarts and validates recovery.
- Tests concurrent writers.
- Tests incomplete batches and interrupted compaction.
- Saves failure traces and database files.

Never allow the oracle to reuse our private index or recovery implementation.

Prepare a review checklist and exact acceptance criteria for me.

## PART 5 — GUIDE MEMBER 4

Member 4 owns the CLI and documentation.

Prepare their exact interface contract.

Verify:
- Required commands from the official PDF.
- Actual Engine API usage.
- Correct argument parsing.
- Exact stdout protocol.
- Correct error mapping.
- Binary-safe input handling where required.
- ACK only after durable success.
- No unsafe automatic retries after uncertain errors.
- Documentation matches actual implementation.

Prepare a CLI integration checklist and final demo checklist for me.

## PART 6 — INTEGRATE EVERYONE'S CODE

Once the other members' implementations are available:

1. Inspect their actual feature branches.
2. Review changes and dependency compatibility.
3. Identify interface conflicts.
4. Prepare a safe integration plan.
5. Test combined code in an isolated workspace where practical.
6. Fix integration problems in the appropriate owner's files with their coordination.
7. Run the complete application and tests.
8. Prepare reviewed pull requests.

Do not force-push, delete branches, overwrite work or automatically merge PRs.

I lead integration, but every member retains ownership of their assigned implementation.

## PART 7 — COMPLETE SYSTEM VERIFICATION

Run real verification for:

- Codec and storage tests.
- Put/Get/Delete.
- Atomic Batch.
- Recovery and torn-tail repair.
- Repeated crashes.
- Concurrent writers.
- Crash-safe compaction.
- Disk-full scenarios.
- Independent torture testing.
- CLI protocol.
- One-million-key recovery under the official two-second target.
- Performance and memory measurements.

Run appropriate Go checks:

`go test -count=1 ./...`

`go vet ./...`

`go test -race -count=1 ./...`

Run benchmarks and preserve actual results.

Verify the real final demonstration environment. Distinguish macOS, native Windows and Linux under WSL2. Do not claim platform support without runtime evidence.

## PART 8 — PREPARE ME TO LEAD THE TEAM

This is especially important.

Give me:

1. A simple explanation of the complete database architecture.
2. Every module's responsibility.
3. How data moves through the database.
4. How transactions work.
5. How crash recovery works.
6. How compaction works.
7. How the torture tool proves reliability.
8. How the CLI communicates with the engine.
9. How the four modules integrate.
10. What questions judges might ask and technically correct answers.

Create separate copy-paste instructions for Members 2, 3 and 4 whenever a task or correction needs to be assigned.

Give me exact file paths, function names, terminal commands and expected results when instructing me.

Assume I am still learning Go. Explain important integration decisions in beginner-friendly language.

## PART 9 — FINAL DELIVERY

My role is complete only when:

- My core storage implementation is verified.
- The shared API and binary format are stable.
- Member 2's work is integrated and verified.
- Member 3's torture harness operates against the real database.
- Member 4's CLI works against the real engine.
- Cross-module tests pass.
- Crash and recovery behavior has genuine test evidence.
- Performance targets are measured.
- The final demo environment has been checked.
- I have the technical documentation and knowledge to guide the team and explain the architecture.

Prepare a final report showing:

- Completed work.
- Actual test results.
- Remaining bugs.
- Each member's integration status.
- Performance measurements.
- Supported platforms.
- Known limitations.
- Final demo instructions.
- Outstanding challenge requirements.

Use honest statuses:
IMPLEMENTED AND TESTED
IMPLEMENTED BUT UNVERIFIED
PENDING MEMBER
PENDING INTEGRATION
BLOCKED
KNOWN FAILING

Do not declare the project complete while mandatory requirements remain unverified.

## EXECUTION RULES

- Start by inspecting the actual repository.
- Continue through all safe work without repeatedly requesting approval for ordinary coding.
- Preserve existing work.
- Work only within my branch when changing my owned files.
- Coordinate before changing another member's files.
- Never fake code, tests, benchmarks or guarantees.
- Do not use banned database/storage libraries.
- Do not sacrifice correctness for speed.
- Use the official challenge PDF as the final authority.
- Do not automatically merge pull requests.
- Do not perform destructive Git operations.

**START NOW. Complete my Member 1 responsibilities, prepare me to guide every teammate, lead integration and take the project through final technical verification and demo readiness.**

My goal is not four separate implementations. My goal is ONE working, reliable database built by Team Innovexa.

FSYNC-OR-DIE — DON'T TRUST THE PROCESS. TRUST THE PROOF.
```

---

## Prompt 9: 1-Million Key Cold Recovery Benchmark Authorization
**Timestamp:** `2026-11-06T10:25:37Z`  
**Transcript Step Index:** `530`  

```text
# MEMBER 1 — FINAL VERIFICATION AND HANDOFF AUTHORIZATION

The reported progress is encouraging. Proceed with the following steps carefully.

### 1. AUDIT BEFORE COMMITTING

Show:

- git branch --show-current
- git status --short
- git diff --stat
- git diff --check
- git log -3 --oneline

Show the actual diff for:
- src/kv/db.go
- src/kv/db_test.go
- scripts/wsl2_verify.sh
- Any other modified files.

Confirm no unrelated or another member's files are included.

Run and report actual results:

- go test -count=1 ./src/kv
- go test -race -count=1 ./src/kv
- go vet ./...

Do not commit if a relevant test fails.

### 2. VERIFY THE RECOVERY BENCHMARK

The reported 50,000-key result is 13.4 ms.

The projected 0.27 seconds for one million keys is not an actual measurement.

Create and execute a real one-million-key recovery benchmark if the environment permits.

Measure the actual database Open/recovery path, not merely an isolated parsing function.

Report dataset size, database file size, Go version, OS, hardware, benchmark iterations and actual results.

If it cannot be executed, mark the million-key requirement UNVERIFIED.

### 3. VERIFY CROSS-PLATFORM RESULTS

Show the exact commands used for:
- Darwin ARM64
- Linux AMD64
- Linux ARM64
- Windows AMD64

Distinguish cross-compilation from actual runtime testing.

Run the WSL2 verification script on the actual demo machine when available. Do not claim WSL2 runtime verification until it has genuinely run.

### 4. COMMIT AUTHORIZATION

After showing the diff and all available verification results, stop and present the proposed commit.

Do not amend or rewrite the original foundation commit.

Do not commit until I explicitly approve this additional implementation commit.

### 5. PULL REQUEST

After authorization and a successful push:

- Open or update the PR from feat/storage-core into main.
- Request Member 2's review.
- Include actual verification evidence.
- Clearly list incomplete features.

DO NOT MERGE.

### 6. TEAM HANDOFF

Prepare three individual messages for Members 2, 3 and 4.

Use the actual commit hash and PR URL only after verifying them.

Tell Member 2 to inspect the actual storage contract and integrate transactions/recovery without building a competing format.

Tell Member 3 to build the independent torture tool.

Tell Member 4 to integrate the CLI with the actual Engine API.

### 7. FINAL STATUS

Clearly separate:
- Implemented and tested.
- Implemented but unverified.
- Pending Member 2.
- Pending Member 3.
- Pending Member 4.
- Pending full-system integration.

Do not claim the complete database is finished until all mandatory challenge requirements have been verified.

Start with the source diff and verification audit.
```

---

## Prompt 10: Member 1 Final Completion & Integrity Checker Mission
**Timestamp:** `2026-11-06T10:32:07Z`  
**Transcript Step Index:** `578`  

```text
# FSYNC-OR-DIE — MEMBER 1 FINAL COMPLETION MISSION
## TECH LEAD + CORE STORAGE ENGINEER + TECHNICAL VERIFICATION LEAD

You are my principal Go storage-engine engineer, database architect, quality engineer and technical mentor.

Your mission is to complete every responsibility belonging to me, Member 1, to a professional engineering standard, with actual implementation evidence, meaningful tests, measured performance and complete technical documentation.

Do not stop at writing a plan or producing another theoretical report. Inspect the actual repository, complete my work, execute tests, fix genuine problems and prepare my final handoff.

Do not promise zero bugs or mathematically perfect accuracy. Achieve the highest defensible engineering quality through implementation, verification and honest evidence.

---

# 1. PROJECT AND MY ROLE

**Team:** Team Innovexa  
**Project:** Fsync-or-Die  
**Tagline:** Don't Trust the Process. Trust the Proof.  
**Language:** Go  
**Repository:** Team Innovexa shared GitHub repository  
**My branch:** `feat/storage-core`  
**My role:** Member 1 — Tech Lead + Core Storage Engineer + Technical Verification Lead  
**Primary demonstration environment:** Designated Windows teammate's laptop, subject to confirmation of native Windows or Ubuntu under WSL2.

## MY RESPONSIBILITIES

I am responsible for:

1. Building and completing the core storage foundation.
2. Maintaining the public engine API and binary-format contract.
3. Verifying my storage implementation.
4. Testing file integrity, durability-related behavior and error handling.
5. Measuring actual large-dataset performance.
6. Verifying my owned platform-specific code.
7. Creating complete technical documentation and handoff instructions.
8. Preparing exact implementation and review instructions for Members 2, 3 and 4.
9. Understanding the entire architecture well enough to guide the team and answer judges' questions.

**IMPORTANT SCOPE BOUNDARY:**

I want you to complete all Member 1 work.

I do NOT want you to take over the implementation assigned to Members 2, 3 or 4.

Do NOT integrate their feature branches, merge their code, rewrite their modules or attempt to finish their assigned deliverables as part of this mission.

You may prepare interfaces, integration contracts, dependency documentation, review instructions and test requirements for them.

Actual cross-member integration will be performed separately after their work is ready.

---

# 2. SOURCE OF TRUTH

Follow this priority:

1. Official `Kill It and Prove It` challenge PDF.
2. Approved team architecture and shared API.
3. Actual source code and repository state.
4. Executed tests and measured results.
5. Planning documents and previous reports.

Read the complete official PDF if it is available.

Verify the exact:
- Required API behavior.
- Key and value size limits.
- Batch operation limits.
- Concurrency requirements.
- Crash scenarios.
- Durability guarantees.
- Recovery rules.
- File-size requirements.
- Compaction requirements.
- Performance targets.
- Required deliverables.
- Judging and demonstration conditions.

Never invent a challenge requirement.

If the PDF is unavailable, identify what remains unverified.

Clearly distinguish:
- Official requirement.
- Approved design.
- Proposed design improvement.
- Implemented feature.
- Executed test result.
- Unverified behavior.

---

# 3. REPOSITORY INSPECTION

Start by inspecting the actual repository.

Run:

```bash
git rev-parse --show-toplevel
git branch --show-current
git status --short
git branch --list
git log -10 --oneline
git remote -v
git ls-files
git ls-files --others --exclude-standard
```

Check whether the previously reported foundation commit exists:

`0a4f44a`

Reported message:

`feat(storage-core): add tested storage foundation and integration contract`

Do not assume the commit or push exists. Verify it.

Inspect the actual files, especially:

```text
go.mod
README.md
src/kv/contract.go
src/kv/format.go
src/kv/format_test.go
src/kv/io.go
src/kv/io_test.go
src/kv/index.go
src/kv/index_test.go
src/kv/capacity.go
src/kv/capacity_test.go
src/kv/db.go
src/kv/db_test.go
src/kv/get.go
src/kv/put_delete.go
src/kv/lock_darwin.go
src/kv/lock_linux.go
src/kv/lock_windows.go
src/kv/bench_test.go
HANDOFF_MEMBER1.md
MEMBER1_TECH_LEAD_MASTER_DELIVERY.md
scripts/wsl2_verify.sh
```

Some files may not exist. Report actual observations.

Preserve all uncommitted work.

Never:
- Reset the branch.
- Force-push.
- Delete branches.
- Switch to another member's branch.
- Overwrite another member's code.
- Modify shared documents owned by Member 4 without coordination.
- Print secrets or credentials.

Work only on `feat/storage-core` and files within my ownership.

---

# 4. COMPLETE MY CORE STORAGE ENGINE

Audit every existing implementation rather than trusting earlier completion reports.

## A. PUBLIC CONTRACT

Verify:

- `Engine` interface.
- `Open` function signature.
- `Options`.
- `OpenInfo`.
- `Stats`.
- `Kind`.
- `Op`.
- Error definitions.
- `errors.Is` compatibility.
- API documentation.

Preserve the established API unless an actual bug or official requirement requires a change.

Document the precise behavior of:
- Put.
- Get.
- Delete.
- Batch.
- Close.
- Check.
- Scan.
- Stats.
- Open.

Do not introduce competing names or alternative API definitions.

## B. BINARY FORMAT

Verify the actual file format byte by byte.

The reported design contains:

**32-byte file header**

- Magic: `KIPDB001`
- Format version.
- Header size.
- Maximum file size.
- Flags.
- CRC32-IEEE.

**32-byte transaction header**

- Magic: `TXN1`
- Transaction version.
- Flags.
- Transaction ID.
- Operation count.
- Reserved field.
- Payload length.

**Operation format**

```text
[kind:1][key length:2][value length:4][key bytes][value bytes]
```

**32-byte commit footer**

- Magic: `COMMIT01`
- Matching transaction ID.
- Total transaction length.
- CRC32-IEEE over transaction header and payload.
- CRC32-IEEE over footer bytes 0–27.

Verify:
- Exact field offsets.
- Exact byte widths.
- Little-endian encoding.
- CRC coverage.
- Transaction length calculations.
- Integer-overflow protection.
- Supported version validation.
- Unknown flag rejection.
- Invalid operation rejection.
- Empty-value handling.
- Maximum key/value boundaries.

Use known expected byte vectors, not just round-trip tests.

Do not silently modify the shared format.

Do not introduce a second storage format.

## C. CHECKED FILE I/O

Verify:

- Exact positioned reads.
- Complete positioned writes.
- Partial-progress handling.
- Zero-progress detection.
- Unexpected EOF.
- Invalid offsets.
- Offset overflow.
- Synchronization error propagation.
- Truncation error propagation.
- Close error handling.

Every relevant I/O failure must be checked and classified.

Never disguise an uncertain I/O result as a definite rejection.

## D. RAM INDEX

Verify that the index stores key-to-offset/length information rather than unnecessarily keeping every value in memory.

Test:

- Binary keys.
- Empty values.
- Put.
- Overwrite.
- Delete.
- Duplicate-key batch updates.
- Ordered application.
- Snapshot isolation.
- Caller-owned returned values.
- Large key counts.
- Concurrent access.

Measure actual memory consumption where practical.

Do not claim one million keys are verified simply because the data structure is designed to support them.

## E. DATABASE LIFECYCLE

Verify:

- File creation.
- Existing database opening.
- Header validation.
- Lock acquisition and release.
- Open failures.
- Close behavior.
- Closed-handle rejection.
- Poisoned-handle rejection.
- Recovery hook behavior.
- File descriptor correctness.
- Resource cleanup.

Never use truncating flags when opening an existing database.

## F. CAPACITY MANAGEMENT

Verify:

- Exact encoded-size accounting.
- Header and footer overhead.
- Checked integer arithmetic.
- Unlimited configuration.
- Exact-fit boundary.
- Over-limit rejection.
- Existing data preservation.
- Persisted maximum-size configuration.
- Delete-at-capacity behavior.
- Compaction dependencies.

A tombstone consumes space. Never pretend ordinary append-only deletion works when the configured file is completely full.

Document Member 2's reducing-compaction dependency without implementing a competing compaction algorithm.

---

# 5. COMPLETE AND VERIFY MY FULL-FILE INTEGRITY CHECKER

The previously reported `db.Check` traverses transactions from offset 32 to `committedEnd`.

Audit and verify that implementation.

It must validate:

1. File header.
2. Transaction header.
3. Version and flags.
4. Operation count.
5. Operation kinds and lengths.
6. Key/value boundaries.
7. Exact payload consumption.
8. Commit footer.
9. Matching transaction IDs.
10. Total transaction size.
11. Transaction CRC.
12. Footer CRC.
13. Transaction boundaries.
14. Final committed offset.

Add meaningful tests for:
- Valid complete database.
- Invalid file header.
- Corrupt transaction header.
- Corrupt operation.
- Corrupt payload.
- Corrupt commit footer.
- Wrong transaction ID.
- Incorrect transaction length.
- CRC mismatch.
- Incomplete tail.
- Unexpected file truncation.
- Invalid committed-end metadata.

**Important:** Check must not silently repair or mutate the database unless the approved API explicitly defines that behavior.

Keep integrity checking separate from recovery repair.

Do not create another competing recovery scanner.

---

# 6. DURABILITY AND CORRECTNESS AUDIT

Verify the common intended mutation protocol against the approved Member 2 transaction contract:

1. Acquire DB mutex.
2. Validate the complete request.
3. Calculate encoded size.
4. Check configured capacity.
5. Record append start.
6. Write transaction header.
7. Write all operations.
8. Write complete commit footer.
9. Perform required final `File.Sync()`.
10. Publish complete index changes.
11. Advance committed boundary.
12. Release lock.
13. Return success.
14. Permit CLI acknowledgement.

The actual production transaction writer remains Member 2's responsibility.

Do not implement a duplicate transaction writer.

Verify these invariants:

- No success before required synchronization succeeds.
- No partial batch becomes visible.
- No partial index publication.
- No replay before complete transaction validation.
- No append behind an unresolved invalid tail.
- No ACK before durable success.
- Returned values are caller-owned.
- Reads cannot race with file replacement.
- Closing and poisoning behavior is explicit.

**CRASH SEMANTICS:**

A transaction without an observed acknowledgement may be fully included or fully absent after recovery.

A process killed during Sync does not automatically mean the transaction is absent.

A valid completed transaction can be recovered even if the ACK was never observed.

An atomic batch must never be partially applied.

Do not claim power-loss safety merely because process-kill behavior is tested.

---

# 7. PLATFORM AND LOCKING VERIFICATION

Our team has two macOS computers and two Windows computers.

The primary demonstration environment is the designated Windows teammate's machine.

Inspect the current implementations:

- `lock_darwin.go`
- `lock_linux.go`
- `lock_windows.go`

Verify:
- Correct build tags.
- Exclusive process ownership.
- Lock acquisition.
- Lock contention.
- Lock release.
- Process termination behavior.
- File descriptor cleanup.
- Unsupported-platform behavior.

A Go mutex protects one process's goroutines. An OS-level file lock is needed to coordinate separate processes.

Do not claim cross-process concurrency is supported merely because both mechanisms exist.

Confirm whether the final demo uses:
- Native Windows, or
- Ubuntu under WSL2.

Do not assume WSL2 is installed.

For WSL2, confirm:
- Linux kernel.
- Architecture.
- Go toolchain.
- Repository location under `/home/...`.
- Filesystem behavior.
- Locking behavior.
- Runtime test results.

Verify actual runtime behavior where possible.

Distinguish:
- Native macOS runtime.
- Native Windows runtime.
- Linux runtime.
- Linux under WSL2.
- Cross-compilation only.

Never describe cross-compilation as runtime testing.

---

# 8. TOOLCHAIN AND BUILD VERIFICATION

Inspect:

```bash
go version
go env GOOS GOARCH GOMOD GOTOOLCHAIN GOPATH
```

Preserve the agreed Go version and module path.

If Go is missing:
- Detect OS and architecture.
- Use the official source https://go.dev/dl/.
- Verify the published checksum.
- Prefer user-directory installation where possible.
- Ask before any privileged installation.
- Do not overwrite a working installation.
- Verify the installed version.

Do not silently change `go.mod` or trigger an unexpected toolchain download.

## Execute actual checks

Run:

```bash
gofmt -w src/kv/*.go
go test -count=1 -v ./src/kv
go vet ./...
go test -race -count=1 ./src/kv
go test -run '^$' -bench . -benchmem ./src/kv
go test -count=1 ./...
```

Run commands supported by the actual current repository.

Do not claim a command passed unless it actually executed successfully.

If a test fails:
1. Show the relevant error.
2. Identify the responsible function.
3. Diagnose the cause.
4. Fix the smallest appropriate change in my owned code.
5. Rerun the test.
6. Record the final result.

Never delete meaningful tests or weaken assertions just to obtain green output.

---

# 9. EMPIRICAL PERFORMANCE — ONE MILLION KEYS

This is one of my most important deliverables.

The previous report stated:

- 50,000 keys.
- 13.4 ms measured.
- Approximately 0.27 seconds extrapolated to one million keys.

Treat only the 50,000-key result as previously reported evidence.

The 0.27-second result is an extrapolation, not a measured million-key result.

I need an actual benchmark of the real production Open/recovery path.

## Benchmark requirements

1. Inspect the official benchmark requirements in the PDF.
2. Build a dataset with at least one million distinct keys.
3. Use the actual database file format.
4. Populate the database through the real implementation or a valid production-compatible preparation method.
5. Close it correctly.
6. Measure the actual database reopening and recovery operation.
7. Include transaction validation, CRC verification and RAM index reconstruction.
8. Separate dataset preparation from recovery time.
9. Record the actual database file size.
10. Record OS, architecture, CPU, RAM and Go version.
11. Execute multiple measured runs.
12. Report median and range, not only the best run.
13. Record peak memory where practical.
14. Preserve exact reproduction commands.

If the official judging workload specifies particular key/value sizes or file layout, reproduce those conditions.

If the full recovery implementation is still owned by Member 2 and is not available in my local branch, do not build a competing recovery scanner simply to produce a benchmark.

In that case:
- Benchmark only the actual available production path.
- Clearly label it preliminary.
- State that the official one-million-key recovery result remains pending Member 2.
- Prepare a reproducible benchmark contract for final integration.

**Acceptance target previously identified: opening/recovery with one million keys in under two seconds. Verify the exact PDF wording.**

Do not report 0.27 seconds as an achieved result unless the actual million-key test measures it.

---

# 10. MEMBER 1 TEST COMPLETENESS

Build a meaningful test suite for all my owned code.

## Codec

- Known byte vectors.
- Endianness.
- Round trips.
- Empty values.
- Binary data.
- Maximum key.
- Maximum value.
- Invalid kinds.
- Unsupported versions.
- Invalid flags.
- Incorrect lengths.
- CRC corruption.
- Integer overflow.
- Truncated records.
- Invalid footer.
- Transaction ID mismatch.

## File I/O

- Full writes.
- Partial writes.
- Zero progress.
- Read errors.
- Write errors.
- Unexpected EOF.
- Sync errors.
- Truncate errors.
- Close errors.
- Invalid offsets.

## Index

- Insert.
- Overwrite.
- Delete.
- Missing key.
- Binary keys.
- Empty values.
- Ordered pending updates.
- Snapshot behavior.
- Caller-owned result.
- Concurrent access.

## Capacity

- Unlimited.
- Exact fit.
- One byte too large.
- Negative offsets.
- Arithmetic overflow.
- Header/footer overhead.
- Failed preflight preserves old data.

## Database

- Open.
- Close.
- Reopen.
- Put/Get.
- Overwrite.
- Delete.
- Empty values.
- Missing key.
- Lock contention.
- Concurrent goroutines.
- Closed state.
- Poisoned state.
- Check integrity.
- I/O errors.

Use table-driven tests and fuzz testing where appropriate.

Coverage percentage is supplemental evidence, not proof of correctness.

Do not label clean-close/reopen tests as crash tests.

---

# 11. CROSS-PLATFORM BUILD EVIDENCE

Where supported, verify compilation for:

```text
darwin/arm64
linux/amd64
linux/arm64
windows/amd64
```

Use appropriate build or test compilation commands.

Also run actual tests on available runtime environments.

For example, a successful cross-compilation does not prove that Windows file locking or atomic replacement works at runtime.

Prepare `scripts/wsl2_verify.sh` for the designated Windows teammate.

The script must:
- Check that it is running under Linux.
- Check architecture.
- Check Go version.
- Check repository location.
- Reject running from `/mnt/c` if the chosen test policy requires the Linux filesystem.
- Run the appropriate tests.
- Run race tests where supported.
- Report actual exit codes.
- Never delete repository data.
- Never overwrite user files.
- Never claim success when a command failed.

Test the script in an available WSL2 environment if possible.

If WSL2 is unavailable, mark actual WSL2 runtime verification pending.

---

# 12. COMPLETE MY OWN DOCUMENTATION

Create or update only documents owned by Member 1.

My main documentation deliverables:

`HANDOFF_MEMBER1.md`

`MEMBER1_TECH_LEAD_MASTER_DELIVERY.md`

They must explain:

## A. Architecture
- Append-only log.
- Binary format.
- RAM index.
- Database lifecycle.
- Capacity enforcement.
- Durability sequence.
- Error behavior.

## B. Source-code map
Every owned file and its purpose.

## C. API contract
Actual exported functions, types and errors.

## D. Internal contracts
Actual function names and ownership boundaries for Member 2.

## E. Testing evidence
Exact commands, actual output summaries, environment and timestamps.

## F. Performance evidence
50,000-key results and the actual million-key result if measured.

## G. Platform support
Separate runtime-verified and compile-only platforms.

## H. Known limitations
Explicitly list compaction, full-cap replacement and any other Member 2 dependencies.

## I. Member handoffs
Prepare three distinct, ready-to-send technical instruction messages.

**Member 2 handoff:**
- Actual codec APIs.
- Transaction format.
- CRC coverage.
- Common transaction writer interface.
- Mutex ownership.
- Index publication.
- Recovery and compaction integration.
- Full-cap behavior.
- Acceptance tests.

**Member 3 handoff:**
- Actual public engine/executable interface.
- Request and ACK semantics.
- Independent oracle rules.
- Required forced-crash scenarios.
- Reproducible seed and artifact requirements.
- No private index/replay dependencies.

**Member 4 handoff:**
- Actual Engine interface.
- Open signature.
- CLI semantics.
- Error mappings.
- ACK protocol.
- Check and scan behavior.
- Build and run instructions.

Never invent APIs or commands that are not present in the source.

Do not overwrite Member 4's README or architecture documents.

---

# 13. TECH LEAD PREPARATION — MAKE ME READY TO GUIDE THE TEAM

Prepare a beginner-friendly but technically accurate explanation of the entire architecture.

I must be able to explain:

1. Why we are building this database.
2. What problem it solves.
3. What makes append-only storage useful.
4. How the binary format works.
5. How Put works.
6. How Get works.
7. How Delete works.
8. How atomic Batch works.
9. What fsync/File.Sync means.
10. What a checksum is.
11. What the RAM index does.
12. How recovery works.
13. How compaction works.
14. How the configured file-size limit works.
15. How process locks differ from goroutine mutexes.
16. What our torture-testing tool proves.
17. What our current performance numbers actually demonstrate.
18. What is tested and what is still unverified.

Prepare:

- A 30-second judge explanation.
- A 2-minute project explanation.
- A 5-minute technical architecture explanation.
- A list of likely judge questions and precise answers.
- A list of misleading claims I must avoid.
- A simple explanation of my personal contribution.
- A simple explanation of Member 2's responsibility.

Give me technically accurate explanations in language a beginner can comfortably present.

---

# 14. GIT AND CHANGE MANAGEMENT

Remain on:

`feat/storage-core`

I authorize scoped edits to my owned source code and documentation.

Preserve the original foundation commit.

Use meaningful additional commits only after checking diffs and all relevant tests.

Do not:
- Amend the original commit.
- Force-push.
- Delete branches.
- Merge another member's branch.
- Merge into main.
- Rewrite another member's code.
- Perform destructive cleanup.

If the branch is pushed, verify the actual remote commit hash.

A PR may be prepared for peer review if the existing GitHub tooling is authenticated, but do not merge it.

If authentication is unavailable, provide browser instructions.

Show `git diff --check` and the final changed-file list.

---

# 15. EXPLICITLY EXCLUDED FROM THIS MISSION

Do not perform cross-member implementation integration.

Do not:
- Merge `feat/transactions-recovery`.
- Merge `feat/torture-harness`.
- Merge `feat/cli-docs`.
- Build Member 2's production recovery or compaction implementation.
- Build Member 3's torture executable.
- Build Member 4's CLI.
- Replace other members' code with your own.
- Claim complete four-member integration.

You may inspect available documents and prepare technical contracts and handoffs. Actual integration is a separate future task.

---

# 16. COMPLETION CONDITIONS

My Member 1 role is complete only when all of the following have an explicit status:

**CORE ENGINE**
- Public contract complete.
- Binary codec verified.
- Checked I/O verified.
- RAM index verified.
- Get/Put/Delete foundation verified.
- Capacity handling verified.
- Database lifecycle verified.
- Full-file integrity checker verified.

**TESTING**
- Core tests pass.
- Race tests pass where supported.
- Vet passes.
- Error-path tests are meaningful.
- Benchmarks actually execute.

**PERFORMANCE**
- Existing 50,000-key result independently reproduced or marked unverified.
- Actual one-million-key benchmark executed if the complete production recovery path is available.
- No extrapolation presented as measurement.

**PLATFORMS**
- macOS runtime status recorded.
- Linux runtime status recorded.
- WSL2 status recorded.
- Windows compilation/runtime status separately recorded.

**DOCUMENTATION**
- HANDOFF_MEMBER1.md complete.
- MEMBER1_TECH_LEAD_MASTER_DELIVERY.md complete.
- All three member handoff messages ready.
- Technical explanation and judge preparation complete.

**GIT**
- Branch and working-tree state verified.
- Actual commits identified.
- Changes preserved.
- No unauthorized merge.

Use these statuses:

- IMPLEMENTED AND TESTED
- IMPLEMENTED BUT UNVERIFIED
- VERIFIED ON MACOS
- VERIFIED ON LINUX
- VERIFIED UNDER WSL2
- CROSS-COMPILED ONLY
- PENDING MEMBER 2
- PENDING MEMBER 3
- PENDING MEMBER 4
- PENDING FUTURE INTEGRATION
- BLOCKED
- KNOWN FAILING

Do not mark the complete four-member database as finished. This mission finishes my own role only.

---

# 17. EXECUTE NOW

Start by inspecting the real repository and actual code.

Then work through every Member 1 responsibility in this prompt.

Do not give me another large planning-only report.

Implement missing features in my ownership, fix real defects, execute tests, measure performance, verify the environment, update documentation and prepare the three technical handoffs.

Do not repeatedly ask for approval for ordinary safe code changes.

Stop only for genuine blockers, privileged system actions, unresolved official requirements or dangerous operations.

At the end, provide one complete Member 1 final delivery report containing:

1. Files inspected.
2. Files created or changed.
3. Actual implementation details.
4. Tests executed and results.
5. Benchmark measurements.
6. Million-key verification status.
7. Platform verification.
8. WSL2 script status.
9. Known limitations.
10. Member 2, 3 and 4 handoff messages.
11. Judge preparation material.
12. Git and branch status.
13. Completed and incomplete checklist.

**FINAL OBJECTIVE: Finish every responsibility belonging to Member 1 to the highest verifiable engineering standard, and leave me fully prepared to guide the other three members. Do not perform the actual cross-member integration in this mission.**

FSYNC-OR-DIE  
TEAM INNOVEXA  
DON'T TRUST THE PROCESS. TRUST THE PROOF.
```

---

## Prompt 11: Member 2 Completion & Torture Delivery Report
**Timestamp:** `2026-11-06T13:08:16Z`  
**Transcript Step Index:** `652`  

```text
FSYNC-OR-DIE — MEMBER 2 MISSION COMPLETE REPORT

Role: Member 2 — Transactions, Recovery, Compaction, Reliability & Torture-Testing Engineer
Team: Team Innovexa
Repository: https://github.com/madhuharshag/Team-Innovexa
Branch: feat/transactions-recovery
Final Commit: e32ea75
Host Target: Native Windows 11 (AMD64 host, windows/386 Go toolchain: go1.27.3)
Status: MISSION COMPLETE & PUSHED TO REMOTE

1. Executive Summary & Verification Evidence

All 11 phases of Member 2’s assignment and the absorbed reliability/torture-testing responsibilities are fully implemented, independently tested, and committed to feat/transactions-recovery.

text
========================================================================================
TASK / REQUIREMENT                             STATUS
========================================================================================
Phase 0: Repository & Environment Audit        IMPLEMENTED AND TESTED (Native Windows 32-bit Go)
Phase 1: Fix F4 One-Byte Truncation Test       IMPLEMENTED AND TESTED (20/20 PASS)
Phase 2: Bound Compaction Memory (1000/16MiB)  IMPLEMENTED AND TESTED (All 10 tests PASS)
Phase 3: Compaction Crash-Safety Audit         IMPLEMENTED AND TESTED (Subprocess + Sharing tests)
Phase 4: Process-Lock Continuity               IMPLEMENTED AND TESTED (LockFileEx verified)
Phase 5: Recovery & Compaction Subproc Tests   IMPLEMENTED AND TESTED (55/55 PASS)
Phase 6: Independent Torture Harness           IMPLEMENTED AND TESTED (12/12 PASS + CLI verified)
Phase 7: 1,000,000-Key Recovery Benchmark      IMPLEMENTED AND TESTED (0.462s < 2.0s goal)
Phase 8: Full Verification Suite               IMPLEMENTED AND TESTED (vet, fmt, tests clean)
Phase 9: Comprehensive Handoff Documentation   IMPLEMENTED AND TESTED (HANDOFF_MEMBER2.md)
Phase 10: Final Technical Evidence Report      IMPLEMENTED AND TESTED (MEMBER2_FINAL_TECHNICAL...)
Phase 11: Git Commit & Remote Delivery         DELIVERED & PUSHED (Commit e32ea75 to origin)
========================================================================================
2. Phase 0 — Repository and Environment Audit
Exact Recorded Terminal Outputs:
powershell
PS D:\test\transaction and recovery engine> git status
On branch feat/transactions-recovery
Your branch is up to date with 'origin/feat/transactions-recovery'.
nothing to commit, working tree clean
PS D:\test\transaction and recovery engine> git branch --show-current
feat/transactions-recovery
PS D:\test\transaction and recovery engine> git log -2 --oneline
e32ea75 feat(recovery,compaction,torture): complete recovery repair, dual-bounded compaction, 1M benchmark, and independent torture harness
3057b66 feat(recovery): implement crash-safe compaction, recovery tests, and Windows process locking
PS D:\test\transaction and recovery engine> git remote -v
origin  https://github.com/madhuharshag/Team-Innovexa.git (fetch)
origin  https://github.com/madhuharshag/Team-Innovexa.git (push)
PS D:\test\transaction and recovery engine> & "C:\Users\sanke\Downloads\go\bin\go.exe" version
go version go1.27.3 windows/386
PS D:\test\transaction and recovery engine> & "C:\Users\sanke\Downloads\go\bin\go.exe" env GOOS GOARCH GOMOD
windows
386
D:\test\transaction and recovery engine\go.mod
Critical Environment Reality:
Host System: Windows 11 AMD64 (64-bit kernel, 24 logical CPU cores).
Active Go Toolchain: windows/386 (32-bit x86 execution).
Implication: Pointers and slice headers are 4 bytes. Virtual address space is capped at ~2-3 GB. Memory bounding in compaction is critical to prevent 32-bit out-of-memory errors. Concurrency race detection (go test -race) is not supported by the Go runtime on 32-bit platforms.
3. Phase 1 — F4 One-Byte Truncation Correction
Flaw in Previous Test

The old test called Put("first", "v1") and Put("second", "v2"). Both calls succeeded and were acknowledged. Truncating one byte from the end and asserting that "second" could be discarded violated the durability contract: an acknowledged write must never be lost.

Corrected Test (TestRecovery_OneByteTruncation_F4 in src/kv/recovery_test.go)
Calls public Put for "first" -> "value1" and "second_ack" -> "value2". Both return success.
Closes database and records the authoritative file size (
𝑆
ack
S
ack
	

).
Constructs an unacknowledged trailing transaction (TxID 3) containing "unack_in_flight_key" and writes it directly to the file without acknowledging it to any caller.
Truncates exactly 1 byte from that trailing unacknowledged transaction.
Reopens the database:
Both acknowledged keys ("first" and "second_ack") survive with exact values.
"unack_in_flight_key" returns ErrNotFound.
Zero partial or invented values exist in the index (info.LiveKeyCount == 2).
info.RepairedTailBytes matches the exact size of the truncated unacknowledged transaction.
Reopened database accepts subsequent writes cleanly.
powershell
PS D:\test\transaction and recovery engine> & "C:\Users\sanke\Downloads\go\bin\go.exe" test -v -run TestRecovery_OneByteTruncation_F4 -count=20 ./src/kv
# Result: 20/20 PASS in 0.688s (Deterministic)
4. Phase 2 — Compaction Dual Limits & Memory Bounding

In src/kv/compaction.go:

Operation Count Bound: DefaultCompactionBatchOps = 1000 operations per transaction.
Encoded Payload Bound: DefaultCompactionBatchBytes = 16 * 1024 * 1024 (16 MiB).
Preflight Check: Calculates opBytes := 7 + len(k) + len(v) before reading or allocating value bytes:
If opBytes > 16 MiB, compaction fails immediately with ErrInvalid, leaving the original database intact.
Flushes the batch before adding an operation that would push it over 1,000 ops or 16 MiB.
Guarantees that legal single maximum values (up to 1 MB) are admitted into an empty batch.
Unit Tests Added in src/kv/compaction_test.go:
TestCompaction_ByteBoundarySplitting: Forces splits across transactions at 300-byte limits.
TestCompaction_SingleMaxSizedValue: Successfully compacts 1 MB values.
TestCompaction_MultipleLargeValues: Reclaims deleted 500 KB values.
TestCompaction_OperationIndividuallyTooLarge: Verifies rejection of oversized operations with ErrInvalid.
TestCompaction_ManySmallValuesOpCountLimit: Splits 500 small keys across 50-op transaction chunks.
5. Phases 3, 4 & 5 — Process-Lock Continuity & Subprocess Tests

All 9 deterministic subprocess tests in src/kv/subprocess_test.go passed natively:

Test	Assertion Verified	Native Result
TestSubprocess_1_SecondProcessLockExclusion	Second process Open returns ErrBusy.	PASS (0.02s)
TestSubprocess_2_LockContinuityDuringCompaction	<path>.lock held continuously while data file descriptors are swapped; second process receives ErrBusy.	PASS (0.04s)
TestSubprocess_3_KillOwnerReleasesLock	Force-killing owner process (Process.Kill()) releases Win32 LockFileEx byte locks; restart recovers DB.	PASS (0.01s)
TestSubprocess_4_KillCompactionWhileWritingTmp	Kill while writing .compact.tmp leaves original file intact; all 30 keys survive.	PASS (0.02s)
TestSubprocess_5_KillCompactionAfterSyncBeforeRename	Kill after SyncFile(tmpFile) before os.Rename leaves original file authoritative; tmp file deleted on restart.	PASS (0.02s)
TestSubprocess_6_KillCompactionAfterRenameBeforeReopen	Kill after rename swaps files recovers new compacted file with all 30 live keys.	PASS (0.04s)
TestSubprocess_7_RenameFailureSafety	Injected rename error does not report success; reopens original file without data loss.	PASS (0.01s)
TestSubprocess_7B_NativeWindowsSharingViolation	Real Windows sharing violation (holding read handle without FILE_SHARE_DELETE triggers Access is denied); engine reopens original file cleanly.	PASS (0.00s)
TestSubprocess_7C_SimulatedReopenFailurePoisonsHandle	Double failure marks db.poisoned, returning ErrPoisoned.	PASS (0.00s)
6. Phase 6 — Independent Torture-Testing Harness (torture/)

Implemented as an independent standalone Go package under torture/:

Independent ACK Oracle: Tracks confirmed mutations separately from database internals; writes an append-only JSON journal with synchronous Sync() before each crash point.
Ambiguous In-Flight Handling: Interrupted operations without ACK are marked INDETERMINATE. Batches enforce All-or-Nothing atomicity (zero partial batch leakage).
Out-of-Process Worker: Database engine runs in an isolated child subprocess (--worker).
Reproducibility: Every run prints its random seed and configuration for immediate deterministic replay.
Acceptance Test Results (go test -v -count=1 ./torture):

All 15 acceptance criteria verified (12 test functions):

text
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
ok      github.com/madhuharshag/Team-Innovexa/torture    1.084s
CLI Execution Example:
powershell
PS D:\test\transaction and recovery engine> & "C:\Users\sanke\Downloads\go\bin\go.exe" run ./torture -seed 99999 -ops 1000 -crashes 10 -writers 4
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
7. Phase 7 — 1,000,000-Key Cold Recovery Benchmark

Benchmark test TestRecovery_1MillionKeys_ColdOpen populated 1,000,000 unique keys across 100 sequential transactions, closed the database, and measured a fresh cold Open() recovery:

text
================================================================
 1,000,000-KEY COLD RECOVERY BENCHMARK RESULTS
================================================================
Environment:          windows / 386 (Go go1.27.3)
CPUs Available:       24 logical cores
Database File Size:   25.76 MB (27,006,432 bytes)
Recovered Txns:       100 transactions
Recovered Live Keys:  1,000,000 keys
Recovery Duration:    461.536 ms (0.462 seconds)
Memory Allocated:     88.45 MB (92,742,272 bytes)
Challenge Threshold:  < 2.0 seconds
Meets < 2.0s Goal:    TRUE (0.462s vs 2.000s — > 4x faster)
================================================================
8. Phases 8, 9, 10 & 11 — Full Verification, Documentation & Git Push
Full Test Execution (go test -count=1 ./...):
text
ok   github.com/madhuharshag/Team-Innovexa/src/kv   14.882s
ok   github.com/madhuharshag/Team-Innovexa/torture  1.453s
Static Analysis & Formatting:
go fmt ./...: Clean across all packages.
go vet ./...: 0 warnings, 0 errors.
go test -race: Honestly disclosed as unsupported by Go runtime on 32-bit windows/386.
Documentation:
HANDOFF_MEMBER2.md: Complete architecture and Member 1 review instructions.
MEMBER2_FINAL_TECHNICAL_EVIDENCE_REPORT.md: Complete technical report.
Git Delivery:
Changes committed to feat/transactions-recovery as commit e32ea75.
Pushed successfully to origin/feat/transactions-recovery:
text
To https://github.com/madhuharshag/Team-Innovexa.git
   e68e909..e32ea75  feat/transactions-recovery -> feat/transactions-recovery
Working tree is clean and up to date with remote.
9. Open Architectural Item for Member 1
Scenario E5 (Tombstone at 100% Capacity):
Appending a DeleteOp tombstone requires 
≥
72
≥72 bytes. If the database reaches 100% capacity (committedEnd >= maxFileBytes), CheckCapacity returns ErrFull. Compaction reclaims space only if deleted keys exist.
Recommendation for Member 1: Approve a reserved tombstone headroom (e.g. 64 KB) or allow tombstones when the file cap is saturated. No unauthorized changes were made to Member 1's capacity policy.
complete report
```

---

## Prompt 12: Tech Lead Authorized Final Integration Reconciliation
**Timestamp:** `2026-11-07T02:28:18Z`  
**Transcript Step Index:** `704`  

```text
# TECH LEAD AUTHORIZED — FINAL INTEGRATION RECONCILIATION

The Tech Lead has completed a technical audit of the storage-core and transactions-recovery branches.

Use the following exact integration facts as authoritative guidance.

## CONFIRMED TEAM BASELINES

Member 1:
Branch: `feat/storage-core`
Reviewed commit: `8cba4df`

Member 2:
Branch: `feat/transactions-recovery`
Reviewed commit: `e32ea75`

Member 4:
Branch: `feat/cli-docs`
Latest reported commit: `8f3634c`

Member 3:
Branch: `feat/torture-harness`
Inspect the latest remote commit and identify unique work.

Also inspect:
`test/adversarial-hardening`

Do not assume these hashes are still the latest remote branch tips. Fetch and verify them.

## 1. EXACT STORAGE RECONCILIATION

The Tech Lead has identified the following compatibility facts.

The following files were reported identical between Member 1 and Member 2:

- `contract.go`
- `format.go`
- `io.go`
- `index.go`
- `capacity.go`
- `get.go`

Preserve their compatible common implementations.

### put_delete.go

Preserve Member 1's `txCount++` tracking.

Inspect the surrounding commit and synchronization sequence to ensure the counter changes only at the correct successful transaction point.

Do not accidentally remove any other valid Member 2 changes.

### db.go

Combine both implementations:

1. Preserve Member 1's enhanced `db.Check()` functionality, including OpCount validation and overflow guards.
2. Preserve Member 2's `db.Compact()` functionality, wired to `db.compactLocked()`.

Do not choose one whole version of db.go and discard the other.

Inspect both versions and perform a deliberate function-level merge.

Add or retain tests covering both Check and Compact behavior.

### lock_windows.go

Use Member 2's real Win32 implementation:

- LockFileEx.
- UnlockFileEx.
- Correct process lock behavior.
- Correct error handling.

Do not leave Member 1's previous Windows ErrConfig implementation in the final project if Member 2's native implementation is compatible.

### compaction.go

Adopt Member 2's complete implementation.

Preserve:
- Bounded operation count.
- Bounded encoded payload size.
- Temporary replacement file handling.
- File synchronization.
- Atomic replacement.
- Descriptor switching.
- In-memory index replacement.
- Failure rollback.
- Windows rename behavior.

Do not independently rewrite compaction.

## 2. TORTURE HARNESS RECONCILIATION

Member 2's branch contains a substantial independent torture harness under `torture/`.

The Tech Lead reports that this harness:
- Starts the engine as an out-of-process child.
- Maintains an independent persistent ACK oracle.
- Performs forced process termination.
- Reopens and checks recovered data.
- Verifies all-or-nothing batch behavior.
- Includes Windows process termination support.

Integrate this implementation.

However, compare `origin/feat/torture-harness` before deciding that Member 3's branch is fully redundant.

Preserve unique improvements from Member 3, particularly:
- Workspace preservation.
- State-aware delta oracle.
- Concurrent writer support.
- Large-batch tests.
- Reproducible seeds.
- Additional subprocess and restart tests.

Do not replace newer torture code with an older version.

Inspect the actual final torture package and retain the strongest compatible implementation and tests from both contributions.

## 3. MEMBER 4 — USE THE LATEST COMPLETED CLI

Member 4's role is already complete.

Integrate the latest remote `feat/cli-docs` commit:

`8f3634c`

Do not follow outdated instructions claiming that Member 4 still needs to build the CLI.

Preserve:
- `cmd/kip/`
- `docs/CLI_USAGE.md`
- `docs/DEMO.md`
- README updates.
- `HANDOFF_MEMBER4.md`
- `MEMBER4_FINAL_SAFETY_VERIFICATION.md`

Confirm that the CLI still works with the actual merged Engine API.

Retain the agreed database-path command syntax.

Do not change CLI command syntax unnecessarily.

Verify exact `serve` compatibility against the final integrated torture harness.

## 4. VERIFICATION ENVIRONMENT RECONCILIATION

Keep Member 2 and Member 4 test evidence separate.

Member 2's earlier report describes a Windows environment with a Windows 386 Go target.

Member 4's latest report describes:
- Windows 11 AMD64.
- Go 1.23.6.
- windows/amd64.

Do not combine these into one test result.

Identify the exact commits tested by each person.

Determine whether the final integrated engine and torture source match Member 2's tested source.

Determine whether the final integrated CLI source matches Member 4's tested source.

Do not claim that either member tested a commit they did not test.

## 5. FINAL INTEGRATION PROCEDURE

Create:

`integration/final-demo`

Preserve remote branch tips and use normal Git merges.

Recommended integration order:

1. Member 1's storage foundation.
2. Member 2's recovery, compaction, locking and torture implementation.
3. Member 3's unique reliability improvements, if not already included.
4. Compatible adversarial-hardening improvements.
5. Member 4's latest CLI commit `8f3634c`.

Use actual commit ancestry to avoid duplicate merges.

Resolve ordinary conflicts carefully.

Do not modify shared contracts or storage formats without evidence and Tech Lead approval.

If a conflict affects durability, atomicity, recovery or compaction safety, investigate before resolving it.

## 6. ESSENTIAL FINAL CHECKS

Run:

`gofmt -w cmd/kip/`

`go build ./...`

`go vet ./...`

`go test -count=1 ./cmd/kip/...`

`go test -count=1 ./src/kv`

`go test -count=1 ./torture`

Run the combined suite where practical:

`go test -count=1 ./...`

Run a CLI smoke test using an isolated temporary database.

Do not use an existing demo or teammate database for destructive tests.

Preserve all test failure workspaces and logs.

Do not repeat Member 2's lengthy torture campaign if the final relevant source tree is identical to the verified source. If it differs, identify the affected files and perform targeted checks.

## 7. MERGE AND RELEASE

Push:

`integration/final-demo`

After reviewing the successful integration, merge it into `main` normally.

If branch protection blocks the push, leave the integration branch pushed and prepare a pull request.

Never force-push.

Never delete feature branches.

Create:

`FINAL_INTEGRATION_REPORT.md`

Include:
- Every included branch and commit.
- The exact reconciliation of put_delete.go and db.go.
- The adopted Windows locking and compaction implementations.
- The torture-harness reconciliation.
- Inclusion of Member 4 commit `8f3634c`.
- Build and test evidence.
- Member 2 and Member 4 verification mapping.
- Final integration hash.
- Final main hash, if merged.
- Remaining blockers.

Execute this mission now.

Do not stop at planning. Produce the integrated, buildable project and report actual results.
```

---

## Prompt 13: Final Judging Documentation Audit & Project Organization
**Timestamp:** `2026-11-07T03:17:24Z`  
**Transcript Step Index:** `890`  

```text
# FSYNC-OR-DIE — FINAL JUDGING DOCUMENTATION AUDIT & PROJECT ORGANIZATION
## TEAM INNOVEXA | FINAL SUBMISSION PREPARATION
### Role: Documentation Auditor, Software Architect and Release Organizer
### Authority: Tech Lead

Your mission is to inspect the complete, integrated Team Innovexa repository, identify every missing challenge deliverable, create missing documentation, and organize the project into a professional, judge-ready structure.

Do not assume that a document is missing merely because an earlier report did not mention it.

Inspect the actual GitHub repository and current filesystem first.

**Your responsibility is to make the repository complete, correctly organized, technically accurate and easy for judges to inspect.**

---

# 1. PROJECT INFORMATION

Project: Fsync-or-Die

Team: Innovexa

Repository:

`https://github.com/madhuharshag/Team-Innovexa.git`

Latest reported main commit:

`b74c79f`

Latest reported integration branch:

`integration/final-demo`

Project language: Go

Project type: Crash-safe key-value database.

The final release has already been integrated.

Your task is primarily documentation auditing, deliverable completion and project organization.

Do not redesign or rewrite the database engine.

# 2. PHASE A — INSPECT THE ACTUAL RELEASE

Start by inspecting the repository.

Run:

`git status --short --branch`

`git remote -v`

`git fetch origin --prune`

`git branch -vv`

`git log --graph --oneline --decorate -30`

Verify the actual latest commit on `origin/main`.

Use the released main branch as the authoritative project baseline.

Do not work from an outdated feature branch.

Identify all existing files, folders, documents, tests, scripts, diagrams and handoff reports.

Search the entire repository for:

- `README.md`
- `ARCHITECTURE.md`
- `BREAKING.md`
- `PROMPTS.md`
- `CLI_USAGE.md`
- `DEMO.md`
- `FINAL_INTEGRATION_REPORT.md`
- `HANDOFF_MEMBER1.md`
- `HANDOFF_MEMBER2.md`
- `HANDOFF_MEMBER3.md`
- `HANDOFF_MEMBER4.md`
- `MEMBER4_FINAL_SAFETY_VERIFICATION.md`
- Existing architecture diagrams.
- Existing test evidence.
- Existing official challenge documents.

If `BREAKING.md` already exists, audit and improve it.

If it does not exist, create it.

Apply the same inspect-first approach to every other required deliverable.

Do not create duplicate documents merely because they appear in different locations.

# 3. PHASE B — VERIFY THE OFFICIAL CHALLENGE REQUIREMENTS

Search the repository for the original challenge PDF, challenge specification and official submission instructions.

If the official PDF is present, inspect its actual contents and extract:

- Mandatory deliverables.
- Required filenames and folder structure.
- Scoring criteria.
- Architecture and proof requirements.
- Crash-testing requirements.
- Design-extension requirements.
- Submission rules.
- Demonstration requirements.

Use the official challenge document as the primary authority.

If the PDF is absent, use the existing team handoffs, integration report and established challenge specifications.

Clearly record that the original PDF was unavailable.

Do not invent additional requirements or claim that an unofficial checklist is the official challenge specification.

Create:

`JUDGING_CHECKLIST.md`

This must map every official judging requirement to:

1. Required artifact.
2. Actual repository file or source directory.
3. Relevant implementation.
4. Supporting test or evidence.
5. Marks allocated, if confirmed by the official document.
6. Verification status.
7. Remaining action, if any.

Use evidence labels:

- VERIFIED
- CODE REVIEWED
- DOCUMENTED
- NOT TESTED
- MISSING
- BLOCKED

The checklist must be honest and traceable.

# 4. PHASE C — REQUIRED PROJECT STRUCTURE

Inspect the current structure and organize it into the following logical structure, adapting to the actual challenge requirements.

Target structure:

`Team-Innovexa/`

- `README.md`
- `ARCHITECTURE.md`
- `BREAKING.md`
- `PROMPTS.md`
- `JUDGING_CHECKLIST.md`
- `FINAL_INTEGRATION_REPORT.md`

`diagrams/`

- Architecture overview.
- Write and durability flow.
- Transaction flow.
- Recovery flow.
- Compaction flow.
- Crash-point illustrations.

`src/`

- Existing Go key-value database implementation.

`torture/`

- Independent crash-torture harness.
- Worker.
- Oracle.
- Runner.
- Campaigns.
- Tests.

`cmd/kip/`

- CLI entry point.
- CLI commands.
- Batch handler.
- Serve IPC implementation.
- CLI tests.

`docs/`

- `CLI_USAGE.md`
- `DEMO.md`
- Member handoff documents.
- Safety verification report.
- Supporting technical notes.

`scripts/`

- Existing Windows or WSL verification scripts.
- Existing reproducible test scripts.
- Existing demonstration helpers, if present.

### Important organization rules

The target structure is a logical guide, not permission to break the project.

Preserve existing Go package locations and import paths.

Do not move source files merely to make the directory tree look attractive.

Keep official challenge-required filenames and root locations exactly where required.

Do not move `BREAKING.md`, `ARCHITECTURE.md` or `PROMPTS.md` into `docs/` if the official challenge expects them at the repository root.

Go test files may remain alongside their packages. Do not create duplicate test directories merely for appearance.

Preserve all handoffs and evidence documents.

If you relocate supplementary documents, update their references and links.

Never delete existing information without a verified reason.

# 5. PHASE D — README.md AUDIT

Ensure the root README contains:

1. Project name.
2. Team Innovexa.
3. Problem statement.
4. What the database does.
5. Key technical features.
6. Architecture overview.
7. Data durability model.
8. Atomic batch support.
9. Crash recovery.
10. Compaction.
11. Torture testing.
12. CLI commands.
13. Build instructions.
14. Test instructions.
15. Demonstration instructions.
16. Platform verification matrix.
17. Known limitations.
18. Links to all essential technical documentation.

The README must tell judges how to understand, build and run the project quickly.

Do not make unsupported claims.

Do not describe process-kill testing as proof of physical power-loss safety.

# 6. PHASE E — ARCHITECTURE.md AUDIT

This is a critical judging document.

Inspect the actual architecture implementation.

The required architecture documentation must include:

### A. System overview

Explain the relationship between:

- CLI.
- Engine API.
- Append-only storage.
- Transaction format.
- CRC verification.
- In-memory index.
- Recovery.
- Compaction.
- File locking.
- Torture testing.

### B. Architecture diagram

Create a clear architecture diagram using Mermaid or another editable source format.

It must reflect the real implementation.

### C. Write path

Explain the actual order of:

1. Input validation.
2. Capacity preflight.
3. Transaction header construction.
4. Operation payload encoding.
5. Commit footer construction.
6. File append.
7. Synchronization.
8. Index publication.
9. Transaction accounting.
10. Successful return and CLI ACK.

Verify the order against actual source code.

Do not simplify or rearrange the actual durability sequence.

### D. Recovery path

Explain actual recovery behavior, checksums, commit markers, torn trailing transactions, index reconstruction and interrupted recovery.

### E. Compaction path

Explain the actual dual-bounded batch approach, temporary file, synchronization, rename, descriptor replacement, index replacement and rollback.

### F. Required proof table

If the challenge specification requires a 32-row proof table, include exactly the required 32 meaningful proof entries.

For every row provide:

- Invariant or claim.
- Why it holds.
- Relevant implementation location.
- Supporting test.
- Verification status.

Do not use 32 repetitive or meaningless entries simply to fill the table.

### G. Three guarantee arguments

Provide three complete technical arguments grounded in the actual implementation.

At minimum, cover the relevant guarantees required by the challenge, including:

- Acknowledged-write durability under the documented failure model.
- Atomic batch behavior and recovery.
- Safe recovery and compaction behavior.

Adjust the precise claims to the official challenge wording.

Every guarantee must explain assumptions and limitations.

# 7. PHASE F — BREAKING.md CREATION OR AUDIT

This document is a critical judging deliverable.

First verify whether it already exists.

If missing, create it in the required location.

It must explain how the database behaves under failures.

Include the following topics where applicable to the real implementation:

### A. Failure model

Distinguish clearly between:

- Process termination.
- Operating-system crash.
- Interrupted file operations.
- Torn writes.
- Device or power failure.
- Filesystem and storage hardware behavior.

State exactly which failures are covered and which are outside the verified model.

### B. Transaction failure points

Analyze:

- Crash before a transaction starts.
- Crash during transaction header writing.
- Crash during payload writing.
- Crash during commit-footer writing.
- Crash during synchronization.
- Crash after successful synchronization but before ACK.
- Crash after ACK.

For every case explain:
- What may exist on disk.
- What recovery observes.
- Whether the operation is acknowledged.
- Whether it is preserved, rejected or indeterminate.

### C. Atomic batch failure points

Explain how batches are encoded and committed.

Explain why recovery must expose either the whole batch or none of it.

Refer to the actual tests.

### D. Recovery interruption

Explain how the next database opening handles an interrupted recovery process, if supported.

### E. Compaction interruption

Analyze:
- Temporary file creation.
- Temporary data writing.
- Synchronization.
- Rename.
- Descriptor switch.
- Rollback after failure.

Ground every explanation in the actual source code.

### F. Capacity and disk-full conditions

Explain:
- Capacity preflight.
- ErrFull handling.
- Tombstone space requirements.
- Whether a deletion may fail when capacity is exhausted.
- ACK behavior.

### G. Concurrency and locking failures

Explain:
- Multiple writers.
- Windows LockFileEx.
- POSIX flock.
- Busy-lock behavior.
- Lock release after process termination.

### H. Crash-point proof table

Create a detailed table with:

| Failure scenario | Relevant code | Expected state | Recovery action | Supporting test | Evidence status |

Use real source references and test names.

Do not invent tests.

If a scenario is described by source inspection but has no direct test, label it CODE REVIEWED rather than VERIFIED.

### I. Known limitations

Clearly describe the limitations of the tested failure model.

Do not claim real physical power-loss protection merely because process termination tests pass.

Do not equate successful `File.Sync()` calls with independently verified behavior of every storage device.

# 8. PHASE G — PROMPTS.md AUDIT

The challenge expects actual chronological, unedited prompts where specified.

Locate original prompts that were preserved in:
- Existing project files.
- Team handoff documents.
- Prompt archives.
- Available development notes.

Preserve actual prompts verbatim.

Organize them chronologically if dates or order can be established.

Do not rewrite their content.

Do not fabricate prompts.

Do not reconstruct missing conversations and label them original.

If some original prompts cannot be recovered, document that limitation transparently.

A useful organization may be:

- Project master prompt.
- Member 1 prompts.
- Member 2 prompts.
- Member 3 prompts.
- Member 4 prompts.
- Important debugging and hardening prompts.

Use actual source material only.

# 9. PHASE H — DIAGRAMS DIRECTORY

Inspect existing diagrams.

Create or update useful architecture diagrams from the real implementation.

Recommended diagrams:

1. Overall system architecture.
2. Put/write durability sequence.
3. Transaction binary structure.
4. Recovery process.
5. Compaction process.
6. Crash failure points.
7. Torture harness process interaction.

Where suitable, preserve editable Mermaid source files such as:

- `diagrams/architecture.mmd`
- `diagrams/write-path.mmd`
- `diagrams/recovery-flow.mmd`
- `diagrams/compaction-flow.mmd`
- `diagrams/torture-flow.mmd`

Do not create misleading decorative diagrams.

Ensure every arrow and component reflects the real source.

Link diagrams from README and ARCHITECTURE.md.

# 10. PHASE I — CLI AND DEMONSTRATION DOCUMENTS

Inspect and preserve the completed Member 4 documents.

Ensure:

`docs/CLI_USAGE.md`

contains every implemented command, exact syntax, input format, output behavior, exit codes and serve protocol.

`docs/DEMO.md`

contains an accurate judge-ready sequence showing:

1. Build.
2. Open.
3. Put.
4. Get.
5. Delete.
6. Batch.
7. Dump.
8. Stats.
9. Check.
10. Compact where supported.
11. Reopen.
12. Verify persisted records.

Use a dedicated demo database.

Never use a torture-test database as demonstration data.

Ensure all Windows commands use valid PowerShell syntax.

# 11. PHASE J — HANDOFF AND EVIDENCE DOCUMENTS

Retain the team handoff and verification documents.

Review:

- `HANDOFF_MEMBER1.md`
- `HANDOFF_MEMBER2.md`
- `HANDOFF_MEMBER3.md`
- `HANDOFF_MEMBER4.md`
- `MEMBER4_FINAL_SAFETY_VERIFICATION.md`
- `FINAL_INTEGRATION_REPORT.md`

Keep source-specific test reports and integration evidence.

If a report is outdated, do not silently rewrite historical results.

Add a clear note describing which later commits superseded it.

Create a consistent documentation index so judges can locate all evidence quickly.

# 12. PHASE K — SOURCE AND DOCUMENTATION CONSISTENCY

Cross-check every technical statement against the actual implementation.

Use actual paths and function names.

For each important claim:
- Identify the source implementation.
- Identify the supporting test.
- Identify any relevant limitations.

Do not make unsupported claims about:
- Physical persistence.
- Power failure.
- Atomicity.
- Recovery speed.
- Platform support.
- Concurrency.
- Test coverage.

Do not invent benchmark results.

Do not change engine code to make documentation claims appear true.

If an inconsistency is found, document it and notify the Tech Lead.

# 13. PHASE L — PROJECT TREE AND ORGANIZATION REVIEW

After auditing and completing the documents, provide the actual final tree.

It should clearly distinguish:

- Production implementation.
- CLI.
- Torture harness.
- Tests.
- Challenge-required documents.
- Supporting team reports.
- Diagrams.
- Scripts.

Remove no files merely because they appear redundant.

If two documents genuinely duplicate one another, preserve their content and recommend consolidation rather than deleting them automatically.

Keep paths stable wherever possible.

Ensure README links work.

Ensure all relative links in Markdown documents resolve correctly.

# 14. PHASE M — FINAL VALIDATION

After documentation changes:

Run:

`git diff --check`

`go build ./...`

`go vet ./...`

`go test -count=1 ./...`

Do not rerun lengthy torture campaigns unless a relevant code change makes that necessary.

Since this is a documentation and organization mission, do not modify working production code.

If tests fail due to environmental or existing code issues:
- Record the exact failure.
- Do not hide it.
- Do not remove tests.
- Do not claim success.

Validate Markdown links and ensure referenced files actually exist.

Check that all required documents are discoverable from the README.

# 15. GIT AND RELEASE SAFETY

The final integrated release is already published on main.

Do not overwrite it.

Create a separate branch:

`docs/final-judging-pack`

Base it on the latest `origin/main`.

Make documentation and safe organization changes on this branch.

Do not modify:
- Storage engine logic.
- Recovery logic.
- Compaction logic.
- Shared contracts.
- Torture algorithms.
- CLI behavior.

Commit the changes with a clear message.

Push:

`git push -u origin docs/final-judging-pack`

If all changes are documentation-only, all references are valid, and verification passes, prepare a normal merge into main.

Do not force-push.

If branch protection prevents a direct merge, leave the branch pushed and prepare a pull request.

# 16. FINAL DELIVERABLES

At the end, ensure the following are present or their absence is explicitly documented:

- README.md
- ARCHITECTURE.md
- BREAKING.md
- PROMPTS.md
- JUDGING_CHECKLIST.md
- FINAL_INTEGRATION_REPORT.md
- diagrams/
- src/
- torture/
- cmd/kip/
- Existing package tests.
- docs/CLI_USAGE.md
- docs/DEMO.md
- Team handoff reports.
- Member safety verification reports.
- scripts/

The actual official challenge specification takes precedence over this target layout.

# 17. FINAL REPORT

Create:

`FINAL_DOCUMENTATION_AUDIT_REPORT.md`

Include:

1. Original release commit inspected.
2. Official challenge document availability.
3. Every required document found.
4. Every missing document created.
5. Every document improved.
6. Final directory tree.
7. Architecture diagram list.
8. BREAKING.md coverage summary.
9. ARCHITECTURE.md proof table status.
10. PROMPTS.md provenance and completeness.
11. Judging checklist with marks where confirmed.
12. Markdown link validation results.
13. Build and test results.
14. Files moved or renamed and reasons.
15. Known limitations.
16. Final documentation branch and commit hash.
17. Whether documentation is merged into main or awaiting pull request.

Use actual evidence.

Do not report 100% completion if mandatory original prompts, proof entries or other required artifacts are unavailable.

# 18. FINAL EXECUTION INSTRUCTION

Inspect first.

Compare against the actual challenge requirements.

Find what is missing.

Create what is missing.

Improve what is incomplete.

Organize the documentation logically.

Preserve working code and all existing member contributions.

Validate every link and technical claim.

Push the completed documentation branch.

**Do not stop after producing a checklist. Execute the documentation audit and complete the missing deliverables.**

BEGIN NOW.
```

---
