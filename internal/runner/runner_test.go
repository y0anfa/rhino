package runner

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/y0anfa/rhino/internal/config"
	"github.com/y0anfa/rhino/internal/models"
)

// The workflow must keep running after the HTTP handler returns: net/http cancels
// the request context as soon as the handler is done.
func TestWebhookHandler_WorkflowSurvivesRequestCompletion(t *testing.T) {
	target := filepath.Join(t.TempDir(), "out.txt")

	w := models.NewWorkflow("wh-test", "webhook test")
	w.SetTrigger(models.Trigger{Name: "t1", Type: models.TriggerWebhook})
	w.AddTask(models.Task{
		Name:     "write",
		Provider: "file",
		Params: map[string]interface{}{
			"operation": "write",
			"path":      target,
			"content":   "ok",
		},
	})
	w.Order = [][]string{{"write"}}

	webhookMutex.Lock()
	webhookWorkflows[w.Name] = *w
	webhookMutex.Unlock()
	t.Cleanup(func() { UnregisterWebhookWorkflow(w.Name) })

	reqCtx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodPost, "/webhook/wh-test", nil).WithContext(reqCtx)
	rec := httptest.NewRecorder()

	webhookHandler(rec, req)
	cancel() // what net/http does once the handler returns

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rec.Code)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(target); err == nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("workflow did not execute after the webhook request completed")
}

// Schedules are validated with cron.ParseStandard (5 fields), so the runner must
// schedule them with the same parser.
func TestCronRunner_SchedulesStandardCronExpression(t *testing.T) {
	w := models.NewWorkflow("cron-test", "cron test")
	w.SetTrigger(models.Trigger{Name: "t1", Type: models.TriggerScheduled, Schedule: "*/1 * * * *"})
	w.AddTask(models.Task{
		Name:     "noop",
		Provider: "shell",
		Params:   map[string]interface{}{"command": "true", "args": []interface{}{}},
	})
	w.Order = [][]string{{"noop"}}

	cr := &CronRunner{Workflow: *w}
	if err := cr.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error starting cron runner: %v", err)
	}
	t.Cleanup(func() { _ = cr.Stop(context.Background()) })

	if n := len(cr.Scheduler.Entries()); n != 1 {
		t.Fatalf("expected 1 scheduled entry, got %d", n)
	}
}

func TestCronRunner_InvalidScheduleReturnsError(t *testing.T) {
	w := models.NewWorkflow("cron-bad", "cron test")
	w.SetTrigger(models.Trigger{Name: "t1", Type: models.TriggerScheduled, Schedule: "not a schedule"})

	cr := &CronRunner{Workflow: *w}
	err := cr.Run(context.Background())
	t.Cleanup(func() { _ = cr.Stop(context.Background()) })
	if err == nil {
		t.Fatal("expected an error for an invalid cron schedule")
	}
}

func TestWebhookHandler_RejectsNonPost(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/webhook/anything", nil)
	rec := httptest.NewRecorder()
	webhookHandler(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected %d for GET, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
	if rec.Header().Get("Allow") != http.MethodPost {
		t.Errorf("expected Allow: POST header, got %q", rec.Header().Get("Allow"))
	}
}

// fakeRunner records whether it was stopped, standing in for a runner that was
// registered at startup.
type fakeRunner struct {
	name    string
	stopped bool
}

func (f *fakeRunner) Run(context.Context) error  { return nil }
func (f *fakeRunner) Stop(context.Context) error { f.stopped = true; return nil }
func (f *fakeRunner) WorkflowName() string       { return f.name }

// useWorkflowsDir points the global config at a temp workflows directory.
func useWorkflowsDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	wfDir := filepath.Join(dir, "workflows")
	if err := os.MkdirAll(wfDir, 0755); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("workflows-dir: "+wfDir+"\nport: 8888\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(config.EnvConfigPath, cfgPath)
	config.Reset()
	t.Cleanup(config.Reset)
	return wfDir
}

// A reload must stop the runner that was registered at startup; otherwise the
// old cron schedule keeps firing next to the new one.
func TestHotReloader_ReplacesStartupRunner(t *testing.T) {
	wfDir := useWorkflowsDir(t)
	path := filepath.Join(wfDir, "reloaded.yaml")
	yaml := `name: reloaded
settings:
  max-tries: 1
  timeout: 5s
trigger:
  name: t1
  type: cron
  schedule: "*/5 * * * *"
tasks:
  - name: noop
    provider: shell
    params:
      command: "true"
order:
  - [noop]
`
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	manager := NewRunnerManager()
	old := &fakeRunner{name: "reloaded"}
	manager.AddRunner(old)

	hr := NewHotReloader(wfDir, manager)
	hr.ctx, hr.cancel = context.WithCancel(context.Background())
	t.Cleanup(hr.cancel)

	hr.handleChange(path)

	if !old.stopped {
		t.Fatal("the startup runner was left running after reload")
	}
	replacement := manager.RunnerFor("reloaded")
	if replacement == nil || replacement == Runner(old) {
		t.Fatalf("expected a fresh runner in the manager, got %#v", replacement)
	}
	t.Cleanup(func() { _ = replacement.Stop(context.Background()) })
	if _, ok := replacement.(*CronRunner); !ok {
		t.Errorf("expected a CronRunner, got %T", replacement)
	}
	if n := len(manager.Runners); n != 1 {
		t.Errorf("expected exactly one runner registered, got %d", n)
	}
}

// Deleting a workflow file must remove its runner without registering a new one.
func TestHotReloader_RemovedWorkflowDropsRunner(t *testing.T) {
	wfDir := useWorkflowsDir(t)

	manager := NewRunnerManager()
	old := &fakeRunner{name: "gone"}
	manager.AddRunner(old)

	hr := NewHotReloader(wfDir, manager)
	hr.ctx, hr.cancel = context.WithCancel(context.Background())
	t.Cleanup(hr.cancel)

	hr.handleChange(filepath.Join(wfDir, "gone.yaml"))

	if !old.stopped {
		t.Fatal("runner for the deleted workflow was not stopped")
	}
	if manager.RunnerFor("gone") != nil {
		t.Fatal("runner for the deleted workflow is still registered")
	}
}
