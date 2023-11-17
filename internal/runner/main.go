package runner

import (
	"context"

	"github.com/y0anfa/rhino/internal/logger"
	"github.com/y0anfa/rhino/internal/models"
)

func Runner() {
	logger.Info("starting runner...")

	workflowsChan := make(chan []models.Workflow)

	go WatchWorkflows(workflowsChan)
	go RunWorkflows(context.Background(), workflowsChan)

	select {}
}
