# Scheduling Strategy

## Goal

Define how recurring schedules are evaluated and advanced for `octopus scheduler tick`.

The scheduler is cron-driven and stateless between ticks except for persisted schedule data in SQLite.

## Schedule Fields

A schedule stores:

- `cron_expr`: the recurring rule, such as `0 9 * * 1`
- `timezone`: the timezone used to evaluate the cron expression
- `next_run_at`: the next concrete timestamp when the schedule becomes due
- `enabled`: whether the schedule is eligible for execution

`cron_expr` is the source of truth for recurrence. `next_run_at` is a cached materialized value derived from `cron_expr` and `timezone`.

## Core Policy

The scheduler uses this execution policy:

- run each due schedule at most once per tick
- after handling a due schedule, advance `next_run_at` to the first future occurrence defined by `cron_expr`
- do not attempt backlog replay for every missed interval

This means a schedule can become overdue, but a single scheduler tick still produces at most one run for that schedule.

## Schedule Lifecycle

### On Schedule Creation

When a schedule is created:

1. parse `cron_expr` in `timezone`
2. compute the first occurrence strictly after the creation time
3. persist that value as `next_run_at`

### On Schedule Update

When `cron_expr` or `timezone` changes:

1. re-parse the schedule definition
2. recompute `next_run_at` from the current time
3. persist the new value

If a schedule is disabled, it should not be considered by the scheduler tick.

## Scheduler Tick Algorithm

Each `octopus scheduler tick` should:

1. get the current time
2. load enabled schedules where `next_run_at <= now`
3. for each due schedule, create one `Run`
4. execute the associated job once
5. persist run status, logs, and outputs
6. advance `next_run_at` until it is strictly greater than `now`
7. save the updated `next_run_at`

The advancement step is important. It ensures that a delayed tick catches the schedule up to the next future slot without creating multiple runs for all missed intervals.

## Advancement Rule

When advancing a due schedule, compute the next occurrence from the prior scheduled time, not from the wall clock time of job completion.

In other words:

- good: "next occurrence after previous `next_run_at`"
- avoid: "next occurrence after current time"

This prevents drift and preserves the cadence defined by `cron_expr`.

## Example

Given:

- `cron_expr = "0 * * * *"`
- `timezone = "UTC"`
- `next_run_at = 2026-03-02T11:00:00Z`

If the scheduler tick runs at `2026-03-02T13:17:00Z`:

1. the schedule is due because `11:00 <= 13:17`
2. one run is created and executed
3. `next_run_at` is advanced from `11:00` to `12:00`, then `13:00`, then `14:00`
4. `14:00` is persisted because it is the first future occurrence

Result:

- runs executed on this tick: `1`
- persisted `next_run_at`: `2026-03-02T14:00:00Z`

## Rationale

This strategy keeps the system simple and predictable:

- scheduler ticks remain idempotent at the schedule level
- cron cadence stays stable over time
- missed scheduler windows do not create a burst of backlog runs
- the database stores a concrete next due time for efficient queries

## Open Questions

The following behaviors should be defined separately if needed:

- concurrency limits across multiple due schedules
- retry behavior for failed runs
- whether `next_run_at` advances when run creation fails before execution starts
- whether disabled schedules retain or clear their previous `next_run_at`
