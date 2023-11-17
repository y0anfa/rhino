package models

import (
	"github.com/y0anfa/rhino/internal/providers"
)

type Task struct {
	Description string              `yaml:"description"`
	Name        string                 `yaml:"name"`
	MaxTries    int                    `yaml:"max-tries"`
	Provider    string                 `yaml:"provider"`
	Params      map[string]interface{} `yaml:"params"`
}

func NewTask(name, desc string, provider string, params map[string]interface{}) *Task {

	return &Task{Name: name, Description: desc, Provider: provider, Params: params}
}

func (t *Task) Run() error {
	// Get the provider.
	provider, err := providers.Get(t.Provider)
	if err != nil {
		return err
	}
	// Run the provider.
	err = provider.Run(t.Params)
	if err != nil {
		return err
	}
	return nil
}
