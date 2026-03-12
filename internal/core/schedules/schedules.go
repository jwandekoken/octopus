package schedules

import (
	"context"
	"time"
)

type Schedule struct {
	ID        string
	JobID     string
	CronExpr  string
	Timezone  string
	NextRunAt time.Time
	Enabled   bool
}

type Repository interface {
	ListDue(ctx context.Context, now time.Time, limit int) ([]Schedule, error)
	Advance(ctx context.Context, scheduleID string, previousNextRunAt time.Time, nextRunAt time.Time) error
}
