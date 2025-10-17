package runner

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/robfig/cron/v3"
	"github.com/y0anfa/rhino/internal/config"
	"github.com/y0anfa/rhino/internal/logger"
	"github.com/y0anfa/rhino/internal/models"
	"go.uber.org/zap"
)

type Runner interface {
	Run(ctx context.Context) error
	Stop(ctx context.Context) error
}

type CronRunner struct {
	Workflow models.Workflow
	Scheduler *cron.Cron
}

func (cr *CronRunner) Run(ctx context.Context) error {
	logger.Info("starting cron runner for workflow ", cr.Workflow.Name)
	cr.Scheduler = cron.New(cron.WithSeconds())
	cr.Scheduler.AddFunc(cr.Workflow.Trigger.Schedule, func() {
		cr.Workflow.Run()
	})
	cr.Scheduler.Start()
	return nil
}

func (cr *CronRunner) Stop(ctx context.Context) error {
	logger.Info("stopping cron runner for workflow ", cr.Workflow.Name)
	if cr.Scheduler != nil {
		cr.Scheduler.Stop()
	}
	return nil
}

type WebhookRunner struct {
	Workflow models.Workflow
}

func (wr *WebhookRunner) Run(ctx context.Context) error {
	logger.Info("registering webhook handler for workflow ", wr.Workflow.Name)
	// Register the workflow with the shared webhook server
	RegisterWebhookWorkflow(wr.Workflow)
	return nil
}

func (wr *WebhookRunner) Stop(ctx context.Context) error {
	logger.Info("unregistering webhook handler for workflow ", wr.Workflow.Name)
	UnregisterWebhookWorkflow(wr.Workflow.Name)
	return nil
}

// Shared webhook server
var (
	webhookServer     *http.Server
	webhookMux        *http.ServeMux
	webhookWorkflows  = make(map[string]models.Workflow)
	webhookMutex      sync.RWMutex
	webhookServerOnce sync.Once
)

// RegisterWebhookWorkflow registers a workflow to be triggered by webhook
func RegisterWebhookWorkflow(workflow models.Workflow) {
	webhookMutex.Lock()
	defer webhookMutex.Unlock()

	// Initialize the shared webhook server once
	webhookServerOnce.Do(func() {
		webhookMux = http.NewServeMux()
		port := config.GetInt("port")
		webhookServer = &http.Server{
			Addr:    ":" + strconv.Itoa(port),
			Handler: webhookMux,
		}

		// Register the main handler
		webhookMux.HandleFunc("/", webhookHandler)

		// Start the server in a goroutine
		go func() {
			logger.Info("starting shared webhook server on port ", port)
			if err := webhookServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error("webhook server error", zap.Error(err))
			}
		}()
	})

	// Register the workflow
	path := "/webhook/" + workflow.Name
	webhookWorkflows[workflow.Name] = workflow
	logger.Info("registered webhook for workflow ", workflow.Name, " at path ", path)
}

// UnregisterWebhookWorkflow unregisters a workflow from webhook triggers
func UnregisterWebhookWorkflow(workflowName string) {
	webhookMutex.Lock()
	defer webhookMutex.Unlock()

	delete(webhookWorkflows, workflowName)
	logger.Info("unregistered webhook for workflow ", workflowName)
}

// webhookHandler handles all webhook requests
func webhookHandler(w http.ResponseWriter, r *http.Request) {
	// Extract workflow name from path: /webhook/{workflow-name}
	path := r.URL.Path
	if len(path) < 9 || path[:9] != "/webhook/" {
		http.Error(w, "Invalid webhook path. Use /webhook/{workflow-name}", http.StatusNotFound)
		return
	}

	workflowName := path[9:]
	if workflowName == "" {
		http.Error(w, "Workflow name required. Use /webhook/{workflow-name}", http.StatusBadRequest)
		return
	}

	webhookMutex.RLock()
	workflow, exists := webhookWorkflows[workflowName]
	webhookMutex.RUnlock()

	if !exists {
		http.Error(w, fmt.Sprintf("Workflow '%s' not found", workflowName), http.StatusNotFound)
		return
	}

	logger.Info("webhook triggered for workflow ", workflowName)

	// Run workflow in a goroutine to avoid blocking the HTTP response
	go func() {
		if err := workflow.Run(); err != nil {
			logger.Error("workflow execution failed", zap.String("workflow", workflowName), zap.Error(err))
		}
	}()

	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintf(w, "Workflow '%s' triggered successfully\n", workflowName)
}

// StopWebhookServer stops the shared webhook server
func StopWebhookServer(ctx context.Context) error {
	if webhookServer != nil {
		logger.Info("stopping shared webhook server")
		return webhookServer.Shutdown(ctx)
	}
	return nil
}
