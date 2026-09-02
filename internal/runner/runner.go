package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/y0anfa/rhino/internal/config"
	"github.com/y0anfa/rhino/internal/logger"
	"github.com/y0anfa/rhino/internal/models"
	"github.com/y0anfa/rhino/internal/store"
	"go.uber.org/zap"
)

var startTime = time.Now()

// logRunError logs a failed run; a run dropped by max-concurrent-runs is
// expected back-pressure, not a failure.
func logRunError(workflow string, err error) {
	if errors.Is(err, models.ErrTooManyRuns) {
		logger.Warn("workflow run skipped: max concurrent runs reached", zap.String("workflow", workflow), zap.Error(err))
		return
	}
	logger.Error("workflow execution failed", zap.String("workflow", workflow), zap.Error(err))
}

type Runner interface {
	Run(ctx context.Context) error
	Stop(ctx context.Context) error
}

type CronRunner struct {
	Workflow  models.Workflow
	Scheduler *cron.Cron
}

func (cr *CronRunner) Run(ctx context.Context) error {
	logger.Info("starting cron runner", zap.String("workflow", cr.Workflow.Name))
	// Schedules are validated with cron.ParseStandard, so use the same 5-field parser here.
	cr.Scheduler = cron.New()
	if _, err := cr.Scheduler.AddFunc(cr.Workflow.Trigger.Schedule, func() {
		if _, err := cr.Workflow.Run(ctx); err != nil {
			logRunError(cr.Workflow.Name, err)
		}
	}); err != nil {
		return fmt.Errorf("cron runner: invalid schedule '%s' for workflow '%s': %w",
			cr.Workflow.Trigger.Schedule, cr.Workflow.Name, err)
	}
	cr.Scheduler.Start()
	return nil
}

func (cr *CronRunner) WorkflowName() string { return cr.Workflow.Name }

func (cr *CronRunner) Stop(ctx context.Context) error {
	logger.Info("stopping cron runner", zap.String("workflow", cr.Workflow.Name))
	if cr.Scheduler != nil {
		cr.Scheduler.Stop()
	}
	return nil
}

type WebhookRunner struct {
	Workflow models.Workflow
}

func (wr *WebhookRunner) Run(ctx context.Context) error {
	logger.Info("registering webhook handler", zap.String("workflow", wr.Workflow.Name))
	// Register the workflow with the shared webhook server
	RegisterWebhookWorkflow(wr.Workflow)
	return nil
}

func (wr *WebhookRunner) WorkflowName() string { return wr.Workflow.Name }

func (wr *WebhookRunner) Stop(ctx context.Context) error {
	logger.Info("unregistering webhook handler", zap.String("workflow", wr.Workflow.Name))
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

		// Register handlers
		webhookMux.HandleFunc("/health", healthHandler)
		webhookMux.HandleFunc("/", webhookHandler)

		// Start the server in a goroutine
		go func() {
			logger.Info("starting shared webhook server", zap.Int("port", port))
			if err := webhookServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error("webhook server error", zap.Error(err))
			}
		}()
	})

	// Register the workflow
	path := "/webhook/" + workflow.Name
	webhookWorkflows[workflow.Name] = workflow
	logger.Info("registered webhook", zap.String("workflow", workflow.Name), zap.String("path", path))
}

// UnregisterWebhookWorkflow unregisters a workflow from webhook triggers
func UnregisterWebhookWorkflow(workflowName string) {
	webhookMutex.Lock()
	defer webhookMutex.Unlock()

	delete(webhookWorkflows, workflowName)
	logger.Info("unregistered webhook", zap.String("workflow", workflowName))
}

// webhookHandler handles all webhook requests
func webhookHandler(w http.ResponseWriter, r *http.Request) {
	// Triggering a run is a side effect, so only POST is accepted: crawlers,
	// browsers, and health probes must not start workflows.
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "Method not allowed. Use POST /webhook/{workflow-name}", http.StatusMethodNotAllowed)
		return
	}

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

	inputs, err := webhookInputs(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Payloads come from external systems, so undeclared fields are ignored;
	// missing required inputs are still a client error.
	if err := workflow.CheckInputs(inputs, false); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	logger.Info("webhook triggered", zap.String("workflow", workflowName), zap.Int("inputs", len(inputs)))

	// The request context is cancelled as soon as this handler returns, so detach
	// from it while keeping any request-scoped values.
	runCtx := context.WithoutCancel(r.Context())
	runID := store.NewID()
	runCtx = models.WithRunID(runCtx, runID)
	runCtx = models.WithLenientInputs(models.WithInputs(runCtx, inputs))
	go func() {
		if _, err := workflow.Run(runCtx); err != nil {
			logRunError(workflowName, err)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"workflow": workflowName,
		"run_id":   runID,
		"status":   "triggered",
	})
}

// maxWebhookBody bounds how much of a payload is read into inputs.
const maxWebhookBody = 1 << 20

// webhookInputs collects inputs from the query string and, for JSON requests,
// the top-level fields of the body. Body fields win over query parameters.
func webhookInputs(r *http.Request) (map[string]string, error) {
	inputs := make(map[string]string)
	for key, values := range r.URL.Query() {
		if len(values) > 0 {
			inputs[key] = values[0]
		}
	}

	if r.Body == nil {
		return inputs, nil
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBody+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read webhook body: %w", err)
	}
	if len(body) > maxWebhookBody {
		return nil, fmt.Errorf("webhook body exceeds %d bytes", maxWebhookBody)
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return inputs, nil
	}
	if ct := r.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(ct, "application/json") {
		// Non-JSON bodies are exposed whole so a workflow can still consume them.
		inputs["body"] = string(body)
		return inputs, nil
	}
	fromBody, err := models.InputsFromJSON(body)
	if err != nil {
		return nil, err
	}
	for k, v := range fromBody {
		inputs[k] = v
	}
	return inputs, nil
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	webhookMutex.RLock()
	workflowCount := len(webhookWorkflows)
	webhookMutex.RUnlock()

	health := map[string]interface{}{
		"status":    "healthy",
		"uptime":    time.Since(startTime).String(),
		"workflows": workflowCount,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// StopWebhookServer stops the shared webhook server
func StopWebhookServer(ctx context.Context) error {
	if webhookServer != nil {
		logger.Info("stopping shared webhook server")
		return webhookServer.Shutdown(ctx)
	}
	return nil
}
