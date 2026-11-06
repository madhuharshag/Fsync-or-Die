package kv

import (
	"errors"
)

// Standard error taxonomy for Fsync-or-Die.
// All errors are sentinel values compatible with errors.Is.
var (
	// ErrNotFound is returned when Get requests a key that does not exist or was deleted.
	ErrNotFound = errors.New("kv: key not found")

	// ErrInvalid indicates an invalid argument, key/value size violation, or illegal operation.
	ErrInvalid = errors.New("kv: invalid argument or operation")

	// ErrFull is returned when an operation would exceed the configured maximum file capacity.
	ErrFull = errors.New("kv: maximum database capacity exceeded")

	// ErrClosed is returned when an operation is attempted on a closed database handle.
	ErrClosed = errors.New("kv: database is closed")

	// ErrBusy is returned when the database file cannot be locked because another process holds it.
	ErrBusy = errors.New("kv: database file is locked by another process")

	// ErrCorrupt is returned when invalid magic, mismatched lengths, or checksum failures occur.
	ErrCorrupt = errors.New("kv: corrupt or torn record encountered")

	// ErrVersion is returned when encountering an unsupported file or transaction format version.
	ErrVersion = errors.New("kv: unsupported file or transaction format version")

	// ErrConfig indicates an invalid database option or configuration setting.
	ErrConfig = errors.New("kv: invalid database configuration")

	// ErrUncertain indicates that a low-level synchronization failure occurred after bytes were written;
	// durability cannot be guaranteed and the database handle must be treated with caution.
	ErrUncertain = errors.New("kv: transaction sync status uncertain after write")

	// ErrPoisoned is returned when the database handle has experienced a fatal I/O failure and is inoperable.
	ErrPoisoned = errors.New("kv: database handle is poisoned due to previous fatal I/O failure")
)

// Challenge specification limits.
const (
	// MinKeyLen is the minimum permitted length of a key in bytes.
	MinKeyLen = 1

	// MaxKeyLen is the maximum permitted length of a key in bytes (1,024 bytes).
	MaxKeyLen = 1024

	// MaxValueLen is the maximum permitted length of a value in bytes (1 MB = 1,048,576 bytes).
	MaxValueLen = 1024 * 1024

	// MaxBatchOps is the maximum number of operations allowed in a single atomic batch.
	MaxBatchOps = 10000
)

// Kind represents the operation kind within a transaction payload.
type Kind uint8

const (
	// PutOp stores or overwrites a key-value pair.
	PutOp Kind = 1

	// DeleteOp records a deletion tombstone for a key.
	DeleteOp Kind = 2
)

// Op represents a single mutation within a transaction or batch.
type Op struct {
	Kind  Kind
	Key   []byte
	Value []byte
}

// Engine defines the primary public contract for the embedded key-value database.
// All public methods are safe for concurrent use by multiple goroutines.
type Engine interface {
	// Put stores or overwrites a key-value pair. Returning nil promises that the
	// mutation is durably synchronized to disk.
	Put(key, value []byte) error

	// Get retrieves the current value for key. Returns ErrNotFound if missing or deleted.
	// The returned slice is caller-owned and safe from internal mutation.
	Get(key []byte) ([]byte, error)

	// Delete durably records the removal of key. Deleting a missing key succeeds with nil error.
	Delete(key []byte) error

	// Batch atomically executes a series of Put and Delete operations in list order.
	Batch(ops []Op) error

	// Compact reclaims storage by writing live entries to a temporary replacement file
	// and publishing it via an atomic descriptor and file transition.
	Compact() error

	// Check scans and verifies format, transaction boundaries, and dual CRCs.
	Check() error

	// Scan traverses all live key-value pairs in the database.
	// fn must not reenter the Engine.
	Scan(fn func(key, value []byte) error) error

	// Stats returns snapshot telemetry for the database.
	Stats() Stats

	// Close flushes buffers, releases locks, and closes file handles.
	Close() error
}

// Options configures the database upon opening.
type Options struct {
	// MaxFileBytes sets the maximum allowable authoritative file size. 0 means unlimited.
	MaxFileBytes uint64

	// ReadOnly opens the database in read-only mode, rejecting all mutation operations.
	ReadOnly bool

	// SyncWrites is retained for API compatibility. In Fsync-or-Die, all successful
	// mutations unconditionally invoke fsync to guarantee durability.
	SyncWrites bool
}

// OpenInfo reports metadata discovered during Open and crash recovery.
type OpenInfo struct {
	// Recreated indicates whether a brand-new file header was initialized.
	Recreated bool

	// RecoveredTransactions is the count of valid transactions replayed.
	RecoveredTransactions uint64

	// LastTxID is the highest transaction ID recovered.
	LastTxID uint64

	// LiveKeyCount is the count of active keys loaded into the RAM index.
	LiveKeyCount uint64

	// RepairedTailBytes is the number of truncated uncommitted or torn tail bytes discarded.
	RepairedTailBytes uint64

	// RecoveryDurationNs is the measured nanoseconds spent executing Open and recovery scan.
	RecoveryDurationNs int64
}

// Stats contains runtime operational counters and measurements.
type Stats struct {
	LiveKeys        uint64
	FileSizeBytes   uint64
	MaxFileBytes    uint64
	LastTxID        uint64
	TxCount         uint64
	CompactionCount uint64
}
