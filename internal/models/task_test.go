package models

import "testing"

func TestNewTask(t *testing.T) {
	params := map[string]interface{}{"command": "echo", "args": []interface{}{"hello"}}
	task := NewTask("t1", "test task", "shell", params)
	if task.Name != "t1" {
		t.Errorf("expected Name=t1, got %s", task.Name)
	}
	if task.Description != "test task" {
		t.Errorf("expected Description='test task', got %s", task.Description)
	}
	if task.Provider != "shell" {
		t.Errorf("expected Provider=shell, got %s", task.Provider)
	}
	if len(task.Params) != 2 {
		t.Errorf("expected 2 params, got %d", len(task.Params))
	}
}

func TestTaskRun_UnknownProvider(t *testing.T) {
	task := NewTask("t1", "test", "nonexistent", map[string]interface{}{"key": "val"})
	_, err := task.Run()
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}
