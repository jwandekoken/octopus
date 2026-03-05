package spike

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jc/octopus/internal/adapters/claudecode"
	"github.com/jc/octopus/internal/adapters/codexcli"
	"github.com/jc/octopus/internal/core/agent"
)

type Service struct {
	adapters map[string]agent.Adapter
}

func NewService() *Service {
	return &Service{
		adapters: map[string]agent.Adapter{
			"codex-cli":   codexcli.New(""),
			"claude-code": claudecode.New(""),
		},
	}
}

func (s *Service) Validate(ctx context.Context) error {
	for name, adapter := range s.adapters {
		if err := adapter.Validate(ctx); err != nil {
			return fmt.Errorf("%s validation failed: %w", name, err)
		}
	}
	return nil
}

func (s *Service) Run(ctx context.Context, tool, prompt, workingDir string, timeout time.Duration) (agent.RunResult, error) {
	adapter, ok := s.adapters[tool]
	if !ok {
		return agent.RunResult{}, fmt.Errorf("unknown tool %q", tool)
	}
	if prompt == "" {
		return agent.RunResult{}, errors.New("prompt is required")
	}

	return adapter.Run(ctx, agent.RunInput{
		Prompt:     prompt,
		WorkingDir: workingDir,
		Timeout:    timeout,
	})
}
