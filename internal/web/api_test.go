package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/y0anfa/rhino/internal/config"
	"github.com/y0anfa/rhino/internal/store"
)

const sampleWorkflow = `name: api-sample
description: sample
settings:
  max-tries: 1
  timeout: 5s
trigger:
  name: t1
  type: manual
tasks:
  - name: hello
    provider: shell
    params:
      command: echo
      args: ["hi"]
order:
  - [hello]
`

func setup(t *testing.T) http.Handler {
	t.Helper()
	dir := t.TempDir()
	wfDir := filepath.Join(dir, "workflows")
	if err := os.MkdirAll(wfDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wfDir, "api-sample.yaml"), []byte(sampleWorkflow), 0644); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("workflows-dir: "+wfDir+"\nport: 8888\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(config.EnvConfigPath, cfgPath)
	config.Reset()
	t.Cleanup(config.Reset)

	if err := store.Init(filepath.Join(dir, "history.db")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.CloseGlobal)

	return NewServer(0).Handler()
}

func do(t *testing.T, h http.Handler, method, path string) (*httptest.ResponseRecorder, map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var body map[string]interface{}
	if rec.Body.Len() > 0 && rec.Header().Get("Content-Type") == "application/json" {
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
	}
	return rec, body
}

func TestAPI_WorkflowDetail(t *testing.T) {
	h := setup(t)

	rec, body := do(t, h, http.MethodGet, "/api/workflows/api-sample")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if body["name"] != "api-sample" {
		t.Errorf("unexpected body: %v", body)
	}
	tasks, _ := body["tasks"].([]interface{})
	if len(tasks) != 1 {
		t.Errorf("expected one task in summary, got %v", body["tasks"])
	}

	rec, _ = do(t, h, http.MethodGet, "/api/workflows/nope")
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown workflow, got %d", rec.Code)
	}
}

func TestAPI_TriggerRun_Wait(t *testing.T) {
	h := setup(t)

	rec, body := do(t, h, http.MethodPost, "/api/workflows/api-sample/run?wait=true")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if body["status"] != "success" {
		t.Fatalf("expected success, got %v", body)
	}
	runID, _ := body["run_id"].(string)

	rec, detail := do(t, h, http.MethodGet, "/api/runs/"+runID)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected run detail, got %d: %s", rec.Code, rec.Body.String())
	}
	tasks, _ := detail["tasks"].([]interface{})
	if len(tasks) != 1 {
		t.Fatalf("expected the run to have one recorded task, got %v", detail["tasks"])
	}
}

func TestAPI_TriggerRun_Async(t *testing.T) {
	h := setup(t)

	rec, body := do(t, h, http.MethodPost, "/api/workflows/api-sample/run")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
	runID, _ := body["run_id"].(string)
	if runID == "" {
		t.Fatalf("expected a run_id, got %v", body)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if run, err := store.Global().GetRun(runID); err == nil && run.Status == store.RunStatusSuccess {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("async run never completed under the returned run_id")
}

func TestAPI_TriggerRun_RequiresPost(t *testing.T) {
	h := setup(t)
	rec, _ := do(t, h, http.MethodGet, "/api/workflows/api-sample/run")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}

func TestAPI_Runs_Filters(t *testing.T) {
	h := setup(t)
	do(t, h, http.MethodPost, "/api/workflows/api-sample/run?wait=true")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/runs?since=1h&limit=5", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var runs []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &runs); err != nil {
		t.Fatalf("expected a JSON array, got %s", rec.Body.String())
	}
	if len(runs) != 1 {
		t.Errorf("expected one run, got %d", len(runs))
	}

	rec, _ = do(t, h, http.MethodGet, "/api/runs?since=yesterday")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for a bad since value, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/runs?workflow=none", nil))
	if rec.Body.String() != "[]\n" {
		t.Errorf("expected an empty JSON array, got %q", rec.Body.String())
	}
}
