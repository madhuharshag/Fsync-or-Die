package main

import (
	"time"
)

// OpKind represents the database operation type.
type OpKind string

const (
	OpPut     OpKind = "PUT"
	OpGet     OpKind = "GET"
	OpDelete  OpKind = "DELETE"
	OpBatch   OpKind = "BATCH"
	OpCompact OpKind = "COMPACT"
)

// AckStatus represents the verification classification of an operation.
type AckStatus string

const (
	StatusPending       AckStatus = "PENDING"
	StatusAcknowledged  AckStatus = "ACKNOWLEDGED"
	StatusRejected      AckStatus = "REJECTED"
	StatusIndeterminate AckStatus = "INDETERMINATE"
)

// OpRecord captures complete telemetry for a single database mutation or query.
type OpRecord struct {
	OpID       uint64     `json:"op_id"`
	Seed       int64      `json:"seed"`
	Kind       OpKind     `json:"kind"`
	Key        string     `json:"key"`
	ValHash    uint32     `json:"val_hash"`
	ValLen     int        `json:"val_len"`
	Value      []byte     `json:"value,omitempty"`
	BatchID    uint64     `json:"batch_id,omitempty"`
	BatchOps   []OpRecord `json:"batch_ops,omitempty"`
	Status     AckStatus  `json:"status"`
	WriterID   int        `json:"writer_id"`
	SeqNum     uint64     `json:"seq_num"`
	CrashCycle int        `json:"crash_cycle"`
	Timestamp  time.Time  `json:"timestamp"`
	ErrorMsg   string     `json:"error_msg,omitempty"`
}

// HarnessConfig defines parameters controlling the torture test run.
type HarnessConfig struct {
	Mode               string        `json:"mode"`
	Seed               int64         `json:"seed"`
	TotalOps           int           `json:"total_ops"`
	Crashes            int           `json:"crashes"`
	Writers            int           `json:"writers"`
	Readers            int           `json:"readers"`
	Duration           time.Duration `json:"duration"`
	DbPath             string        `json:"db_path"`
	OracleLogPath      string        `json:"oracle_log_path"`
	DeterministicCrash string        `json:"deterministic_crash"`
	WorkerMode         bool          `json:"worker_mode"`
	MaxKeyLen          int           `json:"max_key_len"`
	MaxValueLen        int           `json:"max_value_len"`
	MaxBatchOps        int           `json:"max_batch_ops"`
	MaxFileBytes       uint64        `json:"max_file_bytes"`
}

// HarnessReport encapsulates the final audit results and environment metrics.
type HarnessReport struct {
	Mode             string        `json:"mode"`
	Seed             int64         `json:"seed"`
	GoVersion        string        `json:"go_version"`
	OSArch           string        `json:"os_arch"`
	BinaryPath       string        `json:"binary_path"`
	OpsAttempted     uint64        `json:"ops_attempted"`
	OpsAcknowledged  uint64        `json:"ops_acknowledged"`
	OpsIndeterminate uint64        `json:"ops_indeterminate"`
	OpsRejected      uint64        `json:"ops_rejected"`
	Crashes          int           `json:"crashes"`
	Recoveries       int           `json:"recoveries"`
	Duration         time.Duration `json:"duration"`
	Passed           bool          `json:"passed"`
	VerificationMsg  string        `json:"verification_msg"`
	FailureHistory   []OpRecord    `json:"failure_history,omitempty"`
}
