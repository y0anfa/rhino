package models

import (
	"strings"
	"testing"

	_ "github.com/y0anfa/rhino/internal/providers"
)

func TestIntegration_SequentialExecution(t *testing.T) {
	w := NewWorkflow("seq-wf", "sequential workflow")
	w.Settings.Timeout = "5s"
	w.SetTrigger(Trigger{Name: "t1", Type: TriggerManual})
	w.AddTask(Task{
		Name:     "step1",
		Provider: "shell",
		Params:   map[string]interface{}{"command": "echo", "args": []interface{}{"step1"}},
	})
	w.AddTask(Task{
		Name:     "step2",
		Provider: "shell",
		Params:   map[string]interface{}{"command": "echo", "args": []interface{}{"step2"}},
	})
	w.Order = [][]string{{"step1"}, {"step2"}}

	if _, err := w.Run(); err != nil {
		t.Errorf("expected successful sequential run, got error: %v", err)
	}
}

func TestIntegration_ParallelExecution(t *testing.T) {
	w := NewWorkflow("par-wf", "parallel workflow")
	w.Settings.Timeout = "5s"
	w.SetTrigger(Trigger{Name: "t1", Type: TriggerManual})
	w.AddTask(Task{
		Name:     "taskA",
		Provider: "shell",
		Params:   map[string]interface{}{"command": "echo", "args": []interface{}{"A"}},
	})
	w.AddTask(Task{
		Name:     "taskB",
		Provider: "shell",
		Params:   map[string]interface{}{"command": "echo", "args": []interface{}{"B"}},
	})
	w.Order = [][]string{{"taskA", "taskB"}}

	if _, err := w.Run(); err != nil {
		t.Errorf("expected successful parallel run, got error: %v", err)
	}
}

func TestIntegration_RetryThenSucceed(t *testing.T) {
	// This test uses a command that always fails to verify retry behavior.
	// After max retries, it should return an error.
	w := NewWorkflow("retry-wf", "retry workflow")
	w.Settings.MaxTries = 2
	w.Settings.Timeout = "2s"
	w.SetTrigger(Trigger{Name: "t1", Type: TriggerManual})
	w.AddTask(Task{
		Name:     "fail-task",
		Provider: "shell",
		Params:   map[string]interface{}{"command": "false", "args": []interface{}{}},
	})
	w.Order = [][]string{{"fail-task"}}

	_, err := w.Run()
	if err == nil {
		t.Error("expected error after exhausting retries")
	}
	if !strings.Contains(err.Error(), "fail-task") {
		t.Errorf("expected error to reference task name, got: %v", err)
	}
}

func TestIntegration_Timeout(t *testing.T) {
	w := NewWorkflow("timeout-wf", "timeout workflow")
	w.Settings.MaxTries = 1
	w.Settings.Timeout = "100ms"
	w.SetTrigger(Trigger{Name: "t1", Type: TriggerManual})
	w.AddTask(Task{
		Name:     "slow-task",
		Provider: "shell",
		Params:   map[string]interface{}{"command": "sleep", "args": []interface{}{"10"}},
	})
	w.Order = [][]string{{"slow-task"}}

	_, err := w.Run()
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestIntegration_MixedGroups(t *testing.T) {
	// Group 1: parallel tasks succeed
	// Group 2: sequential task succeeds
	w := NewWorkflow("mixed-wf", "mixed workflow")
	w.Settings.Timeout = "5s"
	w.SetTrigger(Trigger{Name: "t1", Type: TriggerManual})
	w.AddTask(Task{
		Name:     "par1",
		Provider: "shell",
		Params:   map[string]interface{}{"command": "echo", "args": []interface{}{"par1"}},
	})
	w.AddTask(Task{
		Name:     "par2",
		Provider: "shell",
		Params:   map[string]interface{}{"command": "echo", "args": []interface{}{"par2"}},
	})
	w.AddTask(Task{
		Name:     "final",
		Provider: "shell",
		Params:   map[string]interface{}{"command": "echo", "args": []interface{}{"final"}},
	})
	w.Order = [][]string{{"par1", "par2"}, {"final"}}

	if _, err := w.Run(); err != nil {
		t.Errorf("expected successful mixed run, got error: %v", err)
	}
}

func TestIntegration_FailureStopsSubsequentGroups(t *testing.T) {
	// Group 1 fails, group 2 should not execute
	w := NewWorkflow("stop-wf", "stop on failure workflow")
	w.Settings.MaxTries = 1
	w.Settings.Timeout = "2s"
	w.SetTrigger(Trigger{Name: "t1", Type: TriggerManual})
	w.AddTask(Task{
		Name:     "fail-first",
		Provider: "shell",
		Params:   map[string]interface{}{"command": "false", "args": []interface{}{}},
	})
	w.AddTask(Task{
		Name:     "never-runs",
		Provider: "shell",
		Params:   map[string]interface{}{"command": "echo", "args": []interface{}{"should not see this"}},
	})
	w.Order = [][]string{{"fail-first"}, {"never-runs"}}

	_, err := w.Run()
	if err == nil {
		t.Error("expected error from first group failure")
	}
	if !strings.Contains(err.Error(), "fail-first") {
		t.Errorf("expected error to reference fail-first, got: %v", err)
	}
}
