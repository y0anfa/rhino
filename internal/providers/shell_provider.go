/*
	Shell Provider inherits from the Provider interface and implements the Name and Run methods.

It also registers itself with the Register function. It is a plugin that allows the user to run shell commands.
*/
package providers

import (
	"fmt"
	"os"
	"os/exec"
)

// ShellProvider is the shell provider.
type ShellProvider struct{}

// Name returns the name of the provider.
func (p *ShellProvider) Name() string {
	return "shell"
}

// Validate validates the provider arguments.
func (p *ShellProvider) Validate(args map[string]interface{}) error {
	requiredParams := []string{"command", "args"}
	for _, param := range requiredParams {
		if args[param] == nil || args[param] == "" {
			return fmt.Errorf("shell provider validation failed: missing required parameter '%s'", param)
		}
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
			if _, ok := value.([]interface{}); !ok {
				return fmt.Errorf("shell provider validation failed: args must be a list, got %T", value)
			}
			for _, arg := range value.([]interface{}) {
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

// Run runs the provider with the given arguments.
func (p *ShellProvider) Run(args map[string]interface{}) error {
	command := args["command"].(string)
	argsSlice := make([]string, len(args["args"].([]interface{})))
	for i, arg := range args["args"].([]interface{}) {
		argsSlice[i] = arg.(string)
	}

	cmd := exec.Command(command, argsSlice...) // #nosec: G204
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Register registers the shell provider.
func init() {
	Register(&ShellProvider{})
}
