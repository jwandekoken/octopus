# Agent Integration Spike

This spike intentionally focuses on the hardest unknown: non-interactive execution with external coding agents.

## Scope

Implemented now:

- shared adapter interface in `internal/core/agent`
- `codex-cli` adapter using `codex exec`
- `claude-code` adapter using `claude --print`
- one executable entry point for feasibility checks

Not implemented yet:

- storage
- scheduler
- TUI
- prompt templates

## Current product commands

Validate local setup:

```bash
go run ./cmd/octopus agent validate
```

Run a configured job:

```bash
go run ./cmd/octopus job run <job-id|name>
```

Execute due schedules:

```bash
go run ./cmd/octopus scheduler tick
```

## What this proves

If both adapters validate and run successfully, we confirm:

- local non-interactive invocations are possible
- common output capture works across both CLIs
- a normalized execution contract is viable for core orchestration

## Next step after spike pass

- persist projects/jobs/runs in SQLite
- wire `scheduler tick` to call this adapter layer
- add TUI views once execution reliability is acceptable

Detailed graduation plan: [Spike Graduation Plan](../in-progress/spike-graduation-plan.md).
