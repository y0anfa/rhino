/*
	Shell Provider inherits from the Provider interface and implements the Name and Run methods.

It also registers itself with the Register function. It is a plugin that allows the user to run shell commands.
*/
package providers

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// maxStderrInError caps how much of a failed command's stderr is attached to
// the returned error so a noisy process cannot flood logs or history.
const maxStderrInError = 4096

// ShellProvider is the shell provider.
type ShellProvider struct{}

// Name returns the name of the provider.
func (p *ShellProvider) Name() string {
	return "shell"
}

// Validate validates the provider arguments. "command" is required; "args" is
// an optional list of strings.
func (p *ShellProvider) Validate(args map[string]interface{}) error {
	if args["command"] == nil || args["command"] == "" {
		return fmt.Errorf("shell provider validation failed: missing required parameter 'command'")
	}

	for key, value := range args {
		switch key {
		case "command":
			if _, ok := value.(string); !ok {
				return fmt.Errorf("shell provider validation failed: command must be a string, got %T", value)
			}
			if value.(string) == "" {
				return fmt.Errorf("shell provider validation failed: command cannot be empty")
			}
		case "args":
			if value == nil {
				continue
			}
			list, ok := value.([]interface{})
			if !ok {
				return fmt.Errorf("shell provider validation failed: args must be a list, got %T", value)
			}
			for _, arg := range list {
				if _, ok := arg.(string); !ok {
					return fmt.Errorf("shell provider validation failed: args must be strings, got %T", arg)
				}
			}
		default:
			return fmt.Errorf("shell provider validation failed: unknown parameter '%s'", key)
		}
	}
	return nil
}

// Run runs the provider with the given arguments. Stdout becomes the task
// output; the exit code is exposed as metadata, and stderr is attached to the
// error when the command fails.
func (p *ShellProvider) Run(ctx context.Context, args map[string]interface{}) (*TaskResult, error) {
	command := args["command"].(string)
	var argsSlice []string
	if raw, ok := args["args"].([]interface{}); ok {
		argsSlice = make([]string, len(raw))
		for i, arg := range raw {
			argsSlice[i] = arg.(string)
		}
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, command, argsSlice...) // #nosec: G204
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()

	metadata := map[string]string{}
	if cmd.ProcessState != nil {
		metadata["exit_code"] = strconv.Itoa(cmd.ProcessState.ExitCode())
	}

	if runErr != nil {
		// A killed process reports "signal: killed"; the deadline is the real cause.
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, fmt.Errorf("%w (%v)", ctxErr, runErr)
		}
		msg := strings.TrimSpace(stderr.String())
		if len(msg) > maxStderrInError {
			msg = msg[:maxStderrInError] + "..."
		}
		if msg != "" {
			return nil, fmt.Errorf("%w: %s", runErr, msg)
		}
		return nil, runErr
	}
	return &TaskResult{Output: stdout.String(), Metadata: metadata}, nil
}

// Register registers the shell provider.
func init() {
	Register(&ShellProvider{})
}
