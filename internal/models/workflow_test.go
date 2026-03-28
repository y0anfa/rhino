package models

import (
	"strings"
	"testing"

	_ "github.com/y0anfa/rhino/internal/providers" // register providers
)

func validWorkflow() *Workflow {
	w := NewWorkflow("test-workflow", "a test workflow")
	w.SetTrigger(Trigger{Name: "t1", Type: TriggerManual})
	w.AddTask(Task{
		Name:     "task1",
		Provider: "shell",
		Params:   map[string]interface{}{"command": "echo", "args": []interface{}{"hello"}},
	})
	w.Order = [][]string{{"task1"}}
	return w
}

func TestNewWorkflow(t *testing.T) {
	w := NewWorkflow("wf", "desc")
	if w.Name != "wf" {
		t.Errorf("expected Name=wf, got %s", w.Name)
	}
	if w.Settings.MaxTries != MaxTriesDefault {
		t.Errorf("expected MaxTries=%d, got %d", MaxTriesDefault, w.Settings.MaxTries)
	}
}

func TestGetTask(t *testing.T) {
	w := validWorkflow()

	task := w.GetTask("task1")
	if task == nil {
		t.Fatal("expected to find task1")
	}
	if task.Name != "task1" {
		t.Errorf("expected Name=task1, got %s", task.Name)
	}

	// Verify it returns a pointer to the actual slice element
	task.MaxTries = 99
	task2 := w.GetTask("task1")
	if task2.MaxTries != 99 {
		t.Error("GetTask should return pointer to slice element, not a copy")
	}
}

func TestGetTask_NotFound(t *testing.T) {
	w := validWorkflow()
	task := w.GetTask("nonexistent")
	if task != nil {
		t.Error("expected nil for nonexistent task")
	}
}

func TestAddTask(t *testing.T) {
	w := NewWorkflow("wf", "desc")
	w.AddTask(Task{Name: "t1"})
	w.AddTask(Task{Name: "t2"})
	if len(w.Tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(w.Tasks))
	}
}

func TestRemoveTask(t *testing.T) {
	w := validWorkflow()
	w.AddTask(Task{Name: "task2"})

	name, err := w.RemoveTask(Task{Name: "task1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "task1" {
		t.Errorf("expected removed task name=task1, got %s", name)
	}
	if len(w.Tasks) != 1 {
		t.Errorf("expected 1 task remaining, got %d", len(w.Tasks))
	}
}

func TestRemoveTask_NotFound(t *testing.T) {
	w := validWorkflow()
	_, err := w.RemoveTask(Task{Name: "nonexistent"})
	if err == nil {
		t.Error("expected error for nonexistent task")
	}
}

func TestValidate_Valid(t *testing.T) {
	w := validWorkflow()
	if err := w.Validate(); err != nil {
		t.Errorf("expected valid workflow, got error: %v", err)
	}
}

func TestValidate_EmptyName(t *testing.T) {
	w := validWorkflow()
	w.Name = ""
	err := w.Validate()
	if err == nil || !strings.Contains(err.Error(), "name is empty") {
		t.Errorf("expected name validation error, got: %v", err)
	}
}

func TestValidate_InvalidMaxTries(t *testing.T) {
	w := validWorkflow()
	w.Settings.MaxTries = 0
	err := w.Validate()
	if err == nil || !strings.Contains(err.Error(), "max tries") {
		t.Errorf("expected max tries validation error, got: %v", err)
	}
}

func TestValidate_EmptyTimeout(t *testing.T) {
	w := validWorkflow()
	w.Settings.Timeout = ""
	err := w.Validate()
	if err == nil || !strings.Contains(err.Error(), "timeout is empty") {
		t.Errorf("expected timeout validation error, got: %v", err)
	}
}

func TestValidate_InvalidTimeout(t *testing.T) {
	w := validWorkflow()
	w.Settings.Timeout = "notaduration"
	err := w.Validate()
	if err == nil || !strings.Contains(err.Error(), "invalid timeout format") {
		t.Errorf("expected timeout format error, got: %v", err)
	}
}

func TestValidate_EmptyTriggerName(t *testing.T) {
	w := validWorkflow()
	w.Trigger.Name = ""
	err := w.Validate()
	if err == nil || !strings.Contains(err.Error(), "trigger name is empty") {
		t.Errorf("expected trigger name error, got: %v", err)
	}
}

func TestValidate_EmptyTriggerType(t *testing.T) {
	w := validWorkflow()
	w.Trigger.Type = ""
	err := w.Validate()
	if err == nil || !strings.Contains(err.Error(), "trigger type is empty") {
		t.Errorf("expected trigger type error, got: %v", err)
	}
}

func TestValidate_CronWithoutSchedule(t *testing.T) {
	w := validWorkflow()
	w.Trigger.Type = TriggerScheduled
	w.Trigger.Schedule = ""
	err := w.Validate()
	if err == nil || !strings.Contains(err.Error(), "trigger schedule is empty") {
		t.Errorf("expected schedule error, got: %v", err)
	}
}

func TestValidate_InvalidCronSchedule(t *testing.T) {
	w := validWorkflow()
	w.Trigger.Type = TriggerScheduled
	w.Trigger.Schedule = "invalid"
	err := w.Validate()
	if err == nil || !strings.Contains(err.Error(), "invalid cron schedule") {
		t.Errorf("expected cron schedule error, got: %v", err)
	}
}

func TestValidate_NoTasks(t *testing.T) {
	w := validWorkflow()
	w.Tasks = nil
	err := w.Validate()
	if err == nil || !strings.Contains(err.Error(), "tasks list is empty") {
		t.Errorf("expected tasks empty error, got: %v", err)
	}
}

func TestValidate_TaskWithoutName(t *testing.T) {
	w := validWorkflow()
	w.Tasks[0].Name = ""
	err := w.Validate()
	if err == nil || !strings.Contains(err.Error(), "task name is empty") {
		t.Errorf("expected task name error, got: %v", err)
	}
}

func TestValidate_TaskWithoutProvider(t *testing.T) {
	w := validWorkflow()
	w.Tasks[0].Provider = ""
	err := w.Validate()
	if err == nil || !strings.Contains(err.Error(), "provider is empty") {
		t.Errorf("expected provider error, got: %v", err)
	}
}

func TestValidate_TaskWithEmptyParams(t *testing.T) {
	w := validWorkflow()
	w.Tasks[0].Params = nil
	err := w.Validate()
	if err == nil || !strings.Contains(err.Error(), "params are empty") {
		t.Errorf("expected params error, got: %v", err)
	}
}

func TestValidate_UnknownProvider(t *testing.T) {
	w := validWorkflow()
	w.Tasks[0].Provider = "nonexistent"
	err := w.Validate()
	if err == nil || !strings.Contains(err.Error(), "unknown provider") {
		t.Errorf("expected unknown provider error, got: %v", err)
	}
}

func TestValidate_EmptyOrder(t *testing.T) {
	w := validWorkflow()
	w.Order = nil
	err := w.Validate()
	if err == nil || !strings.Contains(err.Error(), "order is empty") {
		t.Errorf("expected order empty error, got: %v", err)
	}
}

func TestValidate_EmptyOrderGroup(t *testing.T) {
	w := validWorkflow()
	w.Order = [][]string{{}}
	err := w.Validate()
	if err == nil || !strings.Contains(err.Error(), "order group is empty") {
		t.Errorf("expected order group error, got: %v", err)
	}
}

func TestValidate_OrderReferencesNonexistentTask(t *testing.T) {
	w := validWorkflow()
	w.Order = [][]string{{"nonexistent"}}
	err := w.Validate()
	if err == nil || !strings.Contains(err.Error(), "not found in order") {
		t.Errorf("expected task not found error, got: %v", err)
	}
}

func TestDescribe(t *testing.T) {
	w := validWorkflow()
	desc := w.Describe()
	if !strings.Contains(desc, "test-workflow") {
		t.Error("expected workflow name in describe output")
	}
	if !strings.Contains(desc, "task1") {
		t.Error("expected task name in describe output")
	}
	if !strings.Contains(desc, "shell") {
		t.Error("expected provider in describe output")
	}
}

func TestRun_Success(t *testing.T) {
	w := validWorkflow()
	err := w.Run()
	if err != nil {
		t.Errorf("expected successful run, got error: %v", err)
	}
}

func TestRun_FailingTask(t *testing.T) {
	w := NewWorkflow("fail-wf", "failing workflow")
	w.Settings.Timeout = "2s"
	w.SetTrigger(Trigger{Name: "t1", Type: TriggerManual})
	w.AddTask(Task{
		Name:     "fail-task",
		Provider: "shell",
		Params:   map[string]interface{}{"command": "false", "args": []interface{}{}},
	})
	w.Order = [][]string{{"fail-task"}}

	err := w.Run()
	if err == nil {
		t.Error("expected error from failing task")
	}
}
