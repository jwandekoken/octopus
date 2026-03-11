# Spike Graduation Plan

This document defines how we transition the current agent-integration spike into a stable product capability.

Related docs:

- [Agent Integration Spike](./agent-integration-spike.md)
- [Project Architecture](./project-architecture.md)
- [Product Design](./product-design.md)

## Current spike value

- proven adapter interface
- proven codex/claude non-interactive execution path
- proven output normalization contract

## Migration plan

1. Rename/reposition the spike service

- Move `internal/spike/service.go` to `internal/core/execution/service.go`.
- Rename `spike.Service` to `execution.Service`.

2. Introduce repository ports

- Add interfaces in `internal/core/runs` and `internal/core/jobs` for required reads/writes.
- Start with minimum methods needed by execution and scheduler.

3. Persist run lifecycle

- On start: create run record (`queued` -> `running`).
- On finish: update to `succeeded` / `failed` / `timed_out` with exit code and duration.
- Store stdout/stderr (or file references if large).

4. Replace `spike run` with product commands

- `octopus job run <job-id|name>`
- Optional temporary alias: keep `spike run` for compatibility until removed.

5. Replace `spike validate` with adapter health command

- `octopus agent validate` (all or specific adapter).

6. Reuse execution service inside scheduler

- `octopus scheduler tick` loads due schedules and calls `execution.Service`.
- Scheduler should not know adapter internals.

7. Add production guardrails

- Concurrency limit for runs per tick.
- Timeout policy from job config with sensible defaults.
- Retry policy at scheduler layer (not inside adapters).
- Working directory boundaries validated before run.

8. Testing before removing spike label

- Unit tests: execution state transitions and failure handling.
- Integration tests: SQLite run persistence and adapter invocation.
- CLI tests: `job run` and `scheduler tick` happy/failure paths.

9. Decommission spike namespace

- Remove `octopus spike ...` commands after product commands are stable.
- Keep docs/examples on product commands only.

## Recommended next implementation slice

Implement this thin vertical slice first:

1. SQLite-backed `runs` repository.
2. `execution.Service` using existing adapters.
3. `octopus job run <job>` that creates and finalizes a run.

This makes the spike behavior part of real product flow and unlocks scheduler work immediately after.
