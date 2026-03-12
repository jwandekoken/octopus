package claudecode

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jc/octopus/internal/core/agent"
)

func TestAdapterRunInvocationContract(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "invocation.log")
	binPath := filepath.Join(tmp, "fake-claude")
	script := "#!/bin/sh\n" +
		"set -eu\n" +
		"if [ \"${1:-}\" = \"--help\" ]; then\n" +
		"  echo claude-help\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"${1:-}\" = \"--print\" ]; then\n" +
		"  printf 'cmd=%s\\nprompt=%s\\npwd=%s\\nflag=%s\\n' \"$1\" \"$2\" \"$PWD\" \"${FAKE_FLAG:-}\" > \"$OCTOPUS_LOG\"\n" +
		"  echo run-ok\n" +
		"  exit 0\n" +
		"fi\n" +
		"exit 9\n"
	if err := os.WriteFile(binPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}

	adapter := New(binPath)
	workingDir := tmp
	result, err := adapter.Run(context.Background(), agent.RunInput{
		Prompt:     "hello",
		WorkingDir: workingDir,
		Timeout:    2 * time.Second,
		Env: map[string]string{
			"FAKE_FLAG":   "yes",
			"OCTOPUS_LOG": logPath,
		},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("exit code = %d, want 0", result.ExitCode)
	}
	if !strings.Contains(result.Stdout, "run-ok") {
		t.Fatalf("stdout = %q, want run output", result.Stdout)
	}

	invocation, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read invocation log: %v", err)
	}
	text := string(invocation)
	if !strings.Contains(text, "cmd=--print") {
		t.Fatalf("invocation log missing command: %q", text)
	}
	if !strings.Contains(text, "prompt=hello") {
		t.Fatalf("invocation log missing prompt: %q", text)
	}
	if !strings.Contains(text, "pwd="+workingDir) {
		t.Fatalf("invocation log missing working dir: %q", text)
	}
	if !strings.Contains(text, "flag=yes") {
		t.Fatalf("invocation log missing env propagation: %q", text)
	}
}

func TestAdapterValidateInvocationContract(t *testing.T) {
	tmp := t.TempDir()
	binPath := filepath.Join(tmp, "fake-claude")
	script := "#!/bin/sh\n" +
		"set -eu\n" +
		"if [ \"${1:-}\" = \"--help\" ]; then\n" +
		"  echo claude-help\n" +
		"  exit 0\n" +
		"fi\n" +
		"exit 11\n"
	if err := os.WriteFile(binPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}

	adapter := New(binPath)
	if err := adapter.Validate(context.Background()); err != nil {
		t.Fatalf("Validate error: %v", err)
	}
}
