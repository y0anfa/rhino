package runner

import (
	"context"

	"github.com/y0anfa/rhino/internal/logger"
	"github.com/y0anfa/rhino/internal/workflow"
)

func Runner() {
	logger.Info("starting runner...")

	workflowsChan := make(chan []workflow.Workflow)

	go WatchWorkflows(workflowsChan)
	go RunWorkflows(context.Background(), workflowsChan)

	select {}
}
