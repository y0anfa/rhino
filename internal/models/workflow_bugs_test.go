package models

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/y0anfa/rhino/internal/config"
)

// useWorkflowsDir points the global config at a temp workflows directory.
func useWorkflowsDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	wfDir := filepath.Join(dir, "workflows")
	if err := os.MkdirAll(wfDir, 0755); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(dir, "config.yaml")
	cfg := "workflows-dir: " + wfDir + "\nport: 8888\n"
	if err := os.WriteFile(cfgPath, []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(config.EnvConfigPath, cfgPath)
	config.Reset()
	t.Cleanup(config.Reset)
	return wfDir
}

func TestListWorkflows_OnlyYAMLFiles(t *testing.T) {
	dir := useWorkflowsDir(t)

	for _, name := range []string{"alpha.yaml", "beta.yml", "notes.txt", ".DS_Store"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("name: x\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "archive"), 0755); err != nil {
		t.Fatal(err)
	}

	got, err := ListWorkflows()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := map[string]bool{"alpha": true, "beta": true}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for _, name := range got {
		if !want[name] {
			t.Errorf("unexpected workflow %q in %v", name, got)
		}
	}
}

func TestListWorkflows_KeepsDottedNames(t *testing.T) {
	dir := useWorkflowsDir(t)
	if err := os.WriteFile(filepath.Join(dir, "my.workflow.yaml"), []byte("name: x\n"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := ListWorkflows()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0] != "my.workflow" {
		t.Fatalf("expected [my.workflow], got %v", got)
	}
}

func TestRun_BuildsOrderFromDependsOn(t *testing.T) {
	out := filepath.Join(t.TempDir(), "dag.txt")

	w := NewWorkflow("dag-wf", "dag workflow")
	w.SetTrigger(Trigger{Name: "t1", Type: TriggerManual})
	w.AddTask(Task{
		Name:     "first",
		Provider: "file",
		Params:   map[string]interface{}{"operation": "write", "path": out, "content": "first"},
	})
	w.AddTask(Task{
		Name:      "second",
		Provider:  "file",
		DependsOn: []string{"first"},
		Params:    map[string]interface{}{"operation": "append", "path": out, "content": "-second"},
	})
	// No explicit order: it is derived from depends-on.

	results, err := w.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 task results, got %d: %v", len(results), results)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("expected tasks to run: %v", err)
	}
	if string(data) != "first-second" {
		t.Errorf("expected 'first-second', got %q", string(data))
	}
}

func TestRun_UnknownTaskInOrder(t *testing.T) {
	w := validWorkflow()
	w.Order = [][]string{{"ghost"}}

	_, err := w.Run(context.Background())
	if err == nil {
		t.Fatal("expected an error for an order referencing an unknown task")
	}
	if !strings.Contains(err.Error(), "ghost") {
		t.Errorf("expected error to name the missing task, got: %v", err)
	}
}

func TestRun_NoOrderAndNoDependsOn(t *testing.T) {
	w := validWorkflow()
	w.Order = nil

	if _, err := w.Run(context.Background()); err == nil {
		t.Fatal("expected an error when a workflow has no execution order")
	}
}
