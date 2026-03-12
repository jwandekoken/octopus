package execution

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jc/octopus/internal/core/agent"
	"github.com/jc/octopus/internal/core/jobs"
	"github.com/jc/octopus/internal/core/runs"
)

type fakeJobRepo struct {
	job jobs.Job
}

func (r *fakeJobRepo) GetByID(ctx context.Context, id string) (jobs.Job, error) {
	if r.job.ID == id {
		return r.job, nil
	}
	return jobs.Job{}, jobs.ErrNotFound
}

func (r *fakeJobRepo) GetByName(ctx context.Context, name string) (jobs.Job, error) {
	if r.job.Name == name {
		return r.job, nil
	}
	return jobs.Job{}, jobs.ErrNotFound
}

type fakeRunsRepo struct {
	created runs.Run
	latest  runs.Run
}

func (r *fakeRunsRepo) Create(ctx context.Context, run runs.Run) (runs.Run, error) {
	run.ID = "run_1"
	r.created = run
	r.latest = run
	return run, nil
}

func (r *fakeRunsRepo) Update(ctx context.Context, run runs.Run) error {
	r.latest = run
	return nil
}

type fakeAdapter struct {
	result agent.RunResult
	err    error
}

func (a *fakeAdapter) Name() string                   { return "fake" }
func (a *fakeAdapter) Validate(context.Context) error { return nil }
func (a *fakeAdapter) Run(context.Context, agent.RunInput) (agent.RunResult, error) {
	return a.result, a.err
}

func TestRunJobSucceeded(t *testing.T) {
	jobRepo := &fakeJobRepo{job: jobs.Job{
		ID:             "job_1",
		Name:           "demo",
		Tool:           "codex-cli",
		PromptTemplate: "say hi",
		Enabled:        true,
	}}
	runsRepo := &fakeRunsRepo{}
	adapter := &fakeAdapter{result: agent.RunResult{ExitCode: 0, Stdout: "ok", Duration: 2 * time.Second}}

	svc, err := NewService(jobRepo, runsRepo,
		WithAdapters(map[string]agent.Adapter{"codex-cli": adapter}),
		WithAllowedWorkdirRoots([]string{"."}),
	)
	if err != nil {
		t.Fatalf("NewService error: %v", err)
	}

	run, err := svc.RunJob(context.Background(), "job_1")
	if err != nil {
		t.Fatalf("RunJob error: %v", err)
	}
	if run.Status != runs.StatusSucceeded {
		t.Fatalf("status = %s, want %s", run.Status, runs.StatusSucceeded)
	}
	if run.ExitCode == nil || *run.ExitCode != 0 {
		t.Fatalf("exit code = %v, want 0", run.ExitCode)
	}
}

func TestRunJobFailedOnAgentError(t *testing.T) {
	jobRepo := &fakeJobRepo{job: jobs.Job{
		ID:             "job_1",
		Name:           "demo",
		Tool:           "codex-cli",
		PromptTemplate: "say hi",
		Enabled:        true,
	}}
	runsRepo := &fakeRunsRepo{}
	adapter := &fakeAdapter{result: agent.RunResult{ExitCode: 1, Stderr: "bad"}}

	svc, err := NewService(jobRepo, runsRepo,
		WithAdapters(map[string]agent.Adapter{"codex-cli": adapter}),
		WithAllowedWorkdirRoots([]string{"."}),
	)
	if err != nil {
		t.Fatalf("NewService error: %v", err)
	}

	run, err := svc.RunJob(context.Background(), "job_1")
	if err == nil {
		t.Fatal("expected run error")
	}
	if run.Status != runs.StatusFailed {
		t.Fatalf("status = %s, want %s", run.Status, runs.StatusFailed)
	}
}

func TestRunJobTimedOut(t *testing.T) {
	jobRepo := &fakeJobRepo{job: jobs.Job{
		ID:             "job_1",
		Name:           "demo",
		Tool:           "codex-cli",
		PromptTemplate: "say hi",
		Enabled:        true,
	}}
	runsRepo := &fakeRunsRepo{}
	adapter := &fakeAdapter{err: context.DeadlineExceeded}

	svc, err := NewService(jobRepo, runsRepo,
		WithAdapters(map[string]agent.Adapter{"codex-cli": adapter}),
		WithAllowedWorkdirRoots([]string{"."}),
	)
	if err != nil {
		t.Fatalf("NewService error: %v", err)
	}

	run, err := svc.RunJob(context.Background(), "job_1")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want deadline exceeded", err)
	}
	if run.Status != runs.StatusTimedOut {
		t.Fatalf("status = %s, want %s", run.Status, runs.StatusTimedOut)
	}
}
