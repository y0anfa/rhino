package store

import (
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestSaveRun_CompletedAtNullWhileRunning(t *testing.T) {
	s := testStore(t)

	run := &WorkflowRun{
		ID:           "run-running",
		WorkflowName: "wf",
		Status:       RunStatusRunning,
		TriggerType:  "manual",
		StartedAt:    time.Now(),
	}
	if err := s.SaveRun(run); err != nil {
		t.Fatalf("SaveRun failed: %v", err)
	}

	var completedAt sql.NullTime
	if err := s.db.QueryRow(`SELECT completed_at FROM workflow_runs WHERE id = ?`, run.ID).Scan(&completedAt); err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if completedAt.Valid {
		t.Errorf("expected NULL completed_at for a running run, got %s", completedAt.Time)
	}

	got, err := s.GetRun(run.ID)
	if err != nil {
		t.Fatalf("GetRun failed: %v", err)
	}
	data, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "0001-01-01") {
		t.Errorf("running run should not serialize a year-1 completion time: %s", data)
	}
}

func TestSaveTaskExecution_CompletedAtNullWhileRunning(t *testing.T) {
	s := testStore(t)

	exec := &TaskExecution{
		ID:        "task-running",
		RunID:     "run-1",
		TaskName:  "t1",
		Provider:  "shell",
		Status:    TaskStatusRunning,
		StartedAt: time.Now(),
	}
	if err := s.SaveTaskExecution(exec); err != nil {
		t.Fatalf("SaveTaskExecution failed: %v", err)
	}

	var completedAt sql.NullTime
	if err := s.db.QueryRow(`SELECT completed_at FROM task_executions WHERE id = ?`, exec.ID).Scan(&completedAt); err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if completedAt.Valid {
		t.Errorf("expected NULL completed_at for a running task, got %s", completedAt.Time)
	}

	data, err := json.Marshal(exec)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "0001-01-01") {
		t.Errorf("running task should not serialize a year-1 completion time: %s", data)
	}
}
