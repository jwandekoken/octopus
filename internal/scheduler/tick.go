package scheduler

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jc/octopus/internal/core/runs"
	"github.com/jc/octopus/internal/core/schedules"
	"github.com/robfig/cron/v3"
)

type Execution interface {
	RunJob(ctx context.Context, idOrName string) (runs.Run, error)
}

type Service struct {
	repo        schedules.Repository
	execution   Execution
	concurrency int
	retries     int
	now         func() time.Time
}

type TickResult struct {
	Due      int
	Executed int
	Failed   int
}

func NewService(repo schedules.Repository, execution Execution, concurrency int, retries int) *Service {
	if concurrency <= 0 {
		concurrency = 1
	}
	if retries < 0 {
		retries = 0
	}
	return &Service{
		repo:        repo,
		execution:   execution,
		concurrency: concurrency,
		retries:     retries,
		now:         time.Now,
	}
}

func (s *Service) Tick(ctx context.Context, limit int) (TickResult, error) {
	if limit <= 0 {
		limit = 100
	}

	now := s.now().UTC()
	due, err := s.repo.ListDue(ctx, now, limit)
	if err != nil {
		return TickResult{}, fmt.Errorf("list due schedules: %w", err)
	}

	result := TickResult{Due: len(due)}
	if len(due) == 0 {
		return result, nil
	}

	sem := make(chan struct{}, s.concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for _, schedule := range due {
		schedule := schedule
		wg.Add(1)

		go func() {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				mu.Lock()
				if firstErr == nil {
					firstErr = ctx.Err()
				}
				mu.Unlock()
				return
			}
			defer func() { <-sem }()

			jobErr := s.runWithRetry(ctx, schedule.JobID)
			if jobErr != nil {
				mu.Lock()
				result.Failed++
				mu.Unlock()
			}

			nextRun, nextErr := nextRunAt(schedule, now)
			if nextErr != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = nextErr
				}
				mu.Unlock()
				return
			}
			if err := s.repo.Advance(ctx, schedule.ID, schedule.NextRunAt, nextRun); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("advance schedule %s: %w", schedule.ID, err)
				}
				mu.Unlock()
				return
			}

			mu.Lock()
			result.Executed++
			mu.Unlock()
		}()
	}

	wg.Wait()
	if firstErr != nil {
		return result, firstErr
	}
	return result, nil
}

func (s *Service) runWithRetry(ctx context.Context, jobID string) error {
	var err error
	attempts := s.retries + 1
	for i := 0; i < attempts; i++ {
		_, err = s.execution.RunJob(ctx, jobID)
		if err == nil {
			return nil
		}
	}
	return err
}

func nextRunAt(schedule schedules.Schedule, now time.Time) (time.Time, error) {
	loc, err := time.LoadLocation(strings.TrimSpace(schedule.Timezone))
	if err != nil || loc == nil {
		loc = time.UTC
	}

	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	spec, err := parser.Parse(schedule.CronExpr)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse cron for schedule %s: %w", schedule.ID, err)
	}

	next := schedule.NextRunAt.In(loc)
	for !next.After(now.In(loc)) {
		next = spec.Next(next)
	}

	return next.UTC(), nil
}
