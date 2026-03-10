# Octopus Product Design

## Goal

Build a local-first terminal application that manages projects and runs recurring AI-driven jobs against them.

The product should support:

- registering projects
- defining jobs for those projects
- scheduling recurring executions
- dispatching work to external AI agent CLIs such as `claude-code`, `codex-cli`, and `gemini-cli`
- storing execution history, logs, and outputs

The primary human interface should be a terminal UI (TUI) similar in spirit to tools like `lazygit` or `lazydocker`.

The application itself should not try to be a general-purpose agent. It should orchestrate project data, schedules, prompts, and execution, while delegating the actual AI work to external tools through adapters.

The product should still expose a small CLI surface for automation, scripting, and cron integration.

## Product Boundaries

The product owns:

- project registry
- job definitions
- schedule definitions
- prompt templating
- run history
- logs and artifacts
- adapter integration with external AI CLIs

The product should not initially own:

- a web UI
- team collaboration
- distributed workers
- complex multi-step workflows
- hosting or remote orchestration

## Core Concepts

### Project

Represents a unit of work being managed by the system.

Examples:

- coding repository
- social media content pipeline
- ad campaign workspace
- research initiative

Suggested fields:

- `id`
- `name`
- `type`
- `root_path`
- `description`
- `tags`
- `metadata`

Example project types:

- `code`
- `content`
- `ads`
- `research`

### Job

Represents a reusable task definition for a project. A job describes what tool to run, what prompt to send, and how execution should be configured.

Suggested fields:

- `id`
- `project_id`
- `name`
- `tool`
- `prompt_template`
- `working_dir`
- `input_vars`
- `timeout`
- `enabled`

### Schedule

Represents when a job should run.

Suggested fields:

- `id`
- `job_id`
- `cron_expr`
- `timezone`
- `next_run_at`
- `enabled`

### Run

Represents one execution of a job.

Suggested fields:

- `id`
- `job_id`
- `status`
- `started_at`
- `finished_at`
- `exit_code`
- `stdout`
- `stderr`
- `artifacts`

### Artifact

Represents generated output from a run, usually as file references or structured output metadata.

### Profile

Represents reusable execution defaults.

Examples:

- model selection
- environment variables
- approval mode
- sandbox mode

### Workflow

Represents an ordered set of jobs that should be executed as a single flow.

A workflow allows task piping by passing context and outputs from one step to the next.

Suggested fields:

- `id`
- `project_id`
- `name`
- `description`
- `steps` (ordered references to jobs)
- `on_failure` (stop, continue, or retry policy)
- `enabled`

Initial constraints for future implementation:

- linear sequence only (no DAG/branching)
- sequential execution only
- optional shared context map across steps
- parent workflow run with child job runs for traceability

## Execution Model

The recommended architecture is local-first and cron-driven.

Detailed scheduling semantics are defined in [Scheduling Strategy](./scheduling-strategy.md).

Instead of building a long-running daemon for the first version, the product should:

- store schedules in SQLite
- expose a command such as `octopus scheduler tick`
- have the operating system call that command on a regular cadence, such as every minute

Each scheduler tick should:

1. load schedules that are due
2. create `Run` records
3. execute each matching job
4. store logs, outputs, and status
5. compute and persist the next scheduled run time

This approach is simpler to operate, easier to debug, and more portable than building a background service too early.

## Interface Design

The product should have two interfaces over the same application services:

- a TUI for interactive daily use
- a CLI for automation, scripting, and scheduler entry points

The TUI should be the primary interface for humans. The CLI should remain narrow and stable.

### TUI Design

The TUI should provide an operator-oriented layout with keyboard-driven workflows.

Suggested layout:

- left pane for navigation between projects, jobs, schedules, and runs
- main pane for detail views and editable forms
- lower pane for logs, run output, and status
- modal overlays for create, edit, confirm, and filter flows

Suggested actions:

- create and edit projects
- create and edit jobs
- create and edit schedules
- trigger ad hoc runs
- inspect run history and logs
- enable and disable jobs and schedules
- validate adapter availability

### CLI Design

Suggested command surface:

```bash
octopus project add
octopus project list
octopus project show <project>

octopus job create
octopus job list
octopus job run <job>
octopus job test <job>

octopus schedule add <job>
octopus schedule list
octopus scheduler tick

octopus run list
octopus run show <run>
octopus run logs <run>
```

Example automation workflow:

```bash
octopus project add --name website --type code --root ~/coding/website
octopus job create --project website --name weekly-review --tool codex-cli
octopus schedule add --job weekly-review --cron "0 9 * * 1"
octopus scheduler tick
```

## Agent Adapter Layer

This is the key design abstraction.

The system should define a common adapter interface for external AI CLIs.

In implementation terms, this should be a small Go interface owned by the core execution layer rather than by the TUI:

```go
type AgentAdapter interface {
	Name() string
	Validate(ctx context.Context) error
	Run(ctx context.Context, input RunInput) (RunResult, error)
}
```

Suggested supporting types:

```go
type RunInput struct {
	Prompt     string
	WorkingDir string
	TimeoutSec int
	Env        map[string]string
}

type RunResult struct {
	ExitCode  int
	Stdout    string
	Stderr    string
	Artifacts []string
}
```

Each adapter should only be responsible for:

- validating that the CLI is available
- translating a job into the correct command invocation
- passing prompt and execution context
- capturing stdout and stderr
- returning a normalized result

This keeps the rest of the system independent from tool-specific behavior.

## Prompting Model

Jobs should use prompt templates rather than fixed strings.

Example:

```txt
Project: {{project.name}}
Type: {{project.type}}
Root: {{project.root_path}}

Task:
Review the current status of this project and propose the next 3 actions.

Additional context:
{{job.input_vars.context}}
```

Variables should be allowed from:

- project metadata
- job input variables
- runtime overrides
- previous run outputs where appropriate

This makes the system flexible enough to support both coding and non-coding workflows without changing the execution engine.

## Technology Recommendation

Recommended stack for the first version:

- `Go`
- `Bubble Tea` for the TUI application model
- `Bubbles` for reusable TUI components
- `Lip Gloss` for styling and layout
- `Cobra` for the CLI entry points
- `SQLite` for storage
- `database/sql` with a SQLite driver for persistence
- a cron parser library for schedule evaluation

Why this stack:

- strong fit for a `lazygit`-style TUI
- Bubble Tea provides a mature event loop and update model
- Bubbles gives reusable components for common terminal interactions
- Go compiles to a single portable binary
- good support for subprocess execution and streaming output
- simple SQLite integration
- straightforward concurrency model for scheduler and run execution

`Rust` with `ratatui` remains a viable alternative, but `Go` with `Bubble Tea` is the preferred choice for this project because it offers a more complete TUI component ecosystem and a faster path to an interactive operator interface.

## MVP Scope

Version 1 should include:

- a keyboard-driven TUI for project, job, schedule, and run management
- project registration
- job creation
- manual job execution
- cron-based recurring execution
- one external AI tool per job
- local run history and logs

Version 1 should exclude:

- distributed execution
- web interfaces
- team accounts
- advanced workflow orchestration (workflows are post-MVP)
- remote hosting

## Risks and Design Constraints

The biggest complexity is not the TUI surface. It is the execution behavior of the external agent tools.

Important constraints to define early:

- non-interactive invocation requirements
- timeout handling
- output capture rules
- allowed file system side effects
- concurrency limits
- retry behavior

Potential risks:

- each AI CLI may expose different invocation semantics
- some tools may default to interactive behavior
- output formats may vary significantly
- jobs that modify files need strong working directory boundaries
- recurring jobs need idempotent behavior or clear retry policies

## Suggested Repository Structure

```txt
octopus/
  cmd/
    octopus/
  internal/
    tui/
      app/
      views/
      components/
    cli/
    core/
      projects/
      jobs/
      schedules/
      runs/
    adapters/
      claude-code/
      codex-cli/
      gemini-cli/
    scheduler/
    storage/
    prompts/
  data/
  logs/
  docs/
```

## Recommended First Milestone

Build the first usable slice in this order:

1. implement shared core models and SQLite persistence
2. add one adapter, preferably `codex-cli`
3. implement `scheduler tick`
4. build a TUI view for project list and project creation
5. add job creation and ad hoc job execution flows
6. implement run history and log inspection in the TUI
7. keep a minimal CLI for automation and scheduler entry points

This creates an end-to-end loop before adding more tools or more complex orchestration.

## Future Extensions

After the MVP is stable, likely next steps are:

- support multiple agent adapters with profile-based configuration
- add richer prompt templating and reusable job templates
- support chained jobs or simple workflows (`Workflow`)
- add notifications or reports after scheduled runs
- add richer dashboards and workflow views in the TUI
