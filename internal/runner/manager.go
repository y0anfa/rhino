package runner

import "context"

type RunnerManager struct {
	Runners []Runner
}

func NewRunnerManager() *RunnerManager {
	return &RunnerManager{}
}

func (rm *RunnerManager) AddRunner(r Runner) {
	rm.Runners = append(rm.Runners, r)
}

func (rm *RunnerManager) Run(ctx context.Context) {
	for _, r := range rm.Runners {
		r.Run(ctx)
	}
}

func (rm *RunnerManager) Stop(ctx context.Context) {
	for _, r := range rm.Runners {
		r.Stop(ctx)
	}
	// Stop the shared webhook server if it's running
	StopWebhookServer(ctx)
}