# Rhino

Rhino is a mini Airflow implementation in Go. It provides a simple way to define workflows and schedule tasks.

## Features

- Define workflows in YAML format
- Schedule tasks using cron syntax
- Run tasks concurrently within a workflow
- Control the order of task execution within a workflow
- Support for triggers (multiple workflows on single port)
- Flexible configuration via YAML files and environment variables
- Built-in providers: shell, http, file, slack, discord, email, workflow, and approval
- Per-task execution history with a CLI, TUI, and web dashboard

## Quick Start

1. **Configure Rhino** with a `config.yaml` file:

```yaml
workflows-dir: workflows
port: 8888
```

Or use environment variables:

```bash
export RHINO_WORKFLOWS_DIR=workflows
export RHINO_PORT=8888
```

See [CONFIG.md](CONFIG.md) for detailed configuration options.

2. **Define your workflows** in YAML files in the workflows directory. Each workflow should include:

- Settings (max-tries, timeout)
- Trigger (cron or webhook)
- Tasks (with providers and parameters)
- Order (execution sequence)

Example workflow:

```yaml
name: my-workflow
description: Example workflow
settings:
    max-tries: 3
    timeout: "30s"
trigger:
    name: cron-trigger
    type: cron
    schedule: "0 */6 * * *"  # Every 6 hours
tasks:
  - name: task1
    provider: "shell"
    params:
        command: "echo"
        args: ["Hello, World!"]
order:
  - [task1]
```

3. **Start the Rhino runner**:

```bash
./rhino runner
```

Or with custom configuration:

```bash
RHINO_PORT=9000 ./rhino runner
```

4. **Trigger workflows**:

- Cron workflows run automatically based on their schedule
- Webhook workflows can be triggered via HTTP POST:

```bash
curl -X POST http://localhost:8888/webhook/my-workflow
```

- Manual execution:

```bash
./rhino run my-workflow
```

## Workflow settings

```yaml
settings:
    max-tries: 3              # attempts per task (tasks can override with their own max-tries)
    timeout: "30s"            # per-attempt timeout (tasks can override with their own timeout)
    retry-backoff: exponential  # none | linear | exponential
    retry-base-delay: "1s"
    retry-max-delay: "1m"
    max-concurrent-runs: 1    # drop new runs while this many are already in flight (0 = unlimited)
    max-output-size: 65536    # truncate task output beyond this many bytes (0 = unlimited)
```

Every task is validated up front: names must be unique, every task must appear
exactly once in `order` (or use `depends-on`), and timeouts, backoff settings,
and conditions are checked before anything runs.

## Conditions and templates

Tasks can be gated on the outcome of earlier tasks:

```yaml
condition: "{{task.build.status}} == success"   # also: != , and the literals always / never
```

Params and notification messages can reference `{{task.NAME.output}}`,
`{{task.NAME.metadata.KEY}}`, `{{env.VAR}}`, `{{secret.NAME}}`,
`{{workflow.name}}`, `{{run.id}}`, and, in failure notifications,
`{{workflow.error}}`. Shell tasks expose `exit_code` as metadata, and a failed
command's stderr is included in the task error.

## Execution history

Every run and every task execution is recorded in `~/.rhino/history.db`:

```bash
./rhino history                 # recent runs
./rhino history <run-id>        # per-task status, duration, retries, and errors
./rhino history --since 24h --status failed
```

## Dashboard and API

`./rhino dashboard` serves a web UI on port 9090 (use `--port` to change it)
with a JSON API:

| Endpoint | Description |
|----------|-------------|
| `GET /api/workflows` | Workflow names |
| `GET /api/workflows/{name}` | Trigger, settings, tasks, and order of a workflow |
| `POST /api/workflows/{name}/run` | Trigger a run; returns `202` with a `run_id`. Add `?wait=true` to block until it finishes |
| `GET /api/runs?workflow=&status=&since=24h&limit=` | Recent runs |
| `GET /api/runs/{id}` | A run with its task executions |
| `GET /api/health` | Status, uptime, workflow count |

Webhook triggers on the runner (`POST /webhook/{name}`) only accept `POST` and
respond with the `run_id` of the run they started.

## Commands

- `rhino runner` - Start the workflow runner daemon
- `rhino run <workflow>` - Manually run a specific workflow
- `rhino list` - List all available workflows
- `rhino describe <workflow>` - Show workflow details
- `rhino validate <workflow>` - Validate a workflow without running it
- `rhino history [run-id]` - Inspect execution history
- `rhino dashboard` - Start the web dashboard and API
- `rhino new <workflow>` - Create a new workflow
- `rhino delete <workflow>` - Delete a workflow

## Debugging

Rhino includes extensive logging to help you understand what's happening. If a workflow is not starting as expected, check the logs for error messages and warnings.
