/* Shell Provider inherits from the Provider interface and implements the Name and Run methods.
It also registers itself with the Register function. It is a plugin that allows the user to run shell commands. */
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

// Run runs the provider with the given arguments.
func (p *ShellProvider) Run(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("no command specified")
	}
	cmd := exec.Command(args[0], args[1:]...) // #nosec: G204
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Register registers the shell provider.
func init() {
	Register(&ShellProvider{})
}