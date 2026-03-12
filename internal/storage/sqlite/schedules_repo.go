package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jc/octopus/internal/core/schedules"
)

type SchedulesRepository struct {
	db *sql.DB
}

func (r *SchedulesRepository) ListDue(ctx context.Context, now time.Time, limit int) ([]schedules.Schedule, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if limit <= 0 {
		limit = 100
	}

	const query = `
		SELECT id, job_id, cron_expr, timezone, next_run_at, enabled
		FROM schedules
		WHERE enabled = 1 AND next_run_at <= ?
		ORDER BY next_run_at ASC
		LIMIT ?`

	rows, err := r.db.QueryContext(ctx, query, now.UTC().Format(time.RFC3339Nano), limit)
	if err != nil {
		return nil, fmt.Errorf("query due schedules: %w", err)
	}
	defer rows.Close()

	var out []schedules.Schedule
	for rows.Next() {
		var (
			schedule schedules.Schedule
			nextRun  string
			enabled  int
		)
		if err := rows.Scan(
			&schedule.ID,
			&schedule.JobID,
			&schedule.CronExpr,
			&schedule.Timezone,
			&nextRun,
			&enabled,
		); err != nil {
			return nil, fmt.Errorf("scan schedule: %w", err)
		}
		tm, err := time.Parse(time.RFC3339Nano, nextRun)
		if err != nil {
			return nil, fmt.Errorf("parse next_run_at: %w", err)
		}
		schedule.NextRunAt = tm.UTC()
		schedule.Enabled = enabled == 1
		out = append(out, schedule)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate schedules: %w", err)
	}

	return out, nil
}

func (r *SchedulesRepository) Advance(ctx context.Context, scheduleID string, previousNextRunAt time.Time, nextRunAt time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	const query = `
		UPDATE schedules
		SET next_run_at = ?
		WHERE id = ? AND next_run_at = ?`

	res, err := r.db.ExecContext(
		ctx,
		query,
		nextRunAt.UTC().Format(time.RFC3339Nano),
		scheduleID,
		previousNextRunAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("advance schedule: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("advance rows affected: %w", err)
	}
	if rows == 0 {
		return errors.New("schedule was concurrently updated")
	}

	return nil
}
