package agent

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"sort"
	"strings"
	"syscall"
	"time"
)

// ExecCommand runs a process with optional timeout and normalized env handling.
func ExecCommand(ctx context.Context, cmdName string, args []string, input RunInput) (RunResult, error) {
	result := RunResult{}

	runCtx := ctx
	cancel := func() {}
	if input.Timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, input.Timeout)
	}
	defer cancel()

	cmd := exec.CommandContext(runCtx, cmdName, args...)
	if input.WorkingDir != "" {
		cmd.Dir = input.WorkingDir
	}

	if len(input.Env) > 0 {
		cmd.Env = mergeEnv(input.Env)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	startedAt := time.Now()
	err := cmd.Run()
	result.Duration = time.Since(startedAt)
	result.Stdout = stdout.String()
	result.Stderr = stderr.String()

	if err == nil {
		return result, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			result.ExitCode = status.ExitStatus()
		}
		return result, nil
	}

	return result, err
}

func mergeEnv(extra map[string]string) []string {
	env := os.Environ()
	keys := make([]string, 0, len(extra))
	for k := range extra {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		env = append(env, k+"="+extra[k])
	}
	return env
}

func HelpSanityCheck(ctx context.Context, bin string) error {
	cmd := exec.CommandContext(ctx, bin, "--help")
	cmd.Env = os.Environ()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return err
	}
	combined := strings.ToLower(string(out) + "\n" + stderr.String())
	if len(strings.TrimSpace(combined)) == 0 {
		return errors.New("help output is empty")
	}
	return nil
}
