# TUI MVP Kickoff Plan

This document defines the first implementation slice for the Octopus terminal UI.

Related docs:

- [Product Design](../../product-design.md)
- [Project Architecture](../../project-architecture.md)
- [Software Building Philosophy](../../software-building-philosophy.md)

## Goal

Ship a minimal but usable `octopus tui` experience using thin vertical slices, starting read-only and reusing existing services/repositories.

## MVP scope (v0)

Included:

- `octopus tui` command entrypoint
- three-pane layout:
  - navigation/summary pane
  - detail pane for selected item
  - lower pane for logs/run output
- keyboard controls:
  - `q` to quit
  - `tab` to switch focused pane
  - arrow keys to move selection
- read-only data views from SQLite-backed state
- first operational slice: recent runs list + run detail

Not in v0:

- create/edit flows
- modal forms
- filtering/search
- workflow orchestration UI

## Task checklist

### Phase 1: Entrypoint and shell

- [ ] Add `octopus tui` command in `cmd/octopus/main.go`.
- [ ] Create `internal/tui` package with Bubble Tea app skeleton.
- [ ] Implement base layout rendering and resize handling.
- [ ] Implement global keymap (`q`, `tab`, arrows).

### Phase 2: Static views to validate UX

- [ ] Render three panes with static/mock content.
- [ ] Implement pane focus state and visible cursor/selection behavior.
- [ ] Ensure stable rendering on narrow and wide terminal sizes.

### Phase 3: Data wiring (read-only)

- [ ] Add read models for projects/jobs/schedules/runs (start with runs as primary).
- [ ] Wire TUI startup to open the same SQLite store path rules used by CLI.
- [ ] Populate runs list from real data and show selected run detail.
- [ ] Add empty-state messaging when DB has no rows.

### Phase 4: Refresh and operator ergonomics

- [ ] Add manual refresh keybinding (`r`).
- [ ] Add periodic background refresh (conservative default interval).
- [ ] Preserve selection/focus across refresh when possible.
- [ ] Surface non-fatal data errors in status area without crashing.

### Phase 5: Tests and hardening

- [ ] Add unit tests for model update transitions (pane switch, selection move, quit).
- [ ] Add unit tests for data-to-view state mapping (non-empty and empty states).
- [ ] Add CLI smoke test for `octopus tui` command wiring and startup path.

## Acceptance criteria (MVP done)

- [ ] `octopus tui` launches from project root with no panic.
- [ ] User can switch panes and navigate lists using only keyboard.
- [ ] Runs pane displays real DB-backed rows when present.
- [ ] Selecting a run updates detail pane with status/output metadata.
- [ ] TUI exits cleanly via `q`.
- [ ] Tests cover core update logic and command wiring.

## First implementation slice

Implement in this order:

1. command wiring + app shell
2. static three-pane rendering
3. real runs list + run detail
4. refresh behavior
5. tests
