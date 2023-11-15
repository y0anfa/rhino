package workflow

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/y0anfa/rhino/internal/config"
	"gopkg.in/yaml.v3"
)

type Workflow struct {
	Name        string     `yaml:"name"`
	Description string     `yaml:"description"`
	Trigger     Trigger    `yaml:"trigger"`
	Tasks       []Task 	   `yaml:"tasks"`
	Order	    [][]string `yaml:"order"`
}

func NewWorkflow(name string, desc string) *Workflow {
	return &Workflow{Name: name, Description: desc}
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
	return "", fmt.Errorf("task not found")
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
		if len(t.Command) == 0 {
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
		file, err := os.OpenFile(filepath.Clean(dir+"/"+w.Name+".yaml"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0200)
		if err != nil {
			return err
		}
		_, err = file.Write(data)
		if err != nil {
			return err
		} else {
			return nil
		}
	}
}

func LoadWorkflow(name string) (*Workflow, error) {
	dir := config.GetString("workflows-dir")

	file, err := os.Open(filepath.Clean(dir + "/" + name + ".yaml"))
	if err != nil {
		return nil, err
	} else {
		decoder := yaml.NewDecoder(file)
		var workflow Workflow
		err = decoder.Decode(&workflow)
		if err != nil {
			return nil, err
		} else {
			return &workflow, nil
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
			log.Fatalf("workflow %s is invalid: %s", workflow.Name, err)
		}
		workflows = append(workflows, *workflow)
	}
	return workflows, nil
}

func (w *Workflow) Run() error {
	for _, group := range w.Order {
		var wg sync.WaitGroup

		for _, taskName := range group {
			task := w.GetTask(taskName)
			wg.Add(1)

			go func(t *Task) {
				defer wg.Done()
				err := t.Run()
				if err != nil {
					log.Println(err)
				}
			}(task)
		}

		wg.Wait()
	}

	return nil
}
