package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/jc/octopus/internal/core/runs"
)

func TestRunsRepositoryCreateUpdate(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "octopus.db")
	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open error: %v", err)
	}
	defer func() {
		_ = store.Close()
	}()

	repo := store.Runs()
	ctx := context.Background()

	created, err := repo.Create(ctx, runs.Run{
		JobID:     "job_1",
		Status:    runs.StatusQueued,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected run id")
	}

	now := time.Now().UTC()
	exitCode := 0
	created.Status = runs.StatusSucceeded
	created.StartedAt = &now
	created.FinishedAt = &now
	created.ExitCode = &exitCode
	created.Duration = 1500 * time.Millisecond
	created.Stdout = "ok"
	created.Stderr = ""

	if err := repo.Update(ctx, created); err != nil {
		t.Fatalf("Update error: %v", err)
	}

	persisted, err := repo.getByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("getByID error: %v", err)
	}
	if persisted.Status != runs.StatusSucceeded {
		t.Fatalf("status = %s, want %s", persisted.Status, runs.StatusSucceeded)
	}
	if persisted.ExitCode == nil || *persisted.ExitCode != 0 {
		t.Fatalf("exit code = %v, want 0", persisted.ExitCode)
	}
	if persisted.Duration != 1500*time.Millisecond {
		t.Fatalf("duration = %s, want 1.5s", persisted.Duration)
	}
}
