# Octopus Product Design

## Goal

Build a local-first CLI application that manages projects and runs recurring AI-driven jobs against them.

The product should support:

- registering projects
- defining jobs for those projects
- scheduling recurring executions
- dispatching work to external AI agent CLIs such as `claude-code`, `codex-cli`, and `gemini-cli`
- storing execution history, logs, and outputs

The application itself should not try to be a general-purpose agent. It should orchestrate project data, schedules, prompts, and execution, while delegating the actual AI work to external tools through adapters.

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

## Execution Model

The recommended architecture is local-first and cron-driven.

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

## CLI Design

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

Example workflow:

```bash
octopus project add --name website --type code --root ~/coding/website
octopus job create --project website --name weekly-review --tool codex-cli
octopus schedule add --job weekly-review --cron "0 9 * * 1"
octopus scheduler tick
```

## Agent Adapter Layer

This is the key design abstraction.

The system should define a common adapter interface for external AI CLIs:

```ts
interface AgentAdapter {
  name: "claude-code" | "codex-cli" | "gemini-cli";
  validate(): Promise<void>;
  run(input: {
    prompt: string;
    workingDir: string;
    timeoutSec?: number;
    env?: Record<string, string>;
  }): Promise<{
    exitCode: number;
    stdout: string;
    stderr: string;
    artifacts?: string[];
  }>;
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

- `TypeScript`
- `Node.js`
- `commander` or `oclif` for the CLI
- `SQLite` for storage
- `drizzle` or `better-sqlite3` for persistence
- `cron-parser` for schedule evaluation
- `zod` for input validation

Why this stack:

- fast iteration for CLI development
- good support for subprocess execution
- strong support for JSON-heavy workflows
- simple SQLite integration
- easy prompt and metadata handling

An alternative would be `Go`, which is also a strong fit for a standalone CLI. For an MVP, TypeScript is likely faster to evolve.

## MVP Scope

Version 1 should include:

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
- advanced workflow orchestration
- remote hosting

## Risks and Design Constraints

The biggest complexity is not the CLI surface. It is the execution behavior of the external agent tools.

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
  src/
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

1. implement `project add` and `project list`
2. implement `job create` and `job run`
3. add one adapter, preferably `codex-cli`
4. add SQLite persistence
5. implement `scheduler tick`
6. implement run history and log inspection

This creates an end-to-end loop before adding more tools or more complex orchestration.

## Future Extensions

After the MVP is stable, likely next steps are:

- support multiple agent adapters with profile-based configuration
- add richer prompt templating and reusable job templates
- support chained jobs or simple workflows
- add notifications or reports after scheduled runs
- add a lightweight UI if operational visibility becomes necessary
