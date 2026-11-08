package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/madhuharshag/Fsync-or-Die/src/kv"
)

// Exit codes conforming to the Fsync-or-Die CLI specification:
// 0: Success
// 1: General error (invalid args, engine error, corrupt file, full disk)
// 2: Key not found (specific to get command)
const (
	ExitSuccess  = 0
	ExitError    = 1
	ExitNotFound = 2
)

// ErrUsage is returned when command-line arguments are missing or invalid.
var ErrUsage = errors.New("invalid usage")

func main() {
	exitCode := run(os.Args[1:])
	os.Exit(exitCode)
}

func run(args []string) int {
	if len(args) == 0 {
		printUsage(os.Stderr)
		return ExitError
	}

	// Support worker flag for direct torture-harness child process execution
	if args[0] == "--worker" || (os.Getenv("TORTURE_WORKER") == "1" && (len(args) == 0 || args[0] == "--worker")) {
		return runServeMode("", 0, false)
	}

	cmd := strings.ToLower(args[0])

	switch cmd {
	case "help", "-h", "--help", "-help":
		printUsage(os.Stdout)
		return ExitSuccess

	case "version", "-v", "--version":
		printVersion(os.Stdout)
		return ExitSuccess

	case "open":
		return cmdOpen(args[1:])

	case "put":
		return cmdPut(args[1:])

	case "get":
		return cmdGet(args[1:])

	case "del", "delete":
		return cmdDel(args[1:])

	case "batch":
		return cmdBatch(args[1:])

	case "check":
		return cmdCheck(args[1:])

	case "dump", "scan":
		return cmdDump(args[1:])

	case "stats":
		return cmdStats(args[1:])

	case "compact":
		return cmdCompact(args[1:])

	case "serve":
		return cmdServe(args[1:])

	default:
		fmt.Fprintf(os.Stderr, "error: unknown command '%s'\n\n", args[0])
		printUsage(os.Stderr)
		return ExitError
	}
}

// printError formats errors consistently with the required "error: <msg>" prefix.
func printError(err error) {
	if err == nil {
		return
	}
	if errors.Is(err, kv.ErrUncertain) {
		fmt.Fprintf(os.Stderr, "error: transaction status uncertain (write may or may not have reached disk): %v\n", err)
		return
	}
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
}
