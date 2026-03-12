package jobs

import (
	"context"
	"errors"
	"strings"
	"time"
)

var ErrNotFound = errors.New("job not found")

type Job struct {
	ID             string
	Name           string
	Tool           string
	PromptTemplate string
	WorkingDir     string
	Timeout        time.Duration
	Enabled        bool
}

type Repository interface {
	GetByID(ctx context.Context, id string) (Job, error)
	GetByName(ctx context.Context, name string) (Job, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Resolve(ctx context.Context, idOrName string) (Job, error) {
	ref := strings.TrimSpace(idOrName)
	if ref == "" {
		return Job{}, errors.New("job reference is required")
	}

	job, err := s.repo.GetByID(ctx, ref)
	if err == nil {
		if !job.Enabled {
			return Job{}, errors.New("job is disabled")
		}
		return job, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return Job{}, err
	}

	job, err = s.repo.GetByName(ctx, ref)
	if err != nil {
		return Job{}, err
	}
	if !job.Enabled {
		return Job{}, errors.New("job is disabled")
	}

	return job, nil
}
