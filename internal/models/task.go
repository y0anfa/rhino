package models

import (
	"context"
	"fmt"

	"github.com/y0anfa/rhino/internal/providers"
)

type Task struct {
	Description     string                 `yaml:"description"`
	Name            string                 `yaml:"name"`
	MaxTries        int                    `yaml:"max-tries"`
	Timeout         string                 `yaml:"timeout,omitempty"`
	Provider        string                 `yaml:"provider"`
	Params          map[string]interface{} `yaml:"params"`
	Condition       string                 `yaml:"condition,omitempty"`
	OnFailure       string                 `yaml:"on-failure,omitempty"`
	ContinueOnError bool                   `yaml:"continue-on-error,omitempty"`
}

func NewTask(name, desc string, provider string, params map[string]interface{}) *Task {
	return &Task{Name: name, Description: desc, Provider: provider, Params: params}
}

func (t *Task) Run(ctx context.Context) (*providers.TaskResult, error) {
	return t.RunWithParams(ctx, t.Params)
}

func (t *Task) RunWithParams(ctx context.Context, params map[string]interface{}) (*providers.TaskResult, error) {
	provider, err := providers.Get(t.Provider)
	if err != nil {
		return nil, fmt.Errorf("task execution failed: unknown provider '%s': %w", t.Provider, err)
	}
	err = provider.Validate(params)
	if err != nil {
		return nil, fmt.Errorf("task execution failed: validation failed for task '%s': %w", t.Name, err)
	}
	result, err := provider.Run(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("task execution failed: provider '%s' failed for task '%s': %w", t.Provider, t.Name, err)
	}
	return result, nil
}
