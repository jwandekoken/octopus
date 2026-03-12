package sqlite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jc/octopus/internal/core/runs"
)

type RunsRepository struct {
	db *sql.DB
}

func (r *RunsRepository) Create(ctx context.Context, run runs.Run) (runs.Run, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if run.ID == "" {
		id, err := newID("run")
		if err != nil {
			return runs.Run{}, err
		}
		run.ID = id
	}

	if run.CreatedAt.IsZero() {
		run.CreatedAt = time.Now().UTC()
	}

	const query = `
		INSERT INTO runs (
			id, job_id, status, created_at, started_at, finished_at, exit_code, duration_ms, stdout, stderr, error
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		run.ID,
		run.JobID,
		run.Status,
		run.CreatedAt.UTC().Format(time.RFC3339Nano),
		timePointer(run.StartedAt),
		timePointer(run.FinishedAt),
		intPointer(run.ExitCode),
		run.Duration.Milliseconds(),
		run.Stdout,
		run.Stderr,
		run.Error,
	)
	if err != nil {
		return runs.Run{}, fmt.Errorf("insert run: %w", err)
	}

	return run, nil
}

func (r *RunsRepository) Update(ctx context.Context, run runs.Run) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	const query = `
		UPDATE runs
		SET status = ?, started_at = ?, finished_at = ?, exit_code = ?, duration_ms = ?, stdout = ?, stderr = ?, error = ?
		WHERE id = ?`

	res, err := r.db.ExecContext(ctx, query,
		run.Status,
		timePointer(run.StartedAt),
		timePointer(run.FinishedAt),
		intPointer(run.ExitCode),
		run.Duration.Milliseconds(),
		run.Stdout,
		run.Stderr,
		run.Error,
		run.ID,
	)
	if err != nil {
		return fmt.Errorf("update run: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update rows affected: %w", err)
	}
	if rows == 0 {
		return runs.ErrNotFound
	}
	return nil
}

func newID(prefix string) (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return prefix + "_" + hex.EncodeToString(buf), nil
}

func timePointer(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func intPointer(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func parseOptionalTime(value sql.NullString) (*time.Time, error) {
	if !value.Valid || value.String == "" {
		return nil, nil
	}
	tm, err := time.Parse(time.RFC3339Nano, value.String)
	if err != nil {
		return nil, err
	}
	return &tm, nil
}

func parseOptionalInt(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	iv := int(value.Int64)
	return &iv
}

func (r *RunsRepository) getByID(ctx context.Context, id string) (runs.Run, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	const query = `
		SELECT id, job_id, status, created_at, started_at, finished_at, exit_code, duration_ms, stdout, stderr, error
		FROM runs
		WHERE id = ?`

	var (
		run        runs.Run
		createdAt  string
		startedAt  sql.NullString
		finishedAt sql.NullString
		exitCode   sql.NullInt64
		durationMS int64
	)

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&run.ID,
		&run.JobID,
		&run.Status,
		&createdAt,
		&startedAt,
		&finishedAt,
		&exitCode,
		&durationMS,
		&run.Stdout,
		&run.Stderr,
		&run.Error,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return runs.Run{}, runs.ErrNotFound
	}
	if err != nil {
		return runs.Run{}, fmt.Errorf("query run: %w", err)
	}

	tm, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return runs.Run{}, err
	}
	run.CreatedAt = tm
	run.StartedAt, err = parseOptionalTime(startedAt)
	if err != nil {
		return runs.Run{}, err
	}
	run.FinishedAt, err = parseOptionalTime(finishedAt)
	if err != nil {
		return runs.Run{}, err
	}
	run.ExitCode = parseOptionalInt(exitCode)
	run.Duration = time.Duration(durationMS) * time.Millisecond

	return run, nil
}
