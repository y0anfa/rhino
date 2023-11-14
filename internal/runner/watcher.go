package runner

import (
	"fmt"
	"log"

	"github.com/fsnotify/fsnotify"
	"github.com/y0anfa/rhino/internal/config"
	"github.com/y0anfa/rhino/internal/workflow"
)

func WatchWorkflows(workflowsChan chan<- []workflow.Workflow) error {
	dir := config.GetString("workflows-dir")

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	err = watcher.Add(dir)
	if err != nil {
		return err
	}

	log.Println("watching workflows directory:", dir)
	workflows, err := workflow.LoadWorkflows()
	if err != nil {
		return err
	}

	log.Println("loaded workflows:", workflows)
	workflowsChan <- workflows

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return fmt.Errorf("watcher event channel closed")
			}

			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Println("modified file:", event.Name)
				workflows, err = workflow.LoadWorkflows()
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