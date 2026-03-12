package runs

import (
	"context"
	"errors"
	"time"
)

const (
	StatusQueued    = "queued"
	StatusRunning   = "running"
	StatusSucceeded = "succeeded"
	StatusFailed    = "failed"
	StatusTimedOut  = "timed_out"
)

var ErrNotFound = errors.New("run not found")

type Run struct {
	ID         string
	JobID      string
	Status     string
	CreatedAt  time.Time
	StartedAt  *time.Time
	FinishedAt *time.Time
	ExitCode   *int
	Duration   time.Duration
	Stdout     string
	Stderr     string
	Error      string
}

type Repository interface {
	Create(ctx context.Context, run Run) (Run, error)
	Update(ctx context.Context, run Run) error
}
