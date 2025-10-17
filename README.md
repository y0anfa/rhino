# Rhino

Rhino is a mini Airflow implementation in Go. It provides a simple way to define workflows and schedule tasks.

## Features

- Define workflows in YAML format
- Schedule tasks using cron syntax
- Run tasks concurrently within a workflow
- Control the order of task execution within a workflow
- Support for webhook triggers (multiple workflows on single port)
- Flexible configuration via YAML files and environment variables
- Built-in providers: Shell and HTTP

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

## Commands

- `rhino runner` - Start the workflow runner daemon
- `rhino run <workflow>` - Manually run a specific workflow
- `rhino list` - List all available workflows
- `rhino describe <workflow>` - Show workflow details
- `rhino new <workflow>` - Create a new workflow
- `rhino delete <workflow>` - Delete a workflow

## Debugging

Rhino includes extensive logging to help you understand what's happening. If a workflow is not starting as expected, check the logs for error messages and warnings.
