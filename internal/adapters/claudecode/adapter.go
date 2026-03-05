package claudecode

import (
	"context"
	"errors"
	"os/exec"

	"github.com/jc/octopus/internal/core/agent"
)

type Adapter struct {
	Binary string
}

func New(binary string) *Adapter {
	if binary == "" {
		binary = "claude"
	}
	return &Adapter{Binary: binary}
}

func (a *Adapter) Name() string {
	return "claude-code"
}

func (a *Adapter) Validate(ctx context.Context) error {
	if _, err := exec.LookPath(a.Binary); err != nil {
		return err
	}
	return agent.HelpSanityCheck(ctx, a.Binary)
}

func (a *Adapter) Run(ctx context.Context, input agent.RunInput) (agent.RunResult, error) {
	if input.Prompt == "" {
		return agent.RunResult{}, errors.New("prompt is required")
	}

	// claude --print is the non-interactive contract.
	args := []string{"--print", input.Prompt}
	return agent.ExecCommand(ctx, a.Binary, args, input)
}
