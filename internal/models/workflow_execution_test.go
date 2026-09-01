package models

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/y0anfa/rhino/internal/store"
	"github.com/y0anfa/rhino/internal/template"
)

// useHistoryStore points the global history store at a temp database.
func useHistoryStore(t *testing.T) store.Store {
	t.Helper()
	if err := store.Init(filepath.Join(t.TempDir(), "history.db")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.CloseGlobal)
	return store.Global()
}

func shellTask(name, script string) Task {
	return Task{
		Name:     name,
		Provider: "shell",
		Params:   map[string]interface{}{"command": "sh", "args": []interface{}{"-c", script}},
	}
}

func manualWorkflow(name string, tasks ...Task) *Workflow {
	w := NewWorkflow(name, "test workflow")
	w.Settings.MaxTries = 1
	w.SetTrigger(Trigger{Name: "t1", Type: TriggerManual})
	var order [][]string
	for _, t := range tasks {
		w.AddTask(t)
		order = append(order, []string{t.Name})
	}
	w.Order = order
	return w
}

func TestValidate_RejectsDuplicateTaskNames(t *testing.T) {
	w := manualWorkflow("dup", shellTask("a", "true"), shellTask("a", "true"))
	err := w.Validate()
	if err == nil || !strings.Contains(err.Error(), "duplicate task name 'a'") {
		t.Fatalf("expected duplicate task error, got: %v", err)
	}
}

func TestValidate_RejectsTaskMissingFromOrder(t *testing.T) {
	w := manualWorkflow("missing", shellTask("a", "true"), shellTask("b", "true"))
	w.Order = [][]string{{"a"}}
	err := w.Validate()
	if err == nil || !strings.Contains(err.Error(), "task 'b' is not listed in order") {
		t.Fatalf("expected missing-from-order error, got: %v", err)
	}
}

func TestValidate_RejectsTaskRepeatedInOrder(t *testing.T) {
	w := manualWorkflow("repeat", shellTask("a", "true"))
	w.Order = [][]string{{"a"}, {"a"}}
	err := w.Validate()
	if err == nil || !strings.Contains(err.Error(), "appears more than once") {
		t.Fatalf("expected repeated-in-order error, got: %v", err)
	}
}

func TestValidate_RejectsBadTaskTimeoutAndBackoff(t *testing.T) {
	w := manualWorkflow("bad-timeout", shellTask("a", "true"))
	w.Tasks[0].Timeout = "soon"
	if err := w.Validate(); err == nil || !strings.Contains(err.Error(), "invalid timeout 'soon'") {
		t.Fatalf("expected task timeout error, got: %v", err)
	}

	w = manualWorkflow("bad-backoff", shellTask("a", "true"))
	w.Settings.RetryBackoff = "fibonacci"
	if err := w.Validate(); err == nil || !strings.Contains(err.Error(), "invalid retry-backoff") {
		t.Fatalf("expected retry-backoff error, got: %v", err)
	}
}

func TestValidate_RejectsUnknownTriggerAndCondition(t *testing.T) {
	w := manualWorkflow("bad-trigger", shellTask("a", "true"))
	w.Trigger.Type = "carrier-pigeon"
	if err := w.Validate(); err == nil || !strings.Contains(err.Error(), "unknown trigger type") {
		t.Fatalf("expected trigger type error, got: %v", err)
	}

	w = manualWorkflow("bad-cond", shellTask("a", "true"), shellTask("b", "true"))
	w.Tasks[1].Condition = "{{task.a.status}} == maybe"
	if err := w.Validate(); err == nil || !strings.Contains(err.Error(), "status must be success, failed, or skipped") {
		t.Fatalf("expected condition error, got: %v", err)
	}

	w = manualWorkflow("ok-cond", shellTask("a", "true"), shellTask("b", "true"))
	w.Tasks[1].Condition = "{{task.a.status}} != failed"
	if err := w.Validate(); err != nil {
		t.Fatalf("expected '!=' condition to validate, got: %v", err)
	}
}

func TestEvaluateCondition_NotEqual(t *testing.T) {
	w := &Workflow{}
	statuses := map[string]string{"a": "failed"}

	if !w.evaluateCondition(&Task{Condition: "{{task.a.status}} != success"}, statuses) {
		t.Error("expected 'failed != success' to be true")
	}
	if w.evaluateCondition(&Task{Condition: "{{task.a.status}} != failed"}, statuses) {
		t.Error("expected 'failed != failed' to be false")
	}
	// A task that never ran has no status and never equals a real one.
	if w.evaluateCondition(&Task{Condition: "{{task.ghost.status}} == success"}, statuses) {
		t.Error("expected unknown task status to not equal success")
	}
}

func TestRun_RecordsTaskExecutions(t *testing.T) {
	s := useHistoryStore(t)

	w := manualWorkflow("history",
		shellTask("ok", "echo hello"),
		shellTask("boom", "echo bad >&2; exit 1"),
	)
	w.Tasks[1].ContinueOnError = true
	skipped := shellTask("after", "true")
	skipped.Condition = "{{task.boom.status}} == success"
	w.AddTask(skipped)
	w.Order = append(w.Order, []string{"after"})

	runID := "run-history-test"
	if _, err := w.Run(WithRunID(context.Background(), runID)); err != nil {
		t.Fatalf("unexpected run error: %v", err)
	}

	run, err := s.GetRun(runID)
	if err != nil {
		t.Fatalf("run not recorded under the caller's ID: %v", err)
	}
	if run.Status != store.RunStatusSuccess {
		t.Errorf("expected run status success, got %s", run.Status)
	}

	execs, err := s.GetTaskExecutions(runID)
	if err != nil {
		t.Fatal(err)
	}
	byName := make(map[string]*store.TaskExecution)
	for _, e := range execs {
		byName[e.TaskName] = e
	}
	if len(byName) != 3 {
		t.Fatalf("expected 3 task executions, got %d", len(byName))
	}
	if e := byName["ok"]; e.Status != store.TaskStatusSuccess || strings.TrimSpace(e.Output) != "hello" || e.Provider != "shell" {
		t.Errorf("unexpected 'ok' execution: %+v", e)
	}
	for name, e := range byName {
		if e.CompletedAt.IsZero() || e.CompletedAt.Before(e.StartedAt) {
			t.Errorf("task %s has a bad completion time: started=%s completed=%s", name, e.StartedAt, e.CompletedAt)
		}
		if e.DurationMs < 0 || e.DurationMs > 10_000 {
			t.Errorf("task %s has an implausible duration: %dms", name, e.DurationMs)
		}
	}
	if e := byName["boom"]; e.Status != store.TaskStatusFailed || !strings.Contains(e.Error, "bad") {
		t.Errorf("unexpected 'boom' execution: %+v", e)
	}
	if e := byName["after"]; e.Status != store.TaskStatusSkipped {
		t.Errorf("expected 'after' to be recorded as skipped, got %+v", e)
	}
}

func TestRun_RecordsRetryCount(t *testing.T) {
	s := useHistoryStore(t)

	w := manualWorkflow("retries", shellTask("flaky", "exit 1"))
	w.Settings.MaxTries = 3

	_, err := w.Run(WithRunID(context.Background(), "run-retries"))
	if err == nil {
		t.Fatal("expected the workflow to fail")
	}
	execs, _ := s.GetTaskExecutions("run-retries")
	if len(execs) != 1 || execs[0].Retries != 2 {
		t.Fatalf("expected one execution with 2 retries, got %+v", execs)
	}
}

func TestRun_DoesNotMutateTaskMaxTries(t *testing.T) {
	w := manualWorkflow("no-mutate", shellTask("a", "true"))
	w.Settings.MaxTries = 4
	if _, err := w.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if w.Tasks[0].MaxTries != 0 {
		t.Errorf("Run must not write the workflow default back into the task, got %d", w.Tasks[0].MaxTries)
	}
}

func TestRun_TruncatesOutput(t *testing.T) {
	w := manualWorkflow("truncate", shellTask("big", "printf 'abcdefghij'"))
	w.Settings.MaxOutputSize = 4

	results, err := w.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	res := results["big"]
	if res.Output != "abcd" {
		t.Errorf("expected output truncated to 'abcd', got %q", res.Output)
	}
	if res.Metadata["output_truncated"] != "true" {
		t.Errorf("expected output_truncated metadata, got %v", res.Metadata)
	}
}

func TestRun_EnforcesMaxConcurrentRuns(t *testing.T) {
	w := manualWorkflow("limited", shellTask("slow", "sleep 0.5"))
	w.Settings.MaxConcurrentRuns = 1
	w.Settings.Timeout = "5s"

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := w.Run(context.Background())
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)

	var ok, rejected int
	for err := range errs {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, ErrTooManyRuns):
			rejected++
		default:
			t.Errorf("unexpected error: %v", err)
		}
	}
	if ok != 1 || rejected != 1 {
		t.Fatalf("expected 1 success and 1 rejection, got ok=%d rejected=%d", ok, rejected)
	}

	// The slot is released once the run finishes.
	if _, err := w.Run(context.Background()); err != nil {
		t.Fatalf("expected the slot to be free again, got: %v", err)
	}
}

func TestRun_CancellationDoesNotRetry(t *testing.T) {
	s := useHistoryStore(t)

	w := manualWorkflow("cancel", shellTask("slow", "sleep 5"))
	w.Settings.MaxTries = 5
	w.Settings.Timeout = "10s"

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(100*time.Millisecond, cancel)

	start := time.Now()
	_, err := w.Run(WithRunID(ctx, "run-cancel"))
	if err == nil {
		t.Fatal("expected a cancellation error")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("cancelled run kept retrying for %s", elapsed)
	}
	execs, _ := s.GetTaskExecutions("run-cancel")
	if len(execs) != 1 || execs[0].Retries != 0 {
		t.Fatalf("expected a single attempt after cancellation, got %+v", execs)
	}
}

func TestSendNotifications_ExposesRunContext(t *testing.T) {
	out := filepath.Join(t.TempDir(), "notify.txt")

	w := manualWorkflow("notify", shellTask("boom", "echo nope >&2; exit 1"))
	w.Notifications = &NotificationConfig{
		OnFailure: []NotificationChannel{{
			Provider: "file",
			Params: map[string]interface{}{
				"operation": "write",
				"path":      out,
				"content":   "run={{run.id}} error={{workflow.error}}",
			},
		}},
	}

	_, err := w.Run(WithRunID(context.Background(), "run-notify"))
	if err == nil {
		t.Fatal("expected the workflow to fail")
	}

	data, readErr := readFile(out)
	if readErr != nil {
		t.Fatalf("notification was not delivered: %v", readErr)
	}
	if !strings.Contains(data, "run=run-notify") || !strings.Contains(data, "nope") {
		t.Errorf("notification did not resolve run context: %q", data)
	}
}

func TestTemplateContext_RunFields(t *testing.T) {
	ctx := template.NewContext("wf", "d", "t", "manual")
	ctx.RunID = "abc"
	ctx.SetRunError(errors.New("kaput"))
	if got := ctx.Resolve("{{run.id}}/{{workflow.error}}"); got != "abc/kaput" {
		t.Errorf("unexpected resolution: %q", got)
	}
}

func readFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	return string(data), err
}
