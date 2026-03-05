package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jc/octopus/internal/spike"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	svc := spike.NewService()
	ctx := context.Background()

	switch os.Args[1] {
	case "spike":
		handleSpike(ctx, svc, os.Args[2:])
	default:
		printUsage()
		os.Exit(2)
	}
}

func handleSpike(ctx context.Context, svc *spike.Service, args []string) {
	if len(args) < 1 {
		printSpikeUsage()
		os.Exit(2)
	}

	switch args[0] {
	case "validate":
		if err := svc.Validate(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "validation error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("ok: codex-cli and claude-code are available")
	case "run":
		runFS := flag.NewFlagSet("spike run", flag.ExitOnError)
		tool := runFS.String("tool", "", "agent tool: codex-cli|claude-code")
		prompt := runFS.String("prompt", "", "prompt text")
		workingDir := runFS.String("workdir", "", "working directory")
		timeout := runFS.Duration("timeout", 2*time.Minute, "execution timeout, e.g. 30s, 2m")

		_ = runFS.Parse(args[1:])

		if *tool == "" || *prompt == "" {
			fmt.Fprintln(os.Stderr, "--tool and --prompt are required")
			runFS.Usage()
			os.Exit(2)
		}

		result, err := svc.Run(ctx, *tool, *prompt, *workingDir, *timeout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "run error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("exit_code: %d\n", result.ExitCode)
		fmt.Printf("duration: %s\n", result.Duration.Round(time.Millisecond))
		fmt.Println("--- stdout ---")
		fmt.Print(strings.TrimSpace(result.Stdout) + "\n")
		fmt.Println("--- stderr ---")
		fmt.Print(strings.TrimSpace(result.Stderr) + "\n")
	default:
		printSpikeUsage()
		os.Exit(2)
	}
}

func printUsage() {
	fmt.Println("octopus <command>")
	fmt.Println("commands:")
	fmt.Println("  spike validate")
	fmt.Println("  spike run --tool codex-cli|claude-code --prompt \"...\" [--workdir PATH] [--timeout 2m]")
}

func printSpikeUsage() {
	fmt.Println("octopus spike <command>")
	fmt.Println("commands:")
	fmt.Println("  validate")
	fmt.Println("  run")
}
