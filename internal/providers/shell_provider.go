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
		if args[param] == "" {
			return fmt.Errorf("missing %s parameter", param)
		}
	}

	for key, value := range args {
		switch key {
		case "command":
			if _, ok := value.(string); !ok {
				return fmt.Errorf("invalid command parameter")
			}
		case "args":
			if _, ok := value.([]interface{}); !ok {
				for _, arg := range value.([]interface{}) {
					if _, ok := arg.(string); !ok {
						return fmt.Errorf("invalid args parameter")
					}
				}
			}
		default:
			return fmt.Errorf("unknown parameter: %s", key)
		}
	}
	return nil
}

// Run runs the provider with the given arguments.
func (p *ShellProvider) Run(args map[string]interface{}) error {
	err := p.Validate(args)
	if err != nil {
		return err
	}

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