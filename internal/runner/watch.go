package runner

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/y0anfa/rhino/internal/logger"
	"github.com/y0anfa/rhino/internal/models"
	"go.uber.org/zap"
)

type WatchRunner struct {
	Workflow models.Workflow
	watcher  *fsnotify.Watcher
	ctx      context.Context
	cancel   context.CancelFunc
}

func (wr *WatchRunner) Run(ctx context.Context) error {
	watchDir := wr.Workflow.Trigger.WatchPath
	if watchDir == "" {
		return fmt.Errorf("watch runner: watch-path is required for workflow '%s'", wr.Workflow.Name)
	}

	pattern := wr.Workflow.Trigger.WatchPattern // e.g. "*.txt", empty means all files

	debounce := 500 * time.Millisecond
	if d := wr.Workflow.Trigger.Debounce; d != "" {
		if parsed, err := time.ParseDuration(d); err == nil {
			debounce = parsed
		}
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("watch runner: failed to create watcher: %w", err)
	}
	wr.watcher = watcher

	if err := watcher.Add(watchDir); err != nil {
		watcher.Close()
		return fmt.Errorf("watch runner: failed to watch directory '%s': %w", watchDir, err)
	}

	wr.ctx, wr.cancel = context.WithCancel(ctx)

	go func() {
		var timer *time.Timer
		for {
			select {
			case <-wr.ctx.Done():
				if timer != nil {
					timer.Stop()
				}
				return
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if !wr.matchesEvent(event) {
					continue
				}
				// Match filename against pattern
				if pattern != "" {
					matched, _ := filepath.Match(pattern, filepath.Base(event.Name))
					if !matched {
						continue
					}
				}

				logger.Info("file watch triggered",
					zap.String("workflow", wr.Workflow.Name),
					zap.String("file", event.Name),
					zap.String("op", event.Op.String()))

				// Debounce
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(debounce, func() {
					if _, err := wr.Workflow.Run(wr.ctx); err != nil {
						logRunError(wr.Workflow.Name, err)
					}
				})

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				logger.Error("file watch error",
					zap.String("workflow", wr.Workflow.Name),
					zap.Error(err))
			}
		}
	}()

	logger.Info("started file watch runner",
		zap.String("workflow", wr.Workflow.Name),
		zap.String("dir", watchDir),
		zap.String("pattern", pattern))
	return nil
}

func (wr *WatchRunner) WorkflowName() string { return wr.Workflow.Name }

func (wr *WatchRunner) Stop(_ context.Context) error {
	logger.Info("stopping file watch runner", zap.String("workflow", wr.Workflow.Name))
	if wr.cancel != nil {
		wr.cancel()
	}
	if wr.watcher != nil {
		return wr.watcher.Close()
	}
	return nil
}

func (wr *WatchRunner) matchesEvent(event fsnotify.Event) bool {
	events := wr.Workflow.Trigger.WatchEvents
	if len(events) == 0 {
		return true // match all events
	}
	for _, e := range events {
		switch e {
		case "create":
			if event.Op&fsnotify.Create != 0 {
				return true
			}
		case "modify":
			if event.Op&fsnotify.Write != 0 {
				return true
			}
		case "delete":
			if event.Op&fsnotify.Remove != 0 {
				return true
			}
		case "rename":
			if event.Op&fsnotify.Rename != 0 {
				return true
			}
		}
	}
	return false
}
