package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jc/octopus/internal/core/jobs"
)

type JobsRepository struct {
	db *sql.DB
}

func (r *JobsRepository) GetByID(ctx context.Context, id string) (jobs.Job, error) {
	const query = `SELECT id, name, tool, prompt_template, working_dir, timeout_seconds, enabled FROM jobs WHERE id = ?`
	return r.getOne(ctx, query, id)
}

func (r *JobsRepository) GetByName(ctx context.Context, name string) (jobs.Job, error) {
	const query = `SELECT id, name, tool, prompt_template, working_dir, timeout_seconds, enabled FROM jobs WHERE name = ?`
	return r.getOne(ctx, query, name)
}

func (r *JobsRepository) getOne(ctx context.Context, query string, arg string) (jobs.Job, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var (
		job            jobs.Job
		timeoutSeconds int64
		enabled        int
	)

	err := r.db.QueryRowContext(ctx, query, arg).Scan(
		&job.ID,
		&job.Name,
		&job.Tool,
		&job.PromptTemplate,
		&job.WorkingDir,
		&timeoutSeconds,
		&enabled,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return jobs.Job{}, jobs.ErrNotFound
	}
	if err != nil {
		return jobs.Job{}, fmt.Errorf("query job: %w", err)
	}

	job.Timeout = time.Duration(timeoutSeconds) * time.Second
	job.Enabled = enabled == 1
	return job, nil
}
