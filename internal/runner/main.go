package runner

import (
	"context"
	"log"

	"github.com/y0anfa/rhino/internal/workflow"
)

func Runner() {
	log.Println("starting runner...")

	workflowsChan := make(chan []workflow.Workflow)

	go WatchWorkflows(workflowsChan)
	go RunWorkflows(context.Background(), workflowsChan)

	select {}
}
