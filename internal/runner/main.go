package runner

import (
	"context"
	"log"
	"net/http"

	"github.com/y0anfa/rhino/internal/workflow"
)

func Runner() {
	log.Println("starting runner...")

	workflowsChan := make(chan []workflow.Workflow)

	go WatchWorkflows(workflowsChan)
	go RunWorkflows(context.Background(), workflowsChan)

	http.ListenAndServe(":8080", nil)
}
