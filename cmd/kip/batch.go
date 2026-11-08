package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/madhuharshag/Fsync-or-Die/src/kv"
)

// WireOpJSON represents the JSON schema for batch operations.
// Supports both {"kind": 1, "key": "k", "val": "v"} and {"op": "put", "key": "k", "value": "v"}.
type WireOpJSON struct {
	Kind  uint8  `json:"kind,omitempty"`
	Op    string `json:"op,omitempty"`
	Key   string `json:"key"`
	Val   string `json:"val,omitempty"`
	Value string `json:"value,omitempty"`
}

// cmdBatch handles: kip batch <db-path> <file>
func cmdBatch(args []string) int {
	if len(args) < 2 {
		printError(fmt.Errorf("%w: syntax: kip batch <db-path> <file>", ErrUsage))
		return ExitError
	}
	if len(args) > 2 {
		printError(fmt.Errorf("%w: unexpected extra arguments for batch", ErrUsage))
		return ExitError
	}
	dbPath := args[0]
	filePath := args[1]

	data, err := os.ReadFile(filePath)
	if err != nil {
		printError(fmt.Errorf("failed to read batch file '%s': %w", filePath, err))
		return ExitError
	}

	ops, err := parseBatchData(data)
	if err != nil {
		printError(fmt.Errorf("invalid batch file: %w", err))
		return ExitError
	}

	if len(ops) == 0 {
		printError(fmt.Errorf("%w: batch contains 0 operations", kv.ErrInvalid))
		return ExitError
	}
	if len(ops) > kv.MaxBatchOps {
		printError(fmt.Errorf("%w: batch size %d exceeds maximum limit %d", kv.ErrInvalid, len(ops), kv.MaxBatchOps))
		return ExitError
	}

	// Validate each operation's key and value bounds before calling engine
	for i, op := range ops {
		if len(op.Key) < kv.MinKeyLen || len(op.Key) > kv.MaxKeyLen {
			printError(fmt.Errorf("%w: op #%d key length %d bytes exceeds limits [%d, %d]", kv.ErrInvalid, i+1, len(op.Key), kv.MinKeyLen, kv.MaxKeyLen))
			return ExitError
		}
		if op.Kind == kv.PutOp && len(op.Value) > kv.MaxValueLen {
			printError(fmt.Errorf("%w: op #%d value length %d bytes exceeds limit %d", kv.ErrInvalid, i+1, len(op.Value), kv.MaxValueLen))
			return ExitError
		}
		if op.Kind == kv.DeleteOp && len(op.Value) > 0 {
			printError(fmt.Errorf("%w: op #%d delete operation cannot carry a value payload", kv.ErrInvalid, i+1))
			return ExitError
		}
	}

	db, _, err := kv.Open(dbPath, kv.Options{})
	if err != nil {
		printError(fmt.Errorf("open failed: %w", err))
		return ExitError
	}
	defer func() {
		_ = db.Close()
	}()

	if err := db.Batch(ops); err != nil {
		printError(fmt.Errorf("batch failed: %w", err))
		return ExitError
	}

	// Output confirmed count of durably applied operations
	fmt.Printf("ACK %d\n", len(ops))
	return ExitSuccess
}

// parseBatchData automatically detects and decodes JSON array or line-based format.
func parseBatchData(data []byte) ([]kv.Op, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, nil
	}

	// If starts with '[': parse as JSON array
	if trimmed[0] == '[' {
		return parseBatchJSON(trimmed)
	}

	// Otherwise parse as line-delimited format: "put <key> <val>" or "del <key>"
	return parseBatchLines(trimmed)
}

func parseBatchJSON(data []byte) ([]kv.Op, error) {
	var wireOps []WireOpJSON
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields() // Reject malformed / unknown fields

	if err := decoder.Decode(&wireOps); err != nil {
		return nil, fmt.Errorf("JSON parse error: %w", err)
	}

	ops := make([]kv.Op, 0, len(wireOps))
	for i, wo := range wireOps {
		var kind kv.Kind
		opName := strings.ToLower(strings.TrimSpace(wo.Op))
		if opName == "put" || wo.Kind == uint8(kv.PutOp) {
			kind = kv.PutOp
		} else if opName == "del" || opName == "delete" || wo.Kind == uint8(kv.DeleteOp) {
			kind = kv.DeleteOp
		} else {
			return nil, fmt.Errorf("op #%d has unrecognized kind '%s' / %d", i+1, wo.Op, wo.Kind)
		}

		valStr := wo.Val
		if valStr == "" && wo.Value != "" {
			valStr = wo.Value
		}

		ops = append(ops, kv.Op{
			Kind:  kind,
			Key:   []byte(wo.Key),
			Value: []byte(valStr),
		})
	}
	return ops, nil
}

func parseBatchLines(data []byte) ([]kv.Op, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	var ops []kv.Op
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue // skip empty lines and comments
		}

		parts := strings.SplitN(line, " ", 3)
		action := strings.ToLower(parts[0])

		switch action {
		case "put":
			if len(parts) < 3 {
				return nil, fmt.Errorf("line %d: 'put' requires key and value (syntax: put <key> <value>)", lineNum)
			}
			ops = append(ops, kv.Op{
				Kind:  kv.PutOp,
				Key:   []byte(parts[1]),
				Value: []byte(parts[2]),
			})

		case "del", "delete":
			if len(parts) < 2 {
				return nil, fmt.Errorf("line %d: 'del' requires key (syntax: del <key>)", lineNum)
			}
			ops = append(ops, kv.Op{
				Kind: kv.DeleteOp,
				Key:  []byte(parts[1]),
			})

		default:
			return nil, fmt.Errorf("line %d: unknown operation '%s' (must be 'put' or 'del')", lineNum, parts[0])
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading batch lines: %w", err)
	}
	return ops, nil
}
