package models

import (
	"fmt"

	"github.com/y0anfa/rhino/internal/providers"
)

type Task struct {
	Description string                 `yaml:"description"`
	Name        string                 `yaml:"name"`
	MaxTries    int                    `yaml:"max-tries"`
	Provider    string                 `yaml:"provider"`
	Params      map[string]interface{} `yaml:"params"`
}

func NewTask(name, desc string, provider string, params map[string]interface{}) *Task {

	return &Task{Name: name, Description: desc, Provider: provider, Params: params}
}

func (t *Task) Run() (*providers.TaskResult, error) {
	provider, err := providers.Get(t.Provider)
	if err != nil {
		return nil, fmt.Errorf("task execution failed: unknown provider '%s': %w", t.Provider, err)
	}
	err = provider.Validate(t.Params)
	if err != nil {
		return nil, fmt.Errorf("task execution failed: validation failed for task '%s': %w", t.Name, err)
	}
	result, err := provider.Run(t.Params)
	if err != nil {
		return nil, fmt.Errorf("task execution failed: provider '%s' failed for task '%s': %w", t.Provider, t.Name, err)
	}
	return result, nil
}
