package agent

import (
	"context"
	"time"
)

// Adapter defines a normalized interface for external AI CLI integrations.
type Adapter interface {
	Name() string
	Validate(ctx context.Context) error
	Run(ctx context.Context, input RunInput) (RunResult, error)
}

// RunInput is the common execution contract across agent CLIs.
type RunInput struct {
	Prompt     string
	WorkingDir string
	Timeout    time.Duration
	Env        map[string]string
}

// RunResult normalizes output across tools.
type RunResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Duration time.Duration
}
