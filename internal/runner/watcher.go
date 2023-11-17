package runner

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/y0anfa/rhino/internal/config"
	"github.com/y0anfa/rhino/internal/logger"
	"github.com/y0anfa/rhino/internal/models"
)

func WatchWorkflows(workflowsChan chan<- []models.Workflow) error {
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

	logger.Info("watching workflows directory: ", dir)
	workflows, err := models.LoadWorkflows()
	if err != nil {
		return err
	}

	logger.Info("loaded workflows: ", workflows)
	workflowsChan <- workflows

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return fmt.Errorf("watcher event channel closed")
			}

			if event.Op&fsnotify.Write == fsnotify.Write {
				logger.Info("file modified", event.Name)
				workflows, err = models.LoadWorkflows()
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