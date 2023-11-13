# Rhino

Rhino is a mini Airflow implementation in Go. It provides a simple way to define workflows and schedule tasks.

## Features

- Define workflows in YAML format.
- Schedule tasks using cron syntax.
- Run tasks concurrently within a workflow.
- Control the order of task execution within a workflow.
- Support for webhook triggers.

## Usage

1. Define your workflows in a YAML file. Each workflow should include a list of tasks and an order field that specifies the order in which the tasks should be run.

2. Start the Rhino runner with the command `go run main.go runner`. This will start the runner and load the workflows.

3. The runner will start the workflows according to their schedule. You can also trigger workflows manually using webhooks or 'go run main.go run <workflow>'

## Debugging

Rhino includes extensive logging to help you understand what's happening. If a workflow is not starting as expected, check the logs for error messages and warnings.