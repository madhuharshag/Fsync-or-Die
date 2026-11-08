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

// runWorkerProcess executes the child worker process loop.
// It serves requests from the parent harness over standard I/O pipes.
func runWorkerProcess() {
	scanner := bufio.NewScanner(os.Stdin)
	// Allow large lines (e.g. 2MB values base64 encoded)
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	var db kv.Engine

	respond := func(format string, args ...any) {
		msg := fmt.Sprintf(format, args...)
		fmt.Println(msg)
		_ = os.Stdout.Sync()
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, " ", 4)
		cmd := parts[0]

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
				respond("ERR 0 invalid put format")
				continue
			}
			opID := parts[1]
			keyBytes, err1 := base64.StdEncoding.DecodeString(parts[2])
			valBytes, err2 := base64.StdEncoding.DecodeString(parts[3])
			if err1 != nil || err2 != nil {
				respond("ERR %s base64 decode failed", opID)
				continue
			}

			if err := db.Put(keyBytes, valBytes); err != nil {
				respond("ERR %s %v", opID, err)
			} else {
				respond("ACK %s", opID)
			}

		case "GET":
			if db == nil {
				respond("ERR 0 db not open")
				continue
			}
			if len(parts) < 3 {
				respond("ERR 0 invalid get format")
				continue
			}
			opID := parts[1]
			keyBytes, err := base64.StdEncoding.DecodeString(parts[2])
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

		case "DEL":
			if db == nil {
				respond("ERR 0 db not open")
				continue
			}
			if len(parts) < 3 {
				respond("ERR 0 invalid del format")
				continue
			}
			opID := parts[1]
			keyBytes, err := base64.StdEncoding.DecodeString(parts[2])
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
				respond("ERR 0 invalid batch format")
				continue
			}
			opID := parts[1]
			rawJson, err := base64.StdEncoding.DecodeString(parts[2])
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
			if err := json.Unmarshal(rawJson, &wireOps); err != nil {
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
			return

		case "PING":
			respond("PONG")

		default:
			respond("ERR 0 unknown command: %s", cmd)
		}
	}
}
