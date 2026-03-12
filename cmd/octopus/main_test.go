package main

import (
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jc/octopus/internal/storage/sqlite"
	_ "modernc.org/sqlite"
)

func TestCLIJobRunHappyPath(t *testing.T) {
	tmp := t.TempDir()
	repoRoot := mustRepoRoot(t)
	dbPath := filepath.Join(tmp, "octopus.db")
	prepareDB(t, dbPath)
	seedSQL(t, dbPath, `INSERT INTO jobs (id, name, tool, prompt_template, working_dir, timeout_seconds, enabled)
		VALUES ('job_ok', 'ok', 'codex-cli', 'hello', '', 30, 1)`) // empty working_dir keeps allow-list checks simple.

	binDir := setupFakeCodex(t, tmp)
	cmd := exec.Command("go", "run", "./cmd/octopus", "job", "run", "job_ok")
	cmd.Dir = repoRoot
	cmd.Env = testEnv(t, binDir, dbPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %v\noutput:\n%s", err, out)
	}
	text := string(out)
	if !strings.Contains(text, "status: succeeded") {
		t.Fatalf("output missing success status:\n%s", text)
	}
	if !strings.Contains(text, "exit_code: 0") {
		t.Fatalf("output missing zero exit code:\n%s", text)
	}
}

func TestCLIJobRunFailurePath(t *testing.T) {
	tmp := t.TempDir()
	repoRoot := mustRepoRoot(t)
	dbPath := filepath.Join(tmp, "octopus.db")
	prepareDB(t, dbPath)
	seedSQL(t, dbPath, `INSERT INTO jobs (id, name, tool, prompt_template, working_dir, timeout_seconds, enabled)
		VALUES ('job_fail', 'fail', 'codex-cli', 'FAIL please', '', 30, 1)`)

	binDir := setupFakeCodex(t, tmp)
	cmd := exec.Command("go", "run", "./cmd/octopus", "job", "run", "job_fail")
	cmd.Dir = repoRoot
	cmd.Env = testEnv(t, binDir, dbPath)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected command to fail\noutput:\n%s", out)
	}
	text := string(out)
	if !strings.Contains(text, "run error: agent exited with code 7") {
		t.Fatalf("output missing expected failure:\n%s", text)
	}
}

func TestCLISchedulerTickHappyPath(t *testing.T) {
	tmp := t.TempDir()
	repoRoot := mustRepoRoot(t)
	dbPath := filepath.Join(tmp, "octopus.db")
	prepareDB(t, dbPath)

	now := time.Now().UTC()
	due := now.Add(-1 * time.Minute).Format(time.RFC3339Nano)
	seedSQL(t, dbPath,
		`INSERT INTO jobs (id, name, tool, prompt_template, working_dir, timeout_seconds, enabled)
		 VALUES ('job_due', 'due-job', 'codex-cli', 'hello', '', 30, 1)`,
		`INSERT INTO schedules (id, job_id, cron_expr, timezone, next_run_at, enabled)
		 VALUES ('sched_1', 'job_due', '* * * * *', 'UTC', '`+due+`', 1)`,
	)

	binDir := setupFakeCodex(t, tmp)
	cmd := exec.Command("go", "run", "./cmd/octopus", "scheduler", "tick", "--limit", "10", "--concurrency", "1", "--retries", "0")
	cmd.Dir = repoRoot
	cmd.Env = testEnv(t, binDir, dbPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %v\noutput:\n%s", err, out)
	}
	text := string(out)
	if !strings.Contains(text, "due: 1") || !strings.Contains(text, "executed: 1") || !strings.Contains(text, "failed: 0") {
		t.Fatalf("unexpected scheduler output:\n%s", text)
	}
}

func TestCLISchedulerTickFailureCountPath(t *testing.T) {
	tmp := t.TempDir()
	repoRoot := mustRepoRoot(t)
	dbPath := filepath.Join(tmp, "octopus.db")
	prepareDB(t, dbPath)

	now := time.Now().UTC()
	due := now.Add(-1 * time.Minute).Format(time.RFC3339Nano)
	seedSQL(t, dbPath,
		`INSERT INTO jobs (id, name, tool, prompt_template, working_dir, timeout_seconds, enabled)
		 VALUES ('job_due_fail', 'due-fail', 'codex-cli', 'FAIL please', '', 30, 1)`,
		`INSERT INTO schedules (id, job_id, cron_expr, timezone, next_run_at, enabled)
		 VALUES ('sched_fail', 'job_due_fail', '* * * * *', 'UTC', '`+due+`', 1)`,
	)

	binDir := setupFakeCodex(t, tmp)
	cmd := exec.Command("go", "run", "./cmd/octopus", "scheduler", "tick", "--limit", "10", "--concurrency", "1", "--retries", "0")
	cmd.Dir = repoRoot
	cmd.Env = testEnv(t, binDir, dbPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed unexpectedly: %v\noutput:\n%s", err, out)
	}
	text := string(out)
	if !strings.Contains(text, "due: 1") || !strings.Contains(text, "executed: 1") || !strings.Contains(text, "failed: 1") {
		t.Fatalf("unexpected scheduler output:\n%s", text)
	}
}

func testEnv(t *testing.T, binDir string, dbPath string) []string {
	t.Helper()
	path := os.Getenv("PATH")
	if path == "" {
		t.Fatal("PATH is empty")
	}
	return append(os.Environ(),
		"PATH="+binDir+string(os.PathListSeparator)+path,
		"OCTOPUS_DB_PATH="+dbPath,
	)
}

func setupFakeCodex(t *testing.T, root string) string {
	t.Helper()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("create bin dir: %v", err)
	}
	binPath := filepath.Join(binDir, "codex")
	script := "#!/bin/sh\n" +
		"set -eu\n" +
		"if [ \"${1:-}\" = \"exec\" ]; then\n" +
		"  prompt=\"${2:-}\"\n" +
		"  case \"$prompt\" in\n" +
		"    *FAIL*) echo forced failure 1>&2; exit 7 ;;\n" +
		"  esac\n" +
		"  echo codex-ok\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"${1:-}\" = \"--help\" ]; then\n" +
		"  echo codex-help\n" +
		"  exit 0\n" +
		"fi\n" +
		"echo unsupported 1>&2\n" +
		"exit 9\n"
	if err := os.WriteFile(binPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake codex binary: %v", err)
	}
	return binDir
}

func prepareDB(t *testing.T, path string) {
	t.Helper()
	store, err := sqlite.Open(path)
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close sqlite store: %v", err)
	}
}

func seedSQL(t *testing.T, dbPath string, statements ...string) {
	t.Helper()
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	defer db.Close()

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("exec seed statement %q: %v", stmt, err)
		}
	}
}

func mustRepoRoot(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("unable to locate repository root from %s", cwd)
		}
		dir = parent
	}
}
