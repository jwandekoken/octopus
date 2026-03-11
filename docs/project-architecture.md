# Octopus Project Architecture

This document defines the target architecture for Octopus.

It complements:

- [Product Design](./product-design.md)
- [Agent Integration Spike](./agent-integration-spike.md)
- [Software Building Philosophy](./software-building-philosophy.md)
- [Spike Graduation Plan](./spike-graduation-plan.md)

## Architecture goals

1. Keep user-facing flows simple (`octopus job run`, `octopus scheduler tick`, TUI actions).
2. Isolate domain behavior from infrastructure details (SQLite, subprocess invocation).
3. Treat external AI CLIs as pluggable adapters behind a stable contract.
4. Preserve local-first, debuggable operation.

## Go ecosystem best practices (applied)

1. Keep `cmd/` thin
- `cmd/octopus` should parse args, wire dependencies, call application services.
- No business logic in command handlers.

2. Use `internal/` for product code
- Prevent accidental public API commitments while architecture evolves.

3. Organize by feature + layer, not by “utils”
- Prefer cohesive packages (`projects`, `jobs`, `runs`, `scheduler`) with explicit dependencies.
- Avoid generic helpers that hide domain semantics.

4. Define interfaces at IO boundaries
- Repositories, clock, process runner, and agent adapters are boundaries.
- Keep domain/application logic concrete and easy to read.

5. Favor explicit constructors and dependency injection
- `NewService(repo, adapters, clock, logger)` style wiring.
- No hidden global state.

6. Context-first operations
- All potentially blocking operations should take `context.Context`.
- Respect cancellation/timeouts in scheduler and adapter execution.

7. Errors are wrapped with context
- Use `fmt.Errorf("create run: %w", err)` patterns.
- Keep domain-level sentinel errors where useful (`ErrJobNotFound`, `ErrAdapterUnavailable`).

8. Testing pyramid for Go
- Fast unit tests for domain and service logic.
- Integration tests for SQLite repositories and adapter subprocess behavior.
- Minimal end-to-end command tests for CLI flows.

9. Stable migrations and schemas
- Versioned SQL migrations; never rely on ad-hoc schema mutation.
- Keep migrations idempotent and CI-tested.

10. Structured logs for operators
- Log job/run/schedule IDs and durations.
- Keep log lines parseable and useful for terminal debugging.

## Target package layout

```txt
octopus/
  cmd/
    octopus/
      main.go
  internal/
    app/                     # command wiring/composition root
    cli/                     # CLI command handlers (thin)
    tui/                     # Bubble Tea app/views/components
    core/
      projects/              # project domain + service
      jobs/                  # job domain + service
      schedules/             # schedule domain + service
      runs/                  # run lifecycle domain + service
      execution/             # orchestration of adapter execution
      agent/                 # adapter contract + process runner
    adapters/
      codexcli/
      claudecode/
      geminicli/
    storage/
      sqlite/
        migrations/
        projects_repo.go
        jobs_repo.go
        schedules_repo.go
        runs_repo.go
    scheduler/
      tick.go
```

Notes:

- Keep `internal/core/agent` as the normalized adapter interface package.
- `internal/core/execution` is the product execution orchestration service.

## Runtime architecture

1. CLI/TUI invokes application services.
2. Services read/write project/job/schedule/run state via repositories.
3. Execution service resolves adapter by tool name.
4. Adapter executes CLI non-interactively and returns normalized output.
5. Run record is updated with status, timings, exit code, stdout/stderr metadata.
6. Scheduler tick queries due schedules and invokes the same execution service.
