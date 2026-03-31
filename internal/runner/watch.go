package runner

import (
	"context"
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
	watchPath := wr.Workflow.Trigger.WatchPath
	if watchPath == "" {
		return nil
	}

	debounce := 500 * time.Millisecond
	if d := wr.Workflow.Trigger.Debounce; d != "" {
		if parsed, err := time.ParseDuration(d); err == nil {
			debounce = parsed
		}
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	wr.watcher = watcher

	// Resolve the watch path to a directory
	dir := filepath.Dir(watchPath)
	if err := watcher.Add(dir); err != nil {
		watcher.Close()
		return err
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
				// Match against watch path pattern
				matched, _ := filepath.Match(watchPath, event.Name)
				if !matched {
					matched, _ = filepath.Match(watchPath, filepath.Base(event.Name))
				}
				if !matched {
					continue
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
						logger.Error("workflow execution failed",
							zap.String("workflow", wr.Workflow.Name),
							zap.Error(err))
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
		zap.String("path", watchPath))
	return nil
}

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
