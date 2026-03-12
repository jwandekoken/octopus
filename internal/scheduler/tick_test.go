package scheduler

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jc/octopus/internal/core/runs"
	"github.com/jc/octopus/internal/core/schedules"
)

type fakeScheduleRepo struct {
	due      []schedules.Schedule
	advanced map[string]time.Time
	mu       sync.Mutex
}

func (r *fakeScheduleRepo) ListDue(ctx context.Context, now time.Time, limit int) ([]schedules.Schedule, error) {
	if len(r.due) > limit {
		return r.due[:limit], nil
	}
	return r.due, nil
}

func (r *fakeScheduleRepo) Advance(ctx context.Context, scheduleID string, previousNextRunAt time.Time, nextRunAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.advanced == nil {
		r.advanced = map[string]time.Time{}
	}
	r.advanced[scheduleID] = nextRunAt
	return nil
}

type fakeExecution struct {
	attempts int
	failFor  int
}

func (e *fakeExecution) RunJob(ctx context.Context, idOrName string) (runs.Run, error) {
	e.attempts++
	if e.attempts <= e.failFor {
		return runs.Run{}, errors.New("boom")
	}
	return runs.Run{ID: "r1", Status: runs.StatusSucceeded}, nil
}

func TestTickRetriesAndAdvances(t *testing.T) {
	now := time.Date(2026, 3, 12, 12, 5, 0, 0, time.UTC)
	repo := &fakeScheduleRepo{due: []schedules.Schedule{{
		ID:        "s1",
		JobID:     "j1",
		CronExpr:  "*/5 * * * *",
		Timezone:  "UTC",
		NextRunAt: now.Add(-5 * time.Minute),
		Enabled:   true,
	}}}
	exec := &fakeExecution{failFor: 1}

	svc := NewService(repo, exec, 1, 1)
	svc.now = func() time.Time { return now }

	result, err := svc.Tick(context.Background(), 10)
	if err != nil {
		t.Fatalf("Tick error: %v", err)
	}
	if result.Due != 1 || result.Executed != 1 || result.Failed != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if exec.attempts != 2 {
		t.Fatalf("attempts = %d, want 2", exec.attempts)
	}
	if repo.advanced["s1"].IsZero() {
		t.Fatal("expected schedule advancement")
	}
}
