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

## Commands

Validate local setup:

```bash
go run ./cmd/octopus spike validate
```

Run codex non-interactively:

```bash
go run ./cmd/octopus spike run --tool codex-cli --prompt "Say hello in one line"
```

Run claude non-interactively:

```bash
go run ./cmd/octopus spike run --tool claude-code --prompt "Say hello in one line"
```

Set repository scope for tools:

```bash
go run ./cmd/octopus spike run \
  --tool codex-cli \
  --workdir /path/to/repo \
  --prompt "List changed files and summarize risk"
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
