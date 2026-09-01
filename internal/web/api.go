package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/y0anfa/rhino/internal/logger"
	"github.com/y0anfa/rhino/internal/models"
	"github.com/y0anfa/rhino/internal/store"
	"go.uber.org/zap"
)

type Server struct {
	httpServer *http.Server
	startTime  time.Time
}

func NewServer(port int) *Server {
	s := &Server{
		httpServer: &http.Server{
			Addr: fmt.Sprintf(":%d", port),
		},
		startTime: time.Now(),
	}
	s.httpServer.Handler = s.Handler()
	return s
}

// Handler returns the dashboard's HTTP routes, so they can be served (and
// tested) without binding a port.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/workflows", s.handleWorkflows)
	mux.HandleFunc("/api/workflows/", s.handleWorkflowDetail)
	mux.HandleFunc("/api/runs", s.handleRuns)
	mux.HandleFunc("/api/runs/", s.handleRunDetail)
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/", s.handleDashboard)
	return mux
}

func (s *Server) Start() error {
	logger.Info("starting web dashboard", zap.String("addr", s.httpServer.Addr))
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("web dashboard error", zap.Error(err))
		}
	}()
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) handleWorkflows(w http.ResponseWriter, r *http.Request) {
	workflows, err := models.ListWorkflows()
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, workflows)
}

// workflowSummary is the API view of a workflow definition.
type workflowSummary struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Trigger     models.Trigger  `json:"trigger"`
	Settings    models.Settings `json:"settings"`
	Tasks       []taskSummary   `json:"tasks"`
	Order       [][]string      `json:"order"`
}

type taskSummary struct {
	Name            string   `json:"name"`
	Description     string   `json:"description,omitempty"`
	Provider        string   `json:"provider"`
	Condition       string   `json:"condition,omitempty"`
	ContinueOnError bool     `json:"continue_on_error,omitempty"`
	DependsOn       []string `json:"depends_on,omitempty"`
}

// handleWorkflowDetail serves GET /api/workflows/{name} and
// POST /api/workflows/{name}/run.
func (s *Server) handleWorkflowDetail(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/workflows/")
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		jsonError(w, "workflow name required", http.StatusBadRequest)
		return
	}
	name := parts[0]

	wf, err := models.LoadWorkflow(name)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			jsonError(w, fmt.Sprintf("workflow '%s' not found", name), http.StatusNotFound)
			return
		}
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	switch {
	case len(parts) == 1 && r.Method == http.MethodGet:
		if err := wf.Validate(); err != nil {
			jsonError(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		jsonResponse(w, summarize(wf))
	case len(parts) == 2 && parts[1] == "run":
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.triggerRun(w, r, wf)
	default:
		jsonError(w, "not found", http.StatusNotFound)
	}
}

func summarize(wf models.Workflow) workflowSummary {
	out := workflowSummary{
		Name:        wf.Name,
		Description: wf.Description,
		Trigger:     wf.Trigger,
		Settings:    wf.Settings,
		Order:       wf.Order,
		Tasks:       make([]taskSummary, 0, len(wf.Tasks)),
	}
	for _, t := range wf.Tasks {
		out.Tasks = append(out.Tasks, taskSummary{
			Name:            t.Name,
			Description:     t.Description,
			Provider:        t.Provider,
			Condition:       t.Condition,
			ContinueOnError: t.ContinueOnError,
			DependsOn:       t.DependsOn,
		})
	}
	return out
}

// triggerRun starts a workflow run. By default it returns 202 with the run ID
// immediately; with ?wait=true it blocks and reports the outcome.
func (s *Server) triggerRun(w http.ResponseWriter, r *http.Request, wf models.Workflow) {
	if err := wf.Validate(); err != nil {
		jsonError(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	runID := store.NewID()
	wait := r.URL.Query().Get("wait") == "true"

	if wait {
		ctx := models.WithRunID(r.Context(), runID)
		_, err := wf.Run(ctx)
		if errors.Is(err, models.ErrTooManyRuns) {
			jsonError(w, err.Error(), http.StatusTooManyRequests)
			return
		}
		resp := map[string]interface{}{"workflow": wf.Name, "run_id": runID, "status": store.RunStatusSuccess}
		if err != nil {
			resp["status"] = store.RunStatusFailed
			resp["error"] = err.Error()
		}
		jsonResponse(w, resp)
		return
	}

	ctx := models.WithRunID(context.WithoutCancel(r.Context()), runID)
	go func() {
		if _, err := wf.Run(ctx); err != nil {
			logger.Error("workflow execution failed", zap.String("workflow", wf.Name), zap.Error(err))
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"workflow": wf.Name,
		"run_id":   runID,
		"status":   "triggered",
	})
}

func (s *Server) handleRuns(w http.ResponseWriter, r *http.Request) {
	st := store.Global()
	if st == nil {
		jsonError(w, "store not available", http.StatusServiceUnavailable)
		return
	}

	filter := store.RunFilter{Limit: 100}
	if wf := r.URL.Query().Get("workflow"); wf != "" {
		filter.WorkflowName = wf
	}
	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = store.RunStatus(status)
	}
	if since := r.URL.Query().Get("since"); since != "" {
		d, err := time.ParseDuration(since)
		if err != nil {
			jsonError(w, fmt.Sprintf("invalid since duration '%s': %v", since, err), http.StatusBadRequest)
			return
		}
		filter.Since = d
	}
	if limit := r.URL.Query().Get("limit"); limit != "" {
		n, err := fmt.Sscanf(limit, "%d", &filter.Limit)
		if err != nil || n != 1 || filter.Limit <= 0 {
			jsonError(w, fmt.Sprintf("invalid limit '%s'", limit), http.StatusBadRequest)
			return
		}
	}

	runs, err := st.ListRuns(filter)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if runs == nil {
		runs = []*store.WorkflowRun{}
	}
	jsonResponse(w, runs)
}

func (s *Server) handleRunDetail(w http.ResponseWriter, r *http.Request) {
	st := store.Global()
	if st == nil {
		jsonError(w, "store not available", http.StatusServiceUnavailable)
		return
	}

	// Extract run ID from /api/runs/{id}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/runs/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		jsonError(w, "run ID required", http.StatusBadRequest)
		return
	}
	runID := parts[0]

	run, err := st.GetRun(runID)
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	tasks, err := st.GetTaskExecutions(runID)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if tasks == nil {
		tasks = []*store.TaskExecution{}
	}

	jsonResponse(w, map[string]interface{}{
		"run":   run,
		"tasks": tasks,
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	workflows, err := models.ListWorkflows()
	status := "healthy"
	if err != nil {
		status = "degraded"
	}
	jsonResponse(w, map[string]interface{}{
		"status":    status,
		"uptime":    time.Since(s.startTime).String(),
		"workflows": len(workflows),
	})
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(dashboardHTML))
}

func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
