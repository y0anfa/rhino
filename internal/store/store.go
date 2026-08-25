package store

import (
	"encoding/json"
	"time"
)

type RunStatus string

const (
	RunStatusRunning RunStatus = "running"
	RunStatusSuccess RunStatus = "success"
	RunStatusFailed  RunStatus = "failed"
)

type TaskStatus string

const (
	TaskStatusRunning TaskStatus = "running"
	TaskStatusSuccess TaskStatus = "success"
	TaskStatusFailed  TaskStatus = "failed"
	TaskStatusSkipped TaskStatus = "skipped"
)

type WorkflowRun struct {
	ID           string    `json:"id"`
	WorkflowName string    `json:"workflow_name"`
	WorkflowHash string    `json:"workflow_hash"`
	WorkflowYAML string    `json:"workflow_yaml,omitempty"`
	Status       RunStatus `json:"status"`
	TriggerType  string    `json:"trigger_type"`
	StartedAt    time.Time `json:"started_at"`
	CompletedAt  time.Time `json:"completed_at,omitempty"`
	Error        string    `json:"error,omitempty"`
}

type TaskExecution struct {
	ID         string     `json:"id"`
	RunID      string     `json:"run_id"`
	TaskName   string     `json:"task_name"`
	Provider   string     `json:"provider"`
	Status     TaskStatus `json:"status"`
	StartedAt  time.Time  `json:"started_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	Output     string     `json:"output,omitempty"`
	Error      string     `json:"error,omitempty"`
	Retries    int        `json:"retries"`
	DurationMs int64      `json:"duration_ms"`
}

// MarshalJSON omits an unset completion time instead of emitting year 1.
func (r WorkflowRun) MarshalJSON() ([]byte, error) {
	type alias WorkflowRun
	return json.Marshal(struct {
		alias
		CompletedAt *time.Time `json:"completed_at,omitempty"`
	}{alias: alias(r), CompletedAt: optionalTime(r.CompletedAt)})
}

// MarshalJSON omits an unset completion time instead of emitting year 1.
func (e TaskExecution) MarshalJSON() ([]byte, error) {
	type alias TaskExecution
	return json.Marshal(struct {
		alias
		CompletedAt *time.Time `json:"completed_at,omitempty"`
	}{alias: alias(e), CompletedAt: optionalTime(e.CompletedAt)})
}

func optionalTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

type RunFilter struct {
	WorkflowName string
	Status       RunStatus
	Since        time.Duration
	Limit        int
}

type Store interface {
	SaveRun(run *WorkflowRun) error
	UpdateRun(run *WorkflowRun) error
	GetRun(id string) (*WorkflowRun, error)
	ListRuns(filter RunFilter) ([]*WorkflowRun, error)
	SaveTaskExecution(exec *TaskExecution) error
	GetTaskExecutions(runID string) ([]*TaskExecution, error)
	DeleteRunsBefore(before time.Time) (int64, error)
	Close() error
}
