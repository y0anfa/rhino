package runner

import (
	"context"
	"log"
	"net/http"
	"sync"

	"github.com/robfig/cron/v3"
	"github.com/y0anfa/rhino/internal/workflow"
)

var (
	mux 	 = http.NewServeMux()
	muxSetup sync.Once
)

func RunWorkflows(ctx context.Context, workflowsChan <-chan []workflow.Workflow) {
	activeWorkflows := make(map[string]context.CancelFunc)

	for workflows := range workflowsChan {
		for name, cancel := range activeWorkflows {
			found := false
			for _, wf := range workflows {
				if name == wf.Name {
					found = true
					break
				}
			}
			if !found {
				log.Println("stopping workflow", name)
				cancel()
				delete(activeWorkflows, name)
			}
		}

		for _, wf := range workflows {
			if _, ok := activeWorkflows[wf.Name]; !ok {
				log.Println("starting workflow", wf.Name)
				ctx, cancel := context.WithCancel(ctx)
				activeWorkflows[wf.Name] = cancel
				switch wf.Trigger.Type {
				case workflow.TriggerScheduled:
					go RunScheduledWorkflow(ctx, wf)
				case workflow.TriggerWebhook:
					go RunWebhookWorkflow(ctx, wf)
				default:
					log.Printf("invalid trigger type %s on workflow %s", wf.Trigger.Name, wf.Name)
				}
			}
		}
	}
}

func RunScheduledWorkflow(ctx context.Context, wf workflow.Workflow) {
	c := cron.New()
	_, err := c.AddFunc(wf.Trigger.Schedule, func() {
		select {
		case <-ctx.Done():
			log.Printf("stopping workflow %s", wf.Name)
			return
		default:
			log.Printf("running workflow %s", wf.Name)
			err := wf.Run()
			if err != nil {
				log.Printf("error running workflow %s: %s", wf.Name, err)
			}
		}
	})

	if err != nil {
		log.Printf("error scheduling workflow %s: %s", wf.Name, err)
	}

	c.Start()
	log.Println("started cron on workflow", wf.Name)

	<-ctx.Done()
	log.Println("stopping cron on workflow", wf.Name)
	c.Stop()
}

func RunWebhookWorkflow(ctx context.Context, wf workflow.Workflow) {
	muxSetup.Do(func() {
		go func() {
			log.Fatal(http.ListenAndServe(":8888", mux))
		}()
	})

	mux.HandleFunc("/webhook/"+wf.Name, func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-ctx.Done():
			log.Printf("stopping workflow %s", wf.Name)
			return
		default:
			log.Printf("running workflow %s", wf.Name)
			err := wf.Run()
			if err != nil {
				log.Printf("error running workflow %s: %s", wf.Name, err)
			}
		}
	})
}
