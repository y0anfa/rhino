package models

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/y0anfa/rhino/internal/config"
	"github.com/y0anfa/rhino/internal/logger"
	"github.com/y0anfa/rhino/internal/providers"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type Workflow struct {
	Name        string     `yaml:"name"`
	Description string     `yaml:"description"`
	Settings    Settings   `yaml:"settings"`
	Trigger     Trigger    `yaml:"trigger"`
	Tasks       []Task     `yaml:"tasks"`
	Order       [][]string `yaml:"order"`
}

func NewWorkflow(name string, desc string) *Workflow {
	return &Workflow{Name: name, Description: desc, Settings: *NewSettings(MaxTriesDefault, TimeoutDefault)}
}

func DeleteWorkflow(name string) error {
	dir := config.GetString("workflows-dir")
	return os.Remove(filepath.Join(dir, name+".yaml"))
}

func (w *Workflow) Describe() string {
	desc := "Workflow: " + w.Name + "\n"
	desc += "Description: " + w.Description + "\n"
	desc += "\nSettings:\n"
	desc += fmt.Sprintf("  Max Tries: %d\n", w.Settings.MaxTries)
	desc += fmt.Sprintf("  Timeout: %s\n", w.Settings.Timeout)
	desc += "\nTrigger:\n"
	desc += fmt.Sprintf("  Name: %s\n", w.Trigger.Name)
	desc += fmt.Sprintf("  Type: %s\n", w.Trigger.Type)
	if w.Trigger.Schedule != "" {
		desc += fmt.Sprintf("  Schedule: %s\n", w.Trigger.Schedule)
	}
	desc += "\nTasks:\n"
	for _, t := range w.Tasks {
		desc += fmt.Sprintf("  - %s (provider: %s)\n", t.Name, t.Provider)
		if t.Description != "" {
			desc += fmt.Sprintf("    Description: %s\n", t.Description)
		}
		for k, v := range t.Params {
			desc += fmt.Sprintf("    %s: %v\n", k, v)
		}
	}
	desc += "\nOrder:\n"
	for i, group := range w.Order {
		desc += fmt.Sprintf("  %d: %v\n", i+1, group)
	}
	return desc
}

func ListWorkflows() ([]string, error) {
	dir := config.GetString("workflows-dir")

	var workflows []string
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		workflows = append(workflows, strings.Split(f.Name(), ".")[0])
	}
	return workflows, nil
}

func (w *Workflow) SetTrigger(trigger Trigger) {
	w.Trigger = trigger
}

func (w *Workflow) AddTask(task Task) {
	w.Tasks = append(w.Tasks, task)
}

func (w *Workflow) RemoveTask(task Task) (string, error) {
	for i, t := range w.Tasks {
		if t.Name == task.Name {
			w.Tasks = append(w.Tasks[:i], w.Tasks[i+1:]...)
			return t.Name, nil
		}
	}
	return "", fmt.Errorf("task %s not found", task.Name)
}

func (w *Workflow) GetTask(name string) *Task {
	for i := range w.Tasks {
		if w.Tasks[i].Name == name {
			return &w.Tasks[i]
		}
	}
	return nil
}

func (w *Workflow) Validate() error {
	if w.Name == "" {
		return fmt.Errorf("workflow validation failed: name is empty")
	}
	if w.Settings.MaxTries <= 0 {
		return fmt.Errorf("workflow validation failed: max tries must be greater than 0, got %d", w.Settings.MaxTries)
	}
	if w.Settings.Timeout == "" {
		return fmt.Errorf("workflow validation failed: timeout is empty")
	}
	if _, err := time.ParseDuration(w.Settings.Timeout); err != nil {
		return fmt.Errorf("workflow validation failed: invalid timeout format '%s': %w", w.Settings.Timeout, err)
	}
	if w.Trigger.Name == "" {
		return fmt.Errorf("workflow validation failed: trigger name is empty")
	}
	if w.Trigger.Type == "" {
		return fmt.Errorf("workflow validation failed: trigger type is empty")
	}
	if w.Trigger.Type == TriggerScheduled && w.Trigger.Schedule == "" {
		return fmt.Errorf("workflow validation failed: trigger schedule is empty for cron trigger")
	}
	if w.Trigger.Type == TriggerScheduled {
		if _, err := cron.ParseStandard(w.Trigger.Schedule); err != nil {
			return fmt.Errorf("workflow validation failed: invalid cron schedule '%s': %w", w.Trigger.Schedule, err)
		}
	}
	if len(w.Tasks) == 0 {
		return fmt.Errorf("workflow validation failed: tasks list is empty")
	}
	for _, t := range w.Tasks {
		if t.Name == "" {
			return fmt.Errorf("workflow validation failed: task name is empty")
		}
		if t.Provider == "" {
			return fmt.Errorf("workflow validation failed: task '%s' provider is empty", t.Name)
		}
		if len(t.Params) == 0 {
			return fmt.Errorf("workflow validation failed: task '%s' params are empty", t.Name)
		}
		// Validate task provider
		provider, err := providers.Get(t.Provider)
		if err != nil {
			return fmt.Errorf("workflow validation failed: task '%s' has unknown provider '%s': %w", t.Name, t.Provider, err)
		}
		if err := provider.Validate(t.Params); err != nil {
			return fmt.Errorf("workflow validation failed: task '%s' validation failed: %w", t.Name, err)
		}
	}
	if len(w.Order) == 0 {
		return fmt.Errorf("workflow validation failed: order is empty")
	}
	for _, group := range w.Order {
		if len(group) == 0 {
			return fmt.Errorf("workflow validation failed: order group is empty")
		}
		for _, taskName := range group {
			task := w.GetTask(taskName)
			if task == nil {
				return fmt.Errorf("workflow validation failed: task '%s' not found in order", taskName)
			}
		}
	}
	return nil
}

func (w *Workflow) Save() error {
	dir := config.GetString("workflows-dir")
	path := filepath.Join(dir, w.Name+".yaml")

	data, err := yaml.Marshal(w)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	logger.Info("saving workflow", zap.String("path", path))
	_, err = file.Write(data)
	return err
}

func LoadWorkflow(name string) (Workflow, error) {
	dir := config.GetString("workflows-dir")
	path := filepath.Join(dir, name+".yaml")

	file, err := os.ReadFile(path)
	if err != nil {
		return Workflow{}, err
	}

	var workflow Workflow
	if err = yaml.Unmarshal(file, &workflow); err != nil {
		logger.Error("error decoding workflow", zap.String("workflow", name), zap.Error(err))
		return Workflow{}, err
	}
	return workflow, nil
}

func LoadWorkflows() ([]Workflow, error) {
	var workflows []Workflow
	workflowsList, err := ListWorkflows()
	if err != nil {
		return nil, err
	}
	for _, w := range workflowsList {
		workflow, err := LoadWorkflow(w)
		if err != nil {
			return nil, err
		}
		err = workflow.Validate()
		if err != nil {
			logger.Fatal("workflow is invalid", zap.String("workflow", workflow.Name), zap.Error(err))
		}
		workflows = append(workflows, workflow)
	}
	return workflows, nil
}

func (w *Workflow) Run() error {
	for _, group := range w.Order {
		var wg sync.WaitGroup
		var mu sync.Mutex
		var errs []error

		for _, taskName := range group {
			task := w.GetTask(taskName)
			if task.MaxTries == 0 {
				task.MaxTries = w.Settings.MaxTries
			}
			wg.Add(1)

			go func(t *Task) {
				defer wg.Done()
				var err error
				for try := 0; try < t.MaxTries; try++ {
					timeout, err := time.ParseDuration(w.Settings.Timeout)
					if err != nil {
						logger.Error("workflow execution failed: invalid timeout format", zap.String("timeout", w.Settings.Timeout), zap.Error(err))
						break
					}
					ctx, cancel := context.WithTimeout(context.Background(), timeout)

					errChan := make(chan error, 1)
					go func() {
						errChan <- t.Run()
					}()

					select {
					case <-ctx.Done():
						err = ctx.Err()
						logger.Error("task execution failed: timeout reached", zap.String("task", t.Name), zap.Error(err))
					case err = <-errChan:
						if err != nil {
							logger.Error("task execution failed", zap.String("task", t.Name), zap.Error(err))
						} else {
							logger.Info("task execution succeeded", zap.String("task", t.Name))
						}
					}

					cancel()

					if err == nil {
						break
					}
				}
				if err != nil {
					logger.Error("task execution failed: max retries reached", zap.String("task", t.Name), zap.Error(err))
					mu.Lock()
					errs = append(errs, fmt.Errorf("task '%s' failed: %w", t.Name, err))
					mu.Unlock()
				}
			}(task)
		}

		wg.Wait()

		if len(errs) > 0 {
			return fmt.Errorf("workflow '%s' failed: %v", w.Name, errs)
		}
	}

	return nil
}
