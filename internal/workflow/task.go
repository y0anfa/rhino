package workflow

import "github.com/y0anfa/rhino/internal/providers"

type Task struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	MaxTries    int      `yaml:"max-tries"`
	Provider    string   `yaml:"provider"`
	Command     []string `yaml:"command"`
}

func NewTask(name, desc string, provider string, command []string) *Task {

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
