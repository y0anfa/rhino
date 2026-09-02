package models

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func inputWorkflow() *Workflow {
	w := manualWorkflow("inputs", shellTask("say", "echo env={{input.env}} version={{input.version}}"))
	w.Inputs = map[string]Input{
		"env":     {Default: "staging", Description: "target"},
		"version": {Required: true},
	}
	return w
}

func TestValidate_InputDeclarations(t *testing.T) {
	w := inputWorkflow()
	if err := w.Validate(); err != nil {
		t.Fatalf("expected valid workflow, got: %v", err)
	}

	w = inputWorkflow()
	w.Inputs["bad name"] = Input{}
	if err := w.Validate(); err == nil || !strings.Contains(err.Error(), "invalid input name") {
		t.Errorf("expected invalid name error, got: %v", err)
	}

	w = inputWorkflow()
	w.Inputs["version"] = Input{Required: true, Default: "1"}
	if err := w.Validate(); err == nil || !strings.Contains(err.Error(), "both required and have a default") {
		t.Errorf("expected required+default error, got: %v", err)
	}

	w = inputWorkflow()
	w.Tasks[0].Params["args"] = []interface{}{"-c", "echo {{input.nope}}"}
	if err := w.Validate(); err == nil || !strings.Contains(err.Error(), "undeclared input 'nope'") {
		t.Errorf("expected undeclared input error, got: %v", err)
	}

	w = inputWorkflow()
	w.Notifications = &NotificationConfig{OnFailure: []NotificationChannel{{
		Provider: "file", Params: map[string]interface{}{"operation": "write", "path": "x", "content": "{{input.ghost}}"},
	}}}
	if err := w.Validate(); err == nil || !strings.Contains(err.Error(), "notification 'file' references undeclared input 'ghost'") {
		t.Errorf("expected notification input error, got: %v", err)
	}
}

func TestResolveInputs(t *testing.T) {
	w := inputWorkflow()

	got, err := w.resolveInputs(map[string]string{"version": "1.2.3"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if got["env"] != "staging" || got["version"] != "1.2.3" {
		t.Errorf("unexpected inputs: %v", got)
	}

	_, err = w.resolveInputs(map[string]string{}, true)
	if !errors.Is(err, ErrInvalidInput) || !strings.Contains(err.Error(), "requires input(s) version") {
		t.Errorf("expected missing required error, got: %v", err)
	}

	_, err = w.resolveInputs(map[string]string{"version": "1", "typo": "x"}, true)
	if !errors.Is(err, ErrInvalidInput) || !strings.Contains(err.Error(), "does not declare input(s) typo") {
		t.Errorf("expected undeclared error in strict mode, got: %v", err)
	}

	got, err = w.resolveInputs(map[string]string{"version": "1", "extra": "ignored"}, false)
	if err != nil || got["version"] != "1" {
		t.Errorf("expected lenient mode to ignore extras, got %v, %v", got, err)
	}
	if _, present := got["extra"]; present {
		t.Errorf("lenient mode must not expose undeclared inputs: %v", got)
	}
}

func TestRun_UsesInputsAndRecordsThem(t *testing.T) {
	s := useHistoryStore(t)
	w := inputWorkflow()

	ctx := WithInputs(WithRunID(context.Background(), "run-inputs"), map[string]string{"version": "2.0"})
	results, err := w.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if out := strings.TrimSpace(results["say"].Output); out != "env=staging version=2.0" {
		t.Errorf("unexpected task output: %q", out)
	}

	run, err := s.GetRun("run-inputs")
	if err != nil {
		t.Fatal(err)
	}
	if run.Inputs["version"] != "2.0" || run.Inputs["env"] != "staging" {
		t.Errorf("inputs not recorded on the run: %v", run.Inputs)
	}
}

func TestRun_RejectsMissingRequiredInputBeforeStarting(t *testing.T) {
	s := useHistoryStore(t)
	w := inputWorkflow()

	_, err := w.Run(WithRunID(context.Background(), "run-missing"))
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got: %v", err)
	}
	if _, err := s.GetRun("run-missing"); err == nil {
		t.Error("a run rejected for bad inputs must not be recorded")
	}
}

func TestChildWorkflow_ReceivesInputs(t *testing.T) {
	dir := useWorkflowsDir(t)
	out := filepath.Join(t.TempDir(), "child.txt")

	child := manualWorkflow("child", Task{
		Name:     "write",
		Provider: "file",
		Params:   map[string]interface{}{"operation": "write", "path": out, "content": "got {{input.greeting}}"},
	})
	child.Inputs = map[string]Input{"greeting": {Required: true}}
	if err := child.Save(); err != nil {
		t.Fatal(err)
	}
	_ = dir

	parent := manualWorkflow("parent", Task{
		Name:     "call",
		Provider: "workflow",
		Params:   map[string]interface{}{"workflow": "child", "inputs": map[string]interface{}{"greeting": "hello"}},
	})
	if err := parent.Validate(); err != nil {
		t.Fatal(err)
	}
	if _, err := parent.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	data, err := readFile(out)
	if err != nil || data != "got hello" {
		t.Errorf("child did not receive inputs: %q, %v", data, err)
	}
}

func TestInputsFromJSON(t *testing.T) {
	got, err := InputsFromJSON([]byte(`{"s":"x","n":1.50,"b":true,"nil":null,"obj":{"a":[1,2]}}`))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"s": "x", "n": "1.50", "b": "true", "nil": "", "obj": `{"a":[1,2]}`}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("input %s: expected %q, got %q", k, v, got[k])
		}
	}
	if _, err := InputsFromJSON([]byte(`[1,2]`)); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for a non-object body, got: %v", err)
	}
	if got, err := InputsFromJSON(nil); err != nil || len(got) != 0 {
		t.Errorf("expected empty inputs for an empty body, got %v, %v", got, err)
	}
}

func TestParseInputFlags(t *testing.T) {
	got, err := ParseInputFlags([]string{"a=1", "b=x=y"})
	if err != nil || got["a"] != "1" || got["b"] != "x=y" {
		t.Errorf("unexpected parse result: %v, %v", got, err)
	}
	if _, err := ParseInputFlags([]string{"novalue"}); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}
