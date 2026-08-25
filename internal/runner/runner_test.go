package runner

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

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
