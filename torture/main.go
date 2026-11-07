package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func main() {
	workerFlag := flag.Bool("worker", false, "Run in worker subprocess mode")
	modeFlag := flag.String("mode", "random", "Torture mode: random|large_values|large_batches|compaction_crash|recovery_crash|disk_full")
	seedFlag := flag.Int64("seed", 0, "Random seed (0 = auto-generate)")
	opsFlag := flag.Int("ops", 500, "Total number of operations to execute")
	crashesFlag := flag.Int("crashes", 5, "Number of forced crash/restart cycles")
	writersFlag := flag.Int("writers", 2, "Number of concurrent writer identities")
	readersFlag := flag.Int("readers", 1, "Number of concurrent reader identities")
	dbFlag := flag.String("db", "", "Path to database file (default = temp file)")
	journalFlag := flag.String("journal", "", "Path to oracle journal (default = temp file)")
	deterministicFlag := flag.String("deterministic", "none", "Deterministic crash point: none|writing_tmp|after_sync|after_rename")
	maxBytesFlag := flag.Uint64("maxbytes", 0, "Max database file bytes (0 = unlimited)")
	campaignFlag := flag.Bool("campaign", false, "Run full 1,000-case verification campaign and generate artifacts")

	flag.Parse()

	// Check if full 1,000-case campaign requested
	if *campaignFlag {
		cr, err := NewCampaignRunner()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to initialize campaign runner: %v\n", err)
			os.Exit(1)
		}
		summary, err := cr.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "campaign run failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\n[SUCCESS] 1,000-Case Campaign Completed: %d/%d passed (%s)\n", summary.PassedCases, summary.TotalCases, summary.PassRate)
		return
	}

	// Check if invoked as a worker child process
	if *workerFlag || os.Getenv("TORTURE_WORKER") == "1" {
		runWorkerProcess()
		return
	}

	// Harness runner mode
	seed := *seedFlag
	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	dbPath := *dbFlag
	tempDir := ""
	if dbPath == "" {
		var err error
		tempDir, err = os.MkdirTemp("", "torture_run_*")
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
			os.Exit(1)
		}
		dbPath = filepath.Join(tempDir, "torture.db")
	}

	var runPassed bool
	defer func() {
		if tempDir != "" {
			if runPassed {
				_ = os.RemoveAll(tempDir)
			} else {
				absPath, _ := filepath.Abs(tempDir)
				fmt.Fprintf(os.Stderr, "\n[PRESERVED] Failing workspace preserved for debugging at: %s\n", absPath)
			}
		}
	}()

	journalPath := *journalFlag
	if journalPath == "" && tempDir != "" {
		journalPath = filepath.Join(tempDir, "oracle.log")
	}

	cfg := HarnessConfig{
		Mode:               *modeFlag,
		Seed:               seed,
		TotalOps:           *opsFlag,
		Crashes:            *crashesFlag,
		Writers:            *writersFlag,
		Readers:            *readersFlag,
		DbPath:             dbPath,
		OracleLogPath:      journalPath,
		DeterministicCrash: *deterministicFlag,
		MaxFileBytes:       *maxBytesFlag,
	}

	oracle, err := NewOracle(journalPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize oracle: %v\n", err)
		os.Exit(1)
	}
	defer oracle.Close()

	runner := NewHarnessRunner(cfg, oracle)

	fmt.Println("================================================================")
	fmt.Println(" FSYNC-OR-DIE — INDEPENDENT TORTURE HARNESS")
	fmt.Println("================================================================")
	fmt.Printf("Mode:              %s\n", cfg.Mode)
	fmt.Printf("Seed:              %d\n", cfg.Seed)
	fmt.Printf("Target Ops:        %d\n", cfg.TotalOps)
	fmt.Printf("Planned Crashes:   %d\n", cfg.Crashes)
	fmt.Printf("Writers:           %d\n", cfg.Writers)
	fmt.Printf("Database Path:     %s\n", cfg.DbPath)
	fmt.Printf("Oracle Log:        %s\n", cfg.OracleLogPath)
	fmt.Println("----------------------------------------------------------------")

	report, err := runner.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n[FAIL] TORTURE TEST FAILED: %v\n", err)
		if report != nil {
			printReport(report)
		}
		os.Exit(1)
	}

	if report != nil && report.Passed {
		runPassed = true
	}

	fmt.Println("\n[PASS] TORTURE TEST COMPLETED SUCCESSFULLY")
	printReport(report)
}

func printReport(r *HarnessReport) {
	fmt.Println("================================================================")
	fmt.Println(" TORTURE TEST AUDIT REPORT")
	fmt.Println("================================================================")
	fmt.Printf("Mode:               %s\n", r.Mode)
	fmt.Printf("Status:             %v\n", r.Passed)
	fmt.Printf("Verification Msg:   %s\n", r.VerificationMsg)
	fmt.Printf("Seed:               %d (use --seed %d to reproduce)\n", r.Seed, r.Seed)
	fmt.Printf("Go Version:         %s\n", r.GoVersion)
	fmt.Printf("Host OS/Arch:       %s\n", r.OSArch)
	fmt.Printf("Duration:           %v\n", r.Duration)
	fmt.Printf("Operations Sent:    %d\n", r.OpsAttempted)
	fmt.Printf("Ops Acknowledged:   %d\n", r.OpsAcknowledged)
	fmt.Printf("Ops Indeterminate:  %d\n", r.OpsIndeterminate)
	fmt.Printf("Ops Rejected:       %d\n", r.OpsRejected)
	fmt.Printf("Forced Crashes:     %d\n", r.Crashes)
	fmt.Printf("Successful Reopen:  %d\n", r.Recoveries)
	fmt.Println("================================================================")
}
