package template

import (
	"strings"
	"testing"

	"github.com/y0anfa/rhino/internal/providers"
)

func TestResolve_EnvVar(t *testing.T) {
	t.Setenv("TEST_VAR", "hello")
	ctx := NewContext("wf", "desc", "t1", "manual")
	got := ctx.Resolve("value is {{env.TEST_VAR}}")
	if got != "value is hello" {
		t.Errorf("expected 'value is hello', got '%s'", got)
	}
}

func TestResolve_TaskOutput(t *testing.T) {
	ctx := NewContext("wf", "desc", "t1", "manual")
	ctx.SetTaskResult("task1", &providers.TaskResult{Output: "result123"})
	got := ctx.Resolve("output: {{task.task1.output}}")
	if got != "output: result123" {
		t.Errorf("expected 'output: result123', got '%s'", got)
	}
}

func TestResolve_TaskMetadata(t *testing.T) {
	ctx := NewContext("wf", "desc", "t1", "manual")
	ctx.SetTaskResult("task1", &providers.TaskResult{
		Metadata: map[string]string{"status_code": "200"},
	})
	got := ctx.Resolve("status: {{task.task1.metadata.status_code}}")
	if got != "status: 200" {
		t.Errorf("expected 'status: 200', got '%s'", got)
	}
}

func TestResolve_WorkflowMetadata(t *testing.T) {
	ctx := NewContext("my-wf", "my desc", "t1", "cron")
	got := ctx.Resolve("{{workflow.name}} - {{workflow.description}}")
	if got != "my-wf - my desc" {
		t.Errorf("expected 'my-wf - my desc', got '%s'", got)
	}
}

func TestResolve_TriggerMetadata(t *testing.T) {
	ctx := NewContext("wf", "desc", "my-trigger", "webhook")
	got := ctx.Resolve("{{trigger.name}}/{{trigger.type}}")
	if got != "my-trigger/webhook" {
		t.Errorf("expected 'my-trigger/webhook', got '%s'", got)
	}
}

func TestResolve_Timestamp(t *testing.T) {
	ctx := NewContext("wf", "desc", "t1", "manual")
	got := ctx.Resolve("at {{timestamp}}")
	if !strings.HasPrefix(got, "at 20") {
		t.Errorf("expected timestamp, got '%s'", got)
	}
}

func TestResolve_Date(t *testing.T) {
	ctx := NewContext("wf", "desc", "t1", "manual")
	got := ctx.Resolve("on {{date}}")
	if !strings.HasPrefix(got, "on 20") {
		t.Errorf("expected date, got '%s'", got)
	}
}

func TestResolve_UnknownExpression(t *testing.T) {
	ctx := NewContext("wf", "desc", "t1", "manual")
	got := ctx.Resolve("{{unknown.expr}}")
	if got != "{{unknown.expr}}" {
		t.Errorf("expected unchanged placeholder, got '%s'", got)
	}
}

func TestResolve_MissingTask(t *testing.T) {
	ctx := NewContext("wf", "desc", "t1", "manual")
	got := ctx.Resolve("{{task.missing.output}}")
	if got != "{{task.missing.output}}" {
		t.Errorf("expected unchanged placeholder, got '%s'", got)
	}
}

func TestResolveParams(t *testing.T) {
	ctx := NewContext("wf", "desc", "t1", "manual")
	ctx.SetTaskResult("t1", &providers.TaskResult{Output: "val"})

	params := map[string]interface{}{
		"key":  "{{task.t1.output}}",
		"list": []interface{}{"{{workflow.name}}", "literal"},
		"num":  42,
	}
	resolved := ResolveParams(params, ctx)

	if resolved["key"] != "val" {
		t.Errorf("expected 'val', got '%v'", resolved["key"])
	}
	list := resolved["list"].([]interface{})
	if list[0] != "wf" {
		t.Errorf("expected 'wf', got '%v'", list[0])
	}
	if list[1] != "literal" {
		t.Errorf("expected 'literal', got '%v'", list[1])
	}
	if resolved["num"] != 42 {
		t.Errorf("expected 42, got '%v'", resolved["num"])
	}
}

func TestValidateTemplateRefs(t *testing.T) {
	params := map[string]interface{}{
		"url": "http://example.com/{{task.step1.output}}",
	}

	warnings := ValidateTemplateRefs(params, []string{"step1"})
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}

	warnings = ValidateTemplateRefs(params, []string{})
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning, got %v", warnings)
	}
}
