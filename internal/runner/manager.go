package runner

import (
	"context"
	"errors"
)

type RunnerManager struct {
	Runners []Runner
}

func NewRunnerManager() *RunnerManager {
	return &RunnerManager{}
}

func (rm *RunnerManager) AddRunner(r Runner) {
	rm.Runners = append(rm.Runners, r)
}

func (rm *RunnerManager) Run(ctx context.Context) error {
	var errs []error
	for _, r := range rm.Runners {
		if err := r.Run(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (rm *RunnerManager) Stop(ctx context.Context) error {
	var errs []error
	for _, r := range rm.Runners {
		if err := r.Stop(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if err := StopWebhookServer(ctx); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}