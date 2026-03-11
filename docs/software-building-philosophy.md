# Software Building Philosophy

Based on: [How to write complex software](https://grantslatton.com/how-to-software) by Grant Slatton.

This document defines how we build Octopus going forward.

## Why this exists

Octopus is becoming complex software (multiple layers: CLI/TUI, scheduling, adapters, storage, execution). We need a shared method to avoid premature architecture lock-in and to keep delivery fast.

## Core principles

1. Start with constraints, not architecture
- Write small probes and micro-benchmarks first for unknowns (CLI behavior, timeout behavior, scheduler cadence, SQLite limits, log volume).
- Use results to eliminate bad designs early.

2. Build top-down
- Start from the top layer that users touch (`octopus` commands, then TUI flows).
- Let each layer define what it needs from the layer below.
- Do not start from low-level components just because they are dependency-free.

3. Build in thin layers
- Each layer should do the minimum orchestration and delegate the rest down.
- If a function is hard to read, the layer is probably doing too much.

4. Define the “perfect API” for the next layer
- While implementing a layer, design the lower layer API to make current code clean and obvious.
- Favor domain language over generic utility APIs.

5. Stub aggressively, then recurse
- Stub lower layers just enough to compile and validate flow.
- After the upper layer behavior is clear, implement the lower layer for real using the same method.

6. Mock only IO boundaries
- Use concrete implementations for domain logic.
- Introduce interfaces/mocks primarily for IO (filesystem, subprocesses, network, clock/time, DB drivers), to keep tests fast and deterministic.

7. Backtrack fast when an API is impossible
- If a lower layer cannot implement the API cleanly, delete/rework the upper layer and redesign the seam.
- Do not force unimplementable abstractions.

8. Keep software always demoable
- Prefer partially working vertical slices over “foundation-first” long dead zones.
- At all times, aim to have at least one user-visible path working.

## How we apply this to Octopus

## Layer order (current)

1. User entry points
- CLI commands and later TUI flows (`project`, `job`, `schedule`, `run`, `scheduler tick`).

2. Application services
- Orchestration for validation, run creation, execution, scheduling decisions.

3. Domain and adapter contracts
- `agent.Adapter`, scheduler contracts, run lifecycle contracts.

4. Infrastructure
- SQLite persistence, subprocess execution, filesystem/log writing.

## Working rules for new features

1. Implement vertical slice first
- Example: `job run` should create a run record, execute adapter, store output, and show result before we generalize.

2. Introduce interfaces only at IO seams
- Avoid interface-heavy domain code.
- Keep most business logic concrete and directly testable.

3. Prefer stubs over speculative implementation
- If scheduler depends on persistence details not ready yet, stub repository behavior and complete scheduler flow first.

4. Tests prioritize determinism
- Unit tests run with mocked IO seams only.
- Integration tests cover real adapter invocation and real SQLite behavior.

5. Optimize with data
- Before optimizing concurrency/retries/buffering, add targeted measurements and timing output.

## Definition of done (for a layer)

A layer is “done enough” when:

1. The public behavior is clear and callable from the layer above.
2. The API to lower layers is stable enough to delegate implementation.
3. There are tests for success and failure paths.
4. It is readable without digging into lower-layer internals.

## Near-term implementation policy

For the next milestones, we will follow this sequence:

1. Complete a thin persisted run path (SQLite-backed `runs` with adapter execution).
2. Add scheduler tick using the same execution service.
3. Expand project/job/schedule persistence as needed by active CLI/TUI flows.
4. Add TUI views after execution and scheduling behavior are reliable.

## Notes

- These are heuristics, not rigid rules.
- We deliberately trade a small amount of temporary stub work for better architecture and faster iteration.
