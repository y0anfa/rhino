package store

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func testStore(t *testing.T) *SQLiteStore {
	t.Helper()
	dir := t.TempDir()
	s, err := NewSQLiteStore(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestSaveAndGetRun(t *testing.T) {
	s := testStore(t)
	now := time.Now().Truncate(time.Second)

	run := &WorkflowRun{
		ID:           "run-001",
		WorkflowName: "test-wf",
		WorkflowHash: "abc123",
		WorkflowYAML: "name: test-wf",
		Status:       RunStatusRunning,
		TriggerType:  "manual",
		StartedAt:    now,
	}

	if err := s.SaveRun(run); err != nil {
		t.Fatalf("SaveRun failed: %v", err)
	}

	got, err := s.GetRun("run-001")
	if err != nil {
		t.Fatalf("GetRun failed: %v", err)
	}
	if got.WorkflowName != "test-wf" {
		t.Errorf("expected workflow_name=test-wf, got %s", got.WorkflowName)
	}
	if got.Status != RunStatusRunning {
		t.Errorf("expected status=running, got %s", got.Status)
	}
	if got.WorkflowYAML != "name: test-wf" {
		t.Errorf("expected yaml stored, got %s", got.WorkflowYAML)
	}
}

func TestUpdateRun(t *testing.T) {
	s := testStore(t)
	now := time.Now().Truncate(time.Second)

	run := &WorkflowRun{
		ID: "run-002", WorkflowName: "wf", Status: RunStatusRunning,
		StartedAt: now, TriggerType: "cron",
	}
	s.SaveRun(run)

	run.Status = RunStatusSuccess
	run.CompletedAt = now.Add(5 * time.Second)
	if err := s.UpdateRun(run); err != nil {
		t.Fatalf("UpdateRun failed: %v", err)
	}

	got, _ := s.GetRun("run-002")
	if got.Status != RunStatusSuccess {
		t.Errorf("expected status=success, got %s", got.Status)
	}
}

func TestGetRun_NotFound(t *testing.T) {
	s := testStore(t)
	_, err := s.GetRun("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent run")
	}
}

func TestListRuns(t *testing.T) {
	s := testStore(t)
	now := time.Now().Truncate(time.Second)

	for i, name := range []string{"wf-a", "wf-b", "wf-a"} {
		status := RunStatusSuccess
		if i == 2 {
			status = RunStatusFailed
		}
		s.SaveRun(&WorkflowRun{
			ID: fmt.Sprintf("run-%d", i), WorkflowName: name,
			Status: status, StartedAt: now.Add(time.Duration(i) * time.Second),
			TriggerType: "manual",
		})
	}

	// All runs
	runs, err := s.ListRuns(RunFilter{})
	if err != nil {
		t.Fatalf("ListRuns failed: %v", err)
	}
	if len(runs) != 3 {
		t.Errorf("expected 3 runs, got %d", len(runs))
	}

	// Filter by workflow
	runs, _ = s.ListRuns(RunFilter{WorkflowName: "wf-a"})
	if len(runs) != 2 {
		t.Errorf("expected 2 runs for wf-a, got %d", len(runs))
	}

	// Filter by status
	runs, _ = s.ListRuns(RunFilter{Status: RunStatusFailed})
	if len(runs) != 1 {
		t.Errorf("expected 1 failed run, got %d", len(runs))
	}
}

func TestSaveAndGetTaskExecution(t *testing.T) {
	s := testStore(t)
	now := time.Now().Truncate(time.Second)

	s.SaveRun(&WorkflowRun{
		ID: "run-t1", WorkflowName: "wf", Status: RunStatusRunning,
		StartedAt: now, TriggerType: "manual",
	})

	exec := &TaskExecution{
		ID: "task-001", RunID: "run-t1", TaskName: "step1",
		Provider: "shell", Status: TaskStatusSuccess,
		StartedAt: now, CompletedAt: now.Add(time.Second),
		Output: "hello", Retries: 0, DurationMs: 1000,
	}
	if err := s.SaveTaskExecution(exec); err != nil {
		t.Fatalf("SaveTaskExecution failed: %v", err)
	}

	execs, err := s.GetTaskExecutions("run-t1")
	if err != nil {
		t.Fatalf("GetTaskExecutions failed: %v", err)
	}
	if len(execs) != 1 {
		t.Fatalf("expected 1 execution, got %d", len(execs))
	}
	if execs[0].TaskName != "step1" {
		t.Errorf("expected task_name=step1, got %s", execs[0].TaskName)
	}
	if execs[0].Output != "hello" {
		t.Errorf("expected output=hello, got %s", execs[0].Output)
	}
}

func TestDeleteRunsBefore(t *testing.T) {
	s := testStore(t)
	now := time.Now().Truncate(time.Second)

	s.SaveRun(&WorkflowRun{
		ID: "old", WorkflowName: "wf", Status: RunStatusSuccess,
		StartedAt: now.Add(-48 * time.Hour), TriggerType: "manual",
	})
	s.SaveRun(&WorkflowRun{
		ID: "new", WorkflowName: "wf", Status: RunStatusSuccess,
		StartedAt: now, TriggerType: "manual",
	})

	deleted, err := s.DeleteRunsBefore(now.Add(-24 * time.Hour))
	if err != nil {
		t.Fatalf("DeleteRunsBefore failed: %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}

	runs, _ := s.ListRuns(RunFilter{})
	if len(runs) != 1 {
		t.Errorf("expected 1 remaining run, got %d", len(runs))
	}
}
