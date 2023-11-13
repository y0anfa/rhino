package workflow

import "github.com/y0anfa/rhino/internal/providers"

type Task struct {
	Name        string
	Description string
	Provider    string
	Command     []string
}

func NewTask(name, desc string, parent *Task, provider string, command []string) *Task {
	return &Task{Name: name, Description: desc, Provider: provider, Command: command}
}

func (t *Task) Run() error {
	// Get the provider.
	provider, err := providers.Get(t.Provider)
	if err != nil {
		return err
	}
	// Run the provider.
	return provider.Run(t.Command)
}
