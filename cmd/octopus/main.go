package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jc/octopus/internal/core/execution"
	"github.com/jc/octopus/internal/scheduler"
	"github.com/jc/octopus/internal/storage/sqlite"
)

const defaultDBPath = ".octopus/octopus.db"

// CLI usage quick reference:
//   octopus agent validate [--tool codex-cli|claude-code]
//     --tool: optional, validates only one tool when set.
//   octopus job run <job-id|name>
//     <job-id|name>: required, runs a specific job by ID or name.
//   octopus scheduler tick [--limit 100] [--concurrency 2] [--retries 1]
//     --limit: max schedules handled per tick (default 100).
//     --concurrency: max concurrent runs per tick (default 2).
//     --retries: retry attempts per failed run (default 1).
// Environment:
//   OCTOPUS_DB_PATH: optional SQLite path (default .octopus/octopus.db).
func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	ctx := context.Background()

	switch os.Args[1] {
	case "agent":
		handleAgent(ctx, os.Args[2:])
	case "job":
		handleJob(ctx, os.Args[2:])
	case "scheduler":
		handleScheduler(ctx, os.Args[2:])
	default:
		printUsage()
		os.Exit(2)
	}
}

func handleAgent(ctx context.Context, args []string) {
	if len(args) < 1 || args[0] != "validate" {
		printAgentUsage()
		os.Exit(2)
	}

	validateFS := flag.NewFlagSet("agent validate", flag.ExitOnError)
	tool := validateFS.String("tool", "", "specific tool to validate")
	_ = validateFS.Parse(args[1:])

	svc, err := execution.NewService(nil, nil)
	if err != nil {
		fatalf("initialize execution service: %v", err)
	}

	if err := svc.Validate(ctx, strings.TrimSpace(*tool)); err != nil {
		fatalf("validation error: %v", err)
	}

	if strings.TrimSpace(*tool) == "" {
		fmt.Println("ok: codex-cli and claude-code are available")
		return
	}
	fmt.Printf("ok: %s is available\n", *tool)
}

func handleJob(ctx context.Context, args []string) {
	if len(args) < 2 || args[0] != "run" {
		printJobUsage()
		os.Exit(2)
	}

	store := mustOpenStore()
	defer func() {
		_ = store.Close()
	}()

	svc, err := execution.NewService(store.Jobs(), store.Runs())
	if err != nil {
		fatalf("initialize execution service: %v", err)
	}

	run, err := svc.RunJob(ctx, args[1])
	if err != nil {
		fatalf("run error: %v", err)
	}

	exitCode := 0
	if run.ExitCode != nil {
		exitCode = *run.ExitCode
	}

	fmt.Printf("run_id: %s\n", run.ID)
	fmt.Printf("status: %s\n", run.Status)
	fmt.Printf("exit_code: %d\n", exitCode)
	fmt.Printf("duration: %s\n", run.Duration.Round(time.Millisecond))
	fmt.Println("--- stdout ---")
	fmt.Print(strings.TrimSpace(run.Stdout) + "\n")
	fmt.Println("--- stderr ---")
	fmt.Print(strings.TrimSpace(run.Stderr) + "\n")
}

func handleScheduler(ctx context.Context, args []string) {
	if len(args) < 1 || args[0] != "tick" {
		printSchedulerUsage()
		os.Exit(2)
	}

	tickFS := flag.NewFlagSet("scheduler tick", flag.ExitOnError)
	limit := tickFS.Int("limit", 100, "max schedules handled per tick")
	concurrency := tickFS.Int("concurrency", 2, "max concurrent runs per tick")
	retries := tickFS.Int("retries", 1, "retry attempts per failed run")
	_ = tickFS.Parse(args[1:])

	store := mustOpenStore()
	defer func() {
		_ = store.Close()
	}()

	execSvc, err := execution.NewService(store.Jobs(), store.Runs())
	if err != nil {
		fatalf("initialize execution service: %v", err)
	}

	schedulerSvc := scheduler.NewService(store.Schedules(), execSvc, *concurrency, *retries)
	result, err := schedulerSvc.Tick(ctx, *limit)
	if err != nil {
		fatalf("scheduler tick error: %v", err)
	}

	fmt.Printf("due: %d\n", result.Due)
	fmt.Printf("executed: %d\n", result.Executed)
	fmt.Printf("failed: %d\n", result.Failed)
}

func mustOpenStore() *sqlite.Store {
	dbPath := strings.TrimSpace(os.Getenv("OCTOPUS_DB_PATH"))
	if dbPath == "" {
		dbPath = defaultDBPath
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		fatalf("create db directory: %v", err)
	}

	store, err := sqlite.Open(dbPath)
	if err != nil {
		fatalf("open store: %v", err)
	}
	return store
}

func fatalf(msg string, args ...any) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	os.Exit(1)
}

func printUsage() {
	fmt.Println("octopus <command>")
	fmt.Println("commands:")
	fmt.Println("  agent validate [--tool codex-cli|claude-code]")
	fmt.Println("  job run <job-id|name>")
	fmt.Println("  scheduler tick [--limit 100] [--concurrency 2] [--retries 1]")
}

func printAgentUsage() {
	fmt.Println("octopus agent validate [--tool codex-cli|claude-code]")
}

func printJobUsage() {
	fmt.Println("octopus job run <job-id|name>")
}

func printSchedulerUsage() {
	fmt.Println("octopus scheduler tick [--limit 100] [--concurrency 2] [--retries 1]")
}
