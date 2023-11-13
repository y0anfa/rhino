package runner

import (
	"fmt"
	"log"

	"github.com/fsnotify/fsnotify"
	"github.com/y0anfa/rhino/internal/workflow"
)

func WatchWorkflows(workflowsChan chan<- []workflow.Workflow) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	err = watcher.Add("workflows")
	if err != nil {
		return err
	}

	workflows, err := workflow.LoadWorkflows("workflows")
	if err != nil {
		return err
	}

	workflowsChan <- workflows

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return fmt.Errorf("watcher event channel closed")
			}

			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Println("modified file:", event.Name)
				workflows, err = workflow.LoadWorkflows("workflows")
				if err != nil {
					return err
				}

				workflowsChan <- workflows
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return fmt.Errorf("watcher error channel closed")
			}
			return err
		}
	}
}