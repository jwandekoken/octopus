package execution

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jc/octopus/internal/adapters/claudecode"
	"github.com/jc/octopus/internal/adapters/codexcli"
	"github.com/jc/octopus/internal/core/agent"
	"github.com/jc/octopus/internal/core/jobs"
	"github.com/jc/octopus/internal/core/runs"
)

type Service struct {
	adapters       map[string]agent.Adapter
	jobs           *jobs.Service
	runs           runs.Repository
	defaultTimeout time.Duration
	allowedRoots   []string
	now            func() time.Time
}

type Option func(*Service) error

func WithAdapters(adapters map[string]agent.Adapter) Option {
	return func(s *Service) error {
		if len(adapters) == 0 {
			return errors.New("adapters must not be empty")
		}
		s.adapters = adapters
		return nil
	}
}

func WithDefaultTimeout(timeout time.Duration) Option {
	return func(s *Service) error {
		if timeout <= 0 {
			return errors.New("default timeout must be > 0")
		}
		s.defaultTimeout = timeout
		return nil
	}
}

func WithAllowedWorkdirRoots(roots []string) Option {
	return func(s *Service) error {
		resolved := make([]string, 0, len(roots))
		for _, root := range roots {
			if strings.TrimSpace(root) == "" {
				continue
			}
			abs, err := filepath.Abs(root)
			if err != nil {
				return fmt.Errorf("resolve root %q: %w", root, err)
			}
			resolved = append(resolved, filepath.Clean(abs))
		}
		s.allowedRoots = resolved
		return nil
	}
}

func WithClock(now func() time.Time) Option {
	return func(s *Service) error {
		if now == nil {
			return errors.New("clock must not be nil")
		}
		s.now = now
		return nil
	}
}

func NewService(jobRepo jobs.Repository, runRepo runs.Repository, options ...Option) (*Service, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("read working directory: %w", err)
	}

	svc := &Service{
		adapters: map[string]agent.Adapter{
			"codex-cli":   codexcli.New(""),
			"claude-code": claudecode.New(""),
		},
		runs:           runRepo,
		defaultTimeout: 2 * time.Minute,
		allowedRoots:   []string{filepath.Clean(cwd)},
		now:            time.Now,
	}
	if jobRepo != nil {
		svc.jobs = jobs.NewService(jobRepo)
	}

	for _, opt := range options {
		if err := opt(svc); err != nil {
			return nil, err
		}
	}

	return svc, nil
}

func (s *Service) Validate(ctx context.Context, tool string) error {
	if strings.TrimSpace(tool) == "" {
		for name, adapter := range s.adapters {
			if err := adapter.Validate(ctx); err != nil {
				return fmt.Errorf("%s validation failed: %w", name, err)
			}
		}
		return nil
	}

	adapter, ok := s.adapters[tool]
	if !ok {
		return fmt.Errorf("unknown tool %q", tool)
	}
	if err := adapter.Validate(ctx); err != nil {
		return fmt.Errorf("%s validation failed: %w", tool, err)
	}
	return nil
}

func (s *Service) Run(ctx context.Context, tool, prompt, workingDir string, timeout time.Duration) (agent.RunResult, error) {
	adapter, ok := s.adapters[tool]
	if !ok {
		return agent.RunResult{}, fmt.Errorf("unknown tool %q", tool)
	}
	if strings.TrimSpace(prompt) == "" {
		return agent.RunResult{}, errors.New("prompt is required")
	}
	if err := s.validateWorkingDir(workingDir); err != nil {
		return agent.RunResult{}, err
	}
	if timeout <= 0 {
		timeout = s.defaultTimeout
	}

	return adapter.Run(ctx, agent.RunInput{
		Prompt:     prompt,
		WorkingDir: workingDir,
		Timeout:    timeout,
	})
}

func (s *Service) RunJob(ctx context.Context, idOrName string) (runs.Run, error) {
	if s.jobs == nil {
		return runs.Run{}, errors.New("job repository is not configured")
	}
	if s.runs == nil {
		return runs.Run{}, errors.New("runs repository is not configured")
	}

	job, err := s.jobs.Resolve(ctx, idOrName)
	if err != nil {
		return runs.Run{}, err
	}

	now := s.now().UTC()
	run, err := s.runs.Create(ctx, runs.Run{
		JobID:     job.ID,
		Status:    runs.StatusQueued,
		CreatedAt: now,
	})
	if err != nil {
		return runs.Run{}, fmt.Errorf("create run: %w", err)
	}

	started := s.now().UTC()
	run.Status = runs.StatusRunning
	run.StartedAt = &started
	if err := s.runs.Update(ctx, run); err != nil {
		return runs.Run{}, fmt.Errorf("mark run running: %w", err)
	}

	result, runErr := s.executeJob(ctx, job)
	finished := s.now().UTC()
	run.FinishedAt = &finished
	run.Stdout = result.Stdout
	run.Stderr = result.Stderr
	run.Duration = result.Duration

	if runErr != nil {
		run.Error = runErr.Error()
		if errors.Is(runErr, context.DeadlineExceeded) {
			run.Status = runs.StatusTimedOut
		} else {
			run.Status = runs.StatusFailed
		}
	} else {
		run.Status = runs.StatusSucceeded
	}

	exitCode := result.ExitCode
	run.ExitCode = &exitCode

	if err := s.runs.Update(ctx, run); err != nil {
		return runs.Run{}, fmt.Errorf("finalize run: %w", err)
	}
	if runErr != nil {
		return run, runErr
	}

	return run, nil
}

func (s *Service) executeJob(ctx context.Context, job jobs.Job) (agent.RunResult, error) {
	adapter, ok := s.adapters[job.Tool]
	if !ok {
		return agent.RunResult{}, fmt.Errorf("unknown tool %q", job.Tool)
	}
	if err := s.validateWorkingDir(job.WorkingDir); err != nil {
		return agent.RunResult{}, err
	}
	timeout := job.Timeout
	if timeout <= 0 {
		timeout = s.defaultTimeout
	}

	result, err := adapter.Run(ctx, agent.RunInput{
		Prompt:     job.PromptTemplate,
		WorkingDir: job.WorkingDir,
		Timeout:    timeout,
	})
	if err != nil {
		return result, err
	}
	if result.ExitCode != 0 {
		return result, fmt.Errorf("agent exited with code %d", result.ExitCode)
	}
	return result, nil
}

func (s *Service) validateWorkingDir(workingDir string) error {
	if strings.TrimSpace(workingDir) == "" {
		return nil
	}
	abs, err := filepath.Abs(workingDir)
	if err != nil {
		return fmt.Errorf("resolve working directory: %w", err)
	}
	candidate := filepath.Clean(abs)

	for _, root := range s.allowedRoots {
		rel, err := filepath.Rel(root, candidate)
		if err != nil {
			continue
		}
		if rel == "." || (!strings.HasPrefix(rel, "..") && rel != "") {
			return nil
		}
	}

	return fmt.Errorf("working directory %q is outside allowed roots", candidate)
}
