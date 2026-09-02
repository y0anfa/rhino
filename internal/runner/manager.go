package runner

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/y0anfa/rhino/internal/models"
)

// WorkflowRunner is a Runner bound to a single workflow.
type WorkflowRunner interface {
	Runner
	WorkflowName() string
}

// NewRunnerFor builds the runner matching the workflow's trigger. Manual
// workflows have no runner and return (nil, nil).
func NewRunnerFor(w models.Workflow) (WorkflowRunner, error) {
	switch w.Trigger.Type {
	case models.TriggerScheduled:
		return &CronRunner{Workflow: w}, nil
	case models.TriggerWebhook:
		return &WebhookRunner{Workflow: w}, nil
	case models.TriggerWatch:
		return &WatchRunner{Workflow: w}, nil
	case models.TriggerManual:
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown trigger type '%s' for workflow '%s'", w.Trigger.Type, w.Name)
	}
}

type RunnerManager struct {
	mu      sync.Mutex
	Runners []Runner
}

func NewRunnerManager() *RunnerManager {
	return &RunnerManager{}
}

func (rm *RunnerManager) AddRunner(r Runner) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.Runners = append(rm.Runners, r)
}

// RemoveRunner forgets a runner without stopping it.
func (rm *RunnerManager) RemoveRunner(r Runner) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	for i, existing := range rm.Runners {
		if existing == r {
			rm.Runners = append(rm.Runners[:i], rm.Runners[i+1:]...)
			return
		}
	}
}

// RunnerFor returns the runner registered for the named workflow, if any.
func (rm *RunnerManager) RunnerFor(workflowName string) WorkflowRunner {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	for _, r := range rm.Runners {
		if wr, ok := r.(WorkflowRunner); ok && wr.WorkflowName() == workflowName {
			return wr
		}
	}
	return nil
}

func (rm *RunnerManager) snapshot() []Runner {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	return append([]Runner(nil), rm.Runners...)
}

func (rm *RunnerManager) Run(ctx context.Context) error {
	var errs []error
	for _, r := range rm.snapshot() {
		if err := r.Run(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (rm *RunnerManager) Stop(ctx context.Context) error {
	var errs []error
	for _, r := range rm.snapshot() {
		if err := r.Stop(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if err := StopWebhookServer(ctx); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
