package main

import (
	"fmt"
	"io"
)

const (
	AppName    = "kip"
	AppVersion = "1.0.0"
	AppDesc    = "Fsync-or-Die Crash-Safe Embedded Key-Value Database CLI"
)

func printVersion(w io.Writer) {
	fmt.Fprintf(w, "%s version %s (Team Innovexa)\n", AppName, AppVersion)
}

func printUsage(w io.Writer) {
	fmt.Fprintf(w, `%s - %s

USAGE:
  %s <command> [arguments...]

COMMANDS:
  open  <db-path> [--max-bytes N]
      Opens or creates the database, executes recovery, and reports telemetry.
      Example: %s open ./data.kip --max-bytes 50000000

  put   <db-path> <key> <value>
      Durably stores or updates a key-value pair with fsync.
      Outputs 'ACK <key>' on confirmed disk persistence.
      Example: %s put ./data.kip rice 500kg

  get   <db-path> <key>
      Retrieves the value for key.
      Outputs raw value on stdout (exit 0) or 'NOTFOUND' (exit 2).
      Example: %s get ./data.kip rice

  del   <db-path> <key>
      Durably marks key as deleted (tombstone). Idempotent on missing keys.
      Outputs 'ACK <key>' on confirmed persistence.
      Example: %s del ./data.kip rice

  batch <db-path> <file>
      Atomically executes up to 10,000 operations from a file.
      Accepts either JSON array format or line-based format ("put k v" / "del k").
      Outputs 'ACK <count>' on confirmed atomic commit.
      Example: %s batch ./data.kip ./ops.json

  check <db-path>
      Scans and validates header, payload boundaries, commit footers, and dual CRCs.
      Outputs 'OK: Database integrity verified' or reports corruption.
      Example: %s check ./data.kip

  dump  <db-path>
      Scans and prints all live key-value pairs, sorted lexicographically by key.
      Outputs '<key> = <value>' lines, or '(empty database)'.
      Example: %s dump ./data.kip

  stats <db-path>
      Displays live database statistics (keys, size, capacity, last tx ID).
      Example: %s stats ./data.kip

  compact <db-path>
      Reclaims dead tombstone and update space via crash-safe atomic file replacement.
      Example: %s compact ./data.kip

  serve <db-path> [--max-bytes N]
      Interactive IPC server mode reading commands over stdin and writing to stdout.
      Designed for crash-torture harnesses and automated fault injectors.
      Example: %s serve ./data.kip

  help, --help
      Displays this reference manual.

  version, --version
      Displays executable version information.

EXIT CODES:
  0    Success
  1    General error (syntax, capacity full, corrupt file, engine error)
  2    Key not found (for get command)
`, AppName, AppDesc, AppName, AppName, AppName, AppName, AppName, AppName, AppName, AppName, AppName, AppName, AppName)
}
