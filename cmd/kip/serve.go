package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/madhuharshag/Fsync-or-Die/src/kv"
)

// cmdServe handles: kip serve <db-path> [--max-bytes N]
func cmdServe(args []string) int {
	var dbPath string
	var maxBytes uint64

	if len(args) > 0 {
		var err error
		dbPath, maxBytes, err = parseMaxBytesFlag(args)
		if err != nil {
			printError(err)
			return ExitError
		}
	}

	return runServeMode(dbPath, maxBytes, true)
}

// runServeMode implements the long-running IPC stdin/stdout loop.
// Compatible with Member 3's independent torture harness (torture/worker.go & torture/runner.go)
// as well as interactive manual judge operation.
func runServeMode(initialPath string, maxBytes uint64, autoOpen bool) int {
	scanner := bufio.NewScanner(os.Stdin)
	// Allocate a 1 MB buffer capable of expanding to 10 MB to accommodate 1 MB+ values
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	var db kv.Engine

	respond := func(format string, args ...any) {
		msg := fmt.Sprintf(format, args...)
		fmt.Println(msg)
		_ = os.Stdout.Sync()
	}

	defer func() {
		if db != nil {
			_ = db.Close()
		}
	}()

	// If initial database path provided, open it immediately and emit READY
	if autoOpen && initialPath != "" {
		opts := kv.Options{MaxFileBytes: maxBytes}
		var err error
		db, _, err = kv.Open(initialPath, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: serve open failed: %v\n", err)
			return ExitError
		}
		respond("READY")
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, " ", 4)
		cmd := strings.ToUpper(parts[0])

		switch cmd {
		case "OPEN":
			if len(parts) < 2 {
				respond("ERR 0 missing db path")
				continue
			}
			dbPath := parts[1]
			opts := kv.Options{}
			if len(parts) > 2 {
				if mb, err := strconv.ParseUint(parts[2], 10, 64); err == nil {
					opts.MaxFileBytes = mb
				}
			}
			if db != nil {
				_ = db.Close()
				db = nil
			}
			var err error
			db, _, err = kv.Open(dbPath, opts)
			if err != nil {
				respond("ERR 0 open failed: %v", err)
				continue
			}
			respond("READY")

		case "PUT":
			if db == nil {
				respond("ERR 0 db not open")
				continue
			}
			if len(parts) < 4 {
				respond("ERR 0 invalid put format: expected PUT <opID> <b64key> <b64val>")
				continue
			}
			opID := parts[1]
			keyBytes, err1 := decodeBytes(parts[2])
			valBytes, err2 := decodeBytes(parts[3])
			if err1 != nil || err2 != nil {
				respond("ERR %s base64 decode failed", opID)
				continue
			}

			if err := db.Put(keyBytes, valBytes); err != nil {
				respond("ERR %s %v", opID, err)
			} else {
				// Durability invariant: Emit ACK only after db.Put returns nil (after fsync)
				respond("ACK %s", opID)
			}

		case "GET":
			if db == nil {
				respond("ERR 0 db not open")
				continue
			}
			if len(parts) < 3 {
				respond("ERR 0 invalid get format: expected GET <opID> <b64key>")
				continue
			}
			opID := parts[1]
			keyBytes, err := decodeBytes(parts[2])
			if err != nil {
				respond("ERR %s base64 decode failed", opID)
				continue
			}

			val, err := db.Get(keyBytes)
			if err != nil {
				if errors.Is(err, kv.ErrNotFound) {
					respond("NOT_FOUND %s", opID)
				} else {
					respond("ERR %s %v", opID, err)
				}
			} else {
				encVal := base64.StdEncoding.EncodeToString(val)
				respond("VAL %s %s", opID, encVal)
			}

		case "DEL", "DELETE":
			if db == nil {
				respond("ERR 0 db not open")
				continue
			}
			if len(parts) < 3 {
				respond("ERR 0 invalid del format: expected DEL <opID> <b64key>")
				continue
			}
			opID := parts[1]
			keyBytes, err := decodeBytes(parts[2])
			if err != nil {
				respond("ERR %s base64 decode failed", opID)
				continue
			}

			if err := db.Delete(keyBytes); err != nil {
				respond("ERR %s %v", opID, err)
			} else {
				respond("ACK %s", opID)
			}

		case "BATCH":
			if db == nil {
				respond("ERR 0 db not open")
				continue
			}
			if len(parts) < 3 {
				respond("ERR 0 invalid batch format: expected BATCH <opID> <b64json>")
				continue
			}
			opID := parts[1]
			rawJSON, err := decodeBytes(parts[2])
			if err != nil {
				respond("ERR %s base64 decode failed", opID)
				continue
			}

			type WireOp struct {
				Kind  uint8  `json:"kind"`
				Key   string `json:"key"`
				Value string `json:"val"`
			}
			var wireOps []WireOp
			if err := json.Unmarshal(rawJSON, &wireOps); err != nil {
				respond("ERR %s json unmarshal failed: %v", opID, err)
				continue
			}

			ops := make([]kv.Op, len(wireOps))
			for i, wo := range wireOps {
				ops[i] = kv.Op{
					Kind:  kv.Kind(wo.Kind),
					Key:   []byte(wo.Key),
					Value: []byte(wo.Value),
				}
			}

			if err := db.Batch(ops); err != nil {
				respond("ERR %s %v", opID, err)
			} else {
				respond("ACK %s", opID)
			}

		case "COMPACT":
			if db == nil {
				respond("ERR 0 db not open")
				continue
			}
			opID := "0"
			if len(parts) > 1 {
				opID = parts[1]
			}
			if err := db.Compact(); err != nil {
				respond("ERR %s compact failed: %v", opID, err)
			} else {
				respond("ACK %s", opID)
			}

		case "CLOSE":
			opID := "0"
			if len(parts) > 1 {
				opID = parts[1]
			}
			if db != nil {
				_ = db.Close()
				db = nil
			}
			respond("ACK %s", opID)
			return ExitSuccess

		case "PING":
			respond("PONG")

		default:
			respond("ERR 0 unknown command: %s", cmd)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "serve pipe error: %v\n", err)
		return ExitError
	}
	return ExitSuccess
}

// decodeBytes tries standard base64 decoding first; if that fails and input is not valid base64,
// returns base64 decode error to conform with strict wire protocol.
func decodeBytes(s string) ([]byte, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err == nil {
		return b, nil
	}
	return nil, err
}
