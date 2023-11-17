package runner

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/y0anfa/rhino/internal/config"
	"github.com/y0anfa/rhino/internal/logger"
	"github.com/y0anfa/rhino/internal/models"
)

var (
	mux 	 = http.NewServeMux()
	muxSetup sync.Once
)

func RunWorkflows(ctx context.Context, workflowsChan <-chan []models.Workflow) {
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
				logger.Info("workflow stopping ", name)
				cancel()
				delete(activeWorkflows, name)
			}
		}

		for _, wf := range workflows {
			if _, ok := activeWorkflows[wf.Name]; !ok {
				logger.Info("workflow starting ", wf.Name)
				ctx, cancel := context.WithCancel(ctx)
				activeWorkflows[wf.Name] = cancel
				switch wf.Trigger.Type {
				case models.TriggerScheduled:
					go RunScheduledWorkflow(ctx, wf)
				case models.TriggerWebhook:
					go RunWebhookWorkflow(ctx, wf)
				default:
					logger.Error("invalid trigger type ", wf.Trigger.Name, wf.Name)
				}
			}
		}
	}
}

func RunScheduledWorkflow(ctx context.Context, wf models.Workflow) {
	c := cron.New()
	_, err := c.AddFunc(wf.Trigger.Schedule, func() {
		select {
		case <-ctx.Done():
			logger.Info("workflow stopping ", wf.Name)
			return
		default:
			logger.Info("workflow running ", wf.Name)
			err := wf.Run()
			if err != nil {
				logger.Error("error running workflow ", wf.Name, err)
			}
		}
	})

	if err != nil {
		logger.Error("error scheduling workflow ", wf.Name, err)
	}

	c.Start()
	logger.Info("workflow scheduling started ", wf.Name)

	<-ctx.Done()
	logger.Info("workflow scheduling stopped ", wf.Name)
	c.Stop()
}

func RunWebhookWorkflow(ctx context.Context, wf models.Workflow) {
	muxSetup.Do(func() {
		go func() {
			server := &http.Server{
				Addr: ":" + strconv.Itoa(config.GetInt("port")), 
				Handler: mux,
				ReadHeaderTimeout: 3 * time.Second,
			}
			err := server.ListenAndServe()
			if err != nil {
				logger.Fatal(err)
			}
		}()
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/webhook/"+wf.Name, func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-ctx.Done():
			logger.Info("workflow starting ", wf.Name)
			return
		default:
			logger.Info("workflow running ", wf.Name)
			err := wf.Run()
			if err != nil {
				logger.Error("error running workflow ", wf.Name, err)
			}
		}
	})
}
