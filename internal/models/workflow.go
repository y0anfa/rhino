package models

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/y0anfa/rhino/internal/config"
	"github.com/y0anfa/rhino/internal/logger"
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
	err := os.Remove(dir + "/" + name + ".yaml")
	if err != nil {
		return err
	} else {
		return nil
	}
}

func (w *Workflow) Describe() string {
	desc := "Workflow: " + w.Name + "\n"
	desc += "Description: " + w.Description + "\n"
	desc += "Trigger: " + w.Trigger.Name + "\n"
	desc += "Tasks:\n"
	for _, t := range w.Tasks {
		desc += "  - " + t.Name + "\n"
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
	for _, t := range w.Tasks {
		if t.Name == name {
			return &t
		}
	}
	return nil
}

func (w *Workflow) Validate() error {
	if w.Name == "" {
		return fmt.Errorf("workflow name is empty")
	}
	if w.Settings.MaxTries <= 0 {
		return fmt.Errorf("max tries is invalid")
	}
	if w.Settings.Timeout == "" {
		return fmt.Errorf("timeout is empty")
	}
	if w.Trigger.Name == "" {
		return fmt.Errorf("trigger name is empty")
	}
	if w.Trigger.Type == "" {
		return fmt.Errorf("trigger type is empty")
	}
	if w.Trigger.Type == TriggerScheduled && w.Trigger.Schedule == "" {
		return fmt.Errorf("trigger schedule is empty")
	}
	if len(w.Tasks) == 0 {
		return fmt.Errorf("tasks are empty")
	}
	for _, t := range w.Tasks {
		if t.Name == "" {
			return fmt.Errorf("task name is empty")
		}
		if t.Provider == "" {
			return fmt.Errorf("task %s provider is empty", t.Name)
		}
		if len(t.Params) == 0 {
			return fmt.Errorf("task %s command is empty", t.Name)
		}
	}
	if len(w.Order) == 0 {
		return fmt.Errorf("order is empty")
	}
	for _, group := range w.Order {
		if len(group) == 0 {
			return fmt.Errorf("order group is empty")
		}
		for _, taskName := range group {
			task := w.GetTask(taskName)
			if task == nil {
				return fmt.Errorf("task %s not found", taskName)
			}
		}
	}
	return nil
}

func (w *Workflow) Save() error {
	dir := config.GetString("workflows-dir")

	data, err := yaml.Marshal(w)
	if err != nil {
		return err
	} else {
		file, err := os.OpenFile(filepath.Clean(dir+"/"+w.Name+".yaml"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer file.Close()
		logger.Info("saving workflow to ", filepath.Clean(dir+"/"+w.Name+".yaml"))
		_, err = file.Write(data)
		if err != nil {
			return err
		} else {
			return nil
		}
	}
}

func LoadWorkflow(name string) (Workflow, error) {
	dir := config.GetString("workflows-dir")
	
	file, err := os.ReadFile(filepath.Clean(dir + "/" + name + ".yaml"))
	if err != nil {
		return Workflow{}, err
	} else {
		var workflow Workflow
		err = yaml.Unmarshal(file, &workflow)
		if err != nil {
			logger.Error("error decoding workflow ", name, zap.Error(err))
			return Workflow{}, err
		} else {
			return workflow, nil
		}
	}
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
			logger.Fatal("workflow is invalid ", workflow.Name, zap.Error(err))
		}
		workflows = append(workflows, workflow)
	}
	return workflows, nil
}

func (w *Workflow) Run() error {
	for _, group := range w.Order {
		var wg sync.WaitGroup

		for _, taskName := range group {
			task := w.GetTask(taskName)
			if task.MaxTries == 0 {
				task.MaxTries = w.Settings.MaxTries
			}
			wg.Add(1)

			go func(t *Task) {
				defer wg.Done()
				var err error
				for try := 0; try < task.MaxTries; try++ {
					timeout, err := time.ParseDuration(w.Settings.Timeout)
					if err != nil {
						logger.Error("invalid timeout ", w.Settings.Timeout, zap.Error(err))
						break
					}
					ctx, cancel := context.WithTimeout(context.Background(), timeout)
					defer cancel()

					errChan := make(chan error, 1)
					go func() {
						errChan <- t.Run()
					}()

					select {
					case <-ctx.Done():
						err = ctx.Err()
						logger.Error("task timed out ", t.Name, zap.Error(err))
					case err = <-errChan:
						if err != nil {
							logger.Error("task failed ", t.Name, zap.Error(err))
						} else {
							logger.Info("task succeeded ", t.Name)
						}
					}

					if err == nil {
						break
					}
				}
				if err != nil {
					logger.Error("task reached max tries ", t.Name, zap.Error(err))
				}
			}(task)
		}

		wg.Wait()
	}

	return nil
}
