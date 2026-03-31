package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
	mux := http.NewServeMux()
	s := &Server{
		httpServer: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		},
		startTime: time.Now(),
	}

	mux.HandleFunc("/api/workflows", s.handleWorkflows)
	mux.HandleFunc("/api/runs", s.handleRuns)
	mux.HandleFunc("/api/runs/", s.handleRunDetail)
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/", s.handleDashboard)

	return s
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

	runs, err := st.ListRuns(filter)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
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

	tasks, _ := st.GetTaskExecutions(runID)

	jsonResponse(w, map[string]interface{}{
		"run":   run,
		"tasks": tasks,
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	workflows, _ := models.ListWorkflows()
	jsonResponse(w, map[string]interface{}{
		"status":    "healthy",
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
