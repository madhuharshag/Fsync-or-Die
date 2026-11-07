package main

import (
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/madhuharshag/Team-Innovexa/src/kv"
)

// parseMaxBytesFlag extracts an optional --max-bytes or -max-bytes flag from arguments.
func parseMaxBytesFlag(args []string) (string, uint64, error) {
	if len(args) == 0 {
		return "", 0, fmt.Errorf("%w: missing database path", ErrUsage)
	}

	var dbPath string
	var maxBytes uint64

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--max-bytes" || arg == "-max-bytes" {
			if i+1 >= len(args) {
				return "", 0, fmt.Errorf("%w: --max-bytes requires a numeric value", ErrUsage)
			}
			val, err := strconv.ParseUint(args[i+1], 10, 64)
			if err != nil {
				return "", 0, fmt.Errorf("invalid --max-bytes value '%s': %w", args[i+1], err)
			}
			maxBytes = val
			i++
		} else if stringsHasPrefix(arg, "--max-bytes=") || stringsHasPrefix(arg, "-max-bytes=") {
			parts := splitEqual(arg)
			if len(parts) != 2 {
				return "", 0, fmt.Errorf("%w: invalid flag syntax", ErrUsage)
			}
			val, err := strconv.ParseUint(parts[1], 10, 64)
			if err != nil {
				return "", 0, fmt.Errorf("invalid --max-bytes value '%s': %w", parts[1], err)
			}
			maxBytes = val
		} else if dbPath == "" {
			dbPath = arg
		} else {
			return "", 0, fmt.Errorf("%w: unexpected argument '%s'", ErrUsage, arg)
		}
	}

	if dbPath == "" {
		return "", 0, fmt.Errorf("%w: missing database path", ErrUsage)
	}
	return dbPath, maxBytes, nil
}

func stringsHasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func splitEqual(s string) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}

// cmdOpen handles: kip open <db-path> [--max-bytes N]
func cmdOpen(args []string) int {
	dbPath, maxBytes, err := parseMaxBytesFlag(args)
	if err != nil {
		printError(err)
		return ExitError
	}

	opts := kv.Options{
		MaxFileBytes: maxBytes,
	}

	db, info, err := kv.Open(dbPath, opts)
	if err != nil {
		printError(fmt.Errorf("open failed: %w", err))
		return ExitError
	}
	defer func() {
		_ = db.Close()
	}()

	recMs := float64(info.RecoveryDurationNs) / 1e6
	fmt.Printf("OPENED: %s (recreated=%t, recovered_txns=%d, live_keys=%d, recovery_ms=%.3f)\n",
		dbPath, info.Recreated, info.RecoveredTransactions, info.LiveKeyCount, recMs)
	return ExitSuccess
}

// cmdPut handles: kip put <db-path> <key> <value>
func cmdPut(args []string) int {
	if len(args) < 3 {
		printError(fmt.Errorf("%w: syntax: kip put <db-path> <key> <value>", ErrUsage))
		return ExitError
	}
	if len(args) > 3 {
		printError(fmt.Errorf("%w: unexpected extra arguments for put", ErrUsage))
		return ExitError
	}
	dbPath := args[0]
	key := []byte(args[1])
	value := []byte(args[2])

	if len(key) < kv.MinKeyLen || len(key) > kv.MaxKeyLen {
		printError(fmt.Errorf("%w: key length %d bytes exceeds limits [%d, %d]", kv.ErrInvalid, len(key), kv.MinKeyLen, kv.MaxKeyLen))
		return ExitError
	}
	if len(value) > kv.MaxValueLen {
		printError(fmt.Errorf("%w: value length %d bytes exceeds maximum %d bytes", kv.ErrInvalid, len(value), kv.MaxValueLen))
		return ExitError
	}

	db, _, err := kv.Open(dbPath, kv.Options{})
	if err != nil {
		printError(fmt.Errorf("open failed: %w", err))
		return ExitError
	}
	defer func() {
		_ = db.Close()
	}()

	if err := db.Put(key, value); err != nil {
		printError(fmt.Errorf("put failed: %w", err))
		return ExitError
	}

	// Emit ACK only after database engine fsync succeeds
	fmt.Printf("ACK %s\n", args[1])
	return ExitSuccess
}

// cmdGet handles: kip get <db-path> <key>
func cmdGet(args []string) int {
	if len(args) < 2 {
		printError(fmt.Errorf("%w: syntax: kip get <db-path> <key>", ErrUsage))
		return ExitError
	}
	if len(args) > 2 {
		printError(fmt.Errorf("%w: unexpected extra arguments for get", ErrUsage))
		return ExitError
	}
	dbPath := args[0]
	key := []byte(args[1])

	if len(key) < kv.MinKeyLen || len(key) > kv.MaxKeyLen {
		printError(fmt.Errorf("%w: key length %d bytes exceeds limits [%d, %d]", kv.ErrInvalid, len(key), kv.MinKeyLen, kv.MaxKeyLen))
		return ExitError
	}

	db, _, err := kv.Open(dbPath, kv.Options{ReadOnly: true})
	if err != nil {
		printError(fmt.Errorf("open failed: %w", err))
		return ExitError
	}
	defer func() {
		_ = db.Close()
	}()

	val, err := db.Get(key)
	if err != nil {
		if errors.Is(err, kv.ErrNotFound) {
			// Per challenge specification: missing key outputs NOTFOUND to stdout and exits with code 2
			fmt.Println("NOTFOUND")
			return ExitNotFound
		}
		printError(fmt.Errorf("get failed: %w", err))
		return ExitError
	}

	// Print raw value followed by newline
	fmt.Println(string(val))
	return ExitSuccess
}

// cmdDel handles: kip del <db-path> <key>
func cmdDel(args []string) int {
	if len(args) < 2 {
		printError(fmt.Errorf("%w: syntax: kip del <db-path> <key>", ErrUsage))
		return ExitError
	}
	if len(args) > 2 {
		printError(fmt.Errorf("%w: unexpected extra arguments for del", ErrUsage))
		return ExitError
	}
	dbPath := args[0]
	key := []byte(args[1])

	if len(key) < kv.MinKeyLen || len(key) > kv.MaxKeyLen {
		printError(fmt.Errorf("%w: key length %d bytes exceeds limits [%d, %d]", kv.ErrInvalid, len(key), kv.MinKeyLen, kv.MaxKeyLen))
		return ExitError
	}

	db, _, err := kv.Open(dbPath, kv.Options{})
	if err != nil {
		printError(fmt.Errorf("open failed: %w", err))
		return ExitError
	}
	defer func() {
		_ = db.Close()
	}()

	if err := db.Delete(key); err != nil {
		printError(fmt.Errorf("delete failed: %w", err))
		return ExitError
	}

	fmt.Printf("ACK %s\n", args[1])
	return ExitSuccess
}

// cmdCheck handles: kip check <db-path>
func cmdCheck(args []string) int {
	if len(args) < 1 {
		printError(fmt.Errorf("%w: syntax: kip check <db-path>", ErrUsage))
		return ExitError
	}
	if len(args) > 1 {
		printError(fmt.Errorf("%w: unexpected extra arguments for check", ErrUsage))
		return ExitError
	}
	dbPath := args[0]

	db, info, err := kv.Open(dbPath, kv.Options{ReadOnly: true})
	if err != nil {
		printError(fmt.Errorf("open failed: %w", err))
		return ExitError
	}
	defer func() {
		_ = db.Close()
	}()

	if err := db.Check(); err != nil {
		printError(fmt.Errorf("integrity check failed: %w", err))
		return ExitError
	}

	stats := db.Stats()
	txnCount := stats.TxCount
	if txnCount == 0 && info.RecoveredTransactions > 0 {
		txnCount = info.RecoveredTransactions
	}

	fmt.Printf("OK: Database integrity verified (%d transactions)\n", txnCount)
	return ExitSuccess
}

// entry represents a key-value record for deterministic sorted printing
type kvEntry struct {
	key string
	val string
}

// cmdDump handles: kip dump <db-path>
func cmdDump(args []string) int {
	if len(args) < 1 {
		printError(fmt.Errorf("%w: syntax: kip dump <db-path>", ErrUsage))
		return ExitError
	}
	if len(args) > 1 {
		printError(fmt.Errorf("%w: unexpected extra arguments for dump", ErrUsage))
		return ExitError
	}
	dbPath := args[0]

	db, _, err := kv.Open(dbPath, kv.Options{ReadOnly: true})
	if err != nil {
		printError(fmt.Errorf("open failed: %w", err))
		return ExitError
	}
	defer func() {
		_ = db.Close()
	}()

	var entries []kvEntry
	err = db.Scan(func(key, value []byte) error {
		entries = append(entries, kvEntry{
			key: string(key),
			val: string(value),
		})
		return nil
	})
	if err != nil {
		printError(fmt.Errorf("scan failed: %w", err))
		return ExitError
	}

	if len(entries) == 0 {
		fmt.Println("(empty database)")
		return ExitSuccess
	}

	// Sort entries deterministically by key (lexicographical order)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].key < entries[j].key
	})

	for _, e := range entries {
		fmt.Printf("%s = %s\n", e.key, e.val)
	}
	return ExitSuccess
}

// cmdStats handles: kip stats <db-path>
func cmdStats(args []string) int {
	if len(args) < 1 {
		printError(fmt.Errorf("%w: syntax: kip stats <db-path>", ErrUsage))
		return ExitError
	}
	if len(args) > 1 {
		printError(fmt.Errorf("%w: unexpected extra arguments for stats", ErrUsage))
		return ExitError
	}
	dbPath := args[0]

	db, _, err := kv.Open(dbPath, kv.Options{ReadOnly: true})
	if err != nil {
		printError(fmt.Errorf("open failed: %w", err))
		return ExitError
	}
	defer func() {
		_ = db.Close()
	}()

	stats := db.Stats()
	fmt.Printf("LiveKeys: %d\n", stats.LiveKeys)
	fmt.Printf("FileSizeBytes: %d\n", stats.FileSizeBytes)
	fmt.Printf("MaxFileBytes: %d\n", stats.MaxFileBytes)
	fmt.Printf("LastTxID: %d\n", stats.LastTxID)
	fmt.Printf("TxCount: %d\n", stats.TxCount)
	fmt.Printf("CompactionCount: %d\n", stats.CompactionCount)
	return ExitSuccess
}

// cmdCompact handles: kip compact <db-path>
func cmdCompact(args []string) int {
	if len(args) < 1 {
		printError(fmt.Errorf("%w: syntax: kip compact <db-path>", ErrUsage))
		return ExitError
	}
	if len(args) > 1 {
		printError(fmt.Errorf("%w: unexpected extra arguments for compact", ErrUsage))
		return ExitError
	}
	dbPath := args[0]

	db, _, err := kv.Open(dbPath, kv.Options{})
	if err != nil {
		printError(fmt.Errorf("open failed: %w", err))
		return ExitError
	}
	defer func() {
		_ = db.Close()
	}()

	if err := db.Compact(); err != nil {
		printError(fmt.Errorf("compaction failed: %w", err))
		return ExitError
	}

	stats := db.Stats()
	fmt.Printf("OK: Compaction completed (new size: %d bytes)\n", stats.FileSizeBytes)
	return ExitSuccess
}
