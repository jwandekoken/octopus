# Spike Graduation Plan

This document defines how we transition the current agent-integration spike into a stable product capability.

Related docs:

- [Agent Integration Spike](../done/agent-integration-spike.md)
- [Project Architecture](../../project-architecture.md)
- [Product Design](../../product-design.md)

## Current spike value

- proven adapter interface
- proven codex/claude non-interactive execution path
- proven output normalization contract

## Task checklist

### Phase 1: Turn spike service into product execution service

- [ ] Move `internal/spike/service.go` to `internal/core/execution/service.go`.
- [ ] Rename `spike.Service` to `execution.Service`.
- [ ] Keep behavior parity for existing adapter execution while refactoring.

### Phase 2: Add minimal persistence seam

- [ ] Define repository interfaces in `internal/core/runs` and `internal/core/jobs` required by execution.
- [ ] Add SQLite-backed `runs` repository with create/update methods.
- [ ] Persist run lifecycle states: `queued`, `running`, `succeeded`, `failed`, `timed_out`.
- [ ] Persist run metadata: exit code, duration, stdout/stderr (or references).

### Phase 3: Product CLI commands (replace spike surface)

- [ ] Implement `octopus job run <job-id|name>` using `execution.Service`.
- [ ] Implement `octopus agent validate` for all/specific adapters.
- [ ] Keep temporary alias compatibility for `octopus spike run` and `octopus spike validate`.

### Phase 4: Scheduler integration

- [ ] Wire `octopus scheduler tick` to invoke `execution.Service` for due jobs.
- [ ] Ensure scheduler depends on execution interface, not adapter internals.

### Phase 5: Production guardrails

- [ ] Add concurrency limit for runs per scheduler tick.
- [ ] Add timeout policy from job configuration with defaults.
- [ ] Add retry policy at scheduler layer.
- [ ] Validate and enforce working directory boundaries before execution.

### Phase 6: Tests and hardening

- [ ] Add unit tests for execution state transitions and failure handling.
- [ ] Add integration tests for SQLite run persistence.
- [ ] Add integration tests for adapter invocation contract.
- [ ] Add CLI tests for `job run` and `scheduler tick` happy/failure paths.

### Phase 7: Decommission spike namespace

- [ ] Remove `octopus spike ...` commands after product commands are stable.
- [ ] Update docs/examples to use product commands only.

## Definition of done

- [ ] Spike behavior is reachable through product commands (`job run`, `agent validate`, `scheduler tick`).
- [ ] Run lifecycle is persisted and queryable.
- [ ] Tests cover core success/failure paths.
- [ ] No production flow depends on `internal/spike/*`.
