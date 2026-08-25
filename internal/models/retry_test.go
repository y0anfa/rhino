package models

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/y0anfa/rhino/internal/providers"
)

type countingProvider struct {
	mu     sync.Mutex
	calls  int
	onCall func()
}

func (p *countingProvider) Name() string { return "counting-test" }

func (p *countingProvider) Validate(args map[string]interface{}) error { return nil }

func (p *countingProvider) Run(ctx context.Context, args map[string]interface{}) (*providers.TaskResult, error) {
	p.mu.Lock()
	p.calls++
	n := p.calls
	p.mu.Unlock()
	if p.onCall != nil {
		p.onCall()
	}
	return nil, fmt.Errorf("always fails (call %d)", n)
}

func (p *countingProvider) Calls() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.calls
}

// A cancelled workflow must stop retrying instead of burning through the
// remaining attempts (and re-invoking the provider with a dead context).
func TestRun_StopsRetryingAfterCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := &countingProvider{onCall: cancel}
	providers.Register(p)

	w := NewWorkflow("cancel-retry", "cancelled during backoff")
	w.Settings.MaxTries = 5
	w.Settings.Timeout = "5s"
	w.Settings.RetryBackoff = "linear"
	w.Settings.RetryBaseDelay = "50ms"
	w.Settings.RetryMaxDelay = "1s"
	w.SetTrigger(Trigger{Name: "t1", Type: TriggerManual})
	w.AddTask(Task{Name: "flaky", Provider: "counting-test", Params: map[string]interface{}{"k": "v"}})
	w.Order = [][]string{{"flaky"}}

	if _, err := w.Run(ctx); err == nil {
		t.Fatal("expected the workflow to fail")
	}

	// Give any stray retry goroutine a chance to call the provider again.
	time.Sleep(300 * time.Millisecond)

	if got := p.Calls(); got != 1 {
		t.Errorf("expected 1 provider call after cancellation, got %d", got)
	}
}

func TestRetryDelay_StaysWithinBounds(t *testing.T) {
	for _, backoff := range []string{"linear", "exponential"} {
		w := NewWorkflow("delays", "")
		w.Settings.RetryBackoff = backoff
		w.Settings.RetryBaseDelay = "1s"
		w.Settings.RetryMaxDelay = "10s"

		for attempt := 0; attempt < 100; attempt++ {
			d := w.retryDelay(attempt)
			if d < 0 || d > 10*time.Second {
				t.Fatalf("%s backoff, attempt %d: delay %s outside [0s, 10s]", backoff, attempt, d)
			}
		}
	}
}
