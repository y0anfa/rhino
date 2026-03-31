package runner

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/y0anfa/rhino/internal/logger"
	"github.com/y0anfa/rhino/internal/models"
	"go.uber.org/zap"
)

type HotReloader struct {
	workflowsDir string
	manager      *RunnerManager
	watcher      *fsnotify.Watcher
	ctx          context.Context
	cancel       context.CancelFunc
	runners      map[string]Runner // workflow name -> runner
}

func NewHotReloader(workflowsDir string, manager *RunnerManager) *HotReloader {
	return &HotReloader{
		workflowsDir: workflowsDir,
		manager:      manager,
		runners:      make(map[string]Runner),
	}
}

func (hr *HotReloader) Start(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	hr.watcher = watcher
	hr.ctx, hr.cancel = context.WithCancel(ctx)

	if err := watcher.Add(hr.workflowsDir); err != nil {
		watcher.Close()
		return err
	}

	go hr.watch()
	logger.Info("hot reload enabled", zap.String("dir", hr.workflowsDir))
	return nil
}

func (hr *HotReloader) Stop() error {
	if hr.cancel != nil {
		hr.cancel()
	}
	if hr.watcher != nil {
		return hr.watcher.Close()
	}
	return nil
}

func (hr *HotReloader) watch() {
	debounce := time.NewTimer(0)
	debounce.Stop()
	var pendingFile string

	for {
		select {
		case <-hr.ctx.Done():
			debounce.Stop()
			return
		case event, ok := <-hr.watcher.Events:
			if !ok {
				return
			}
			if !strings.HasSuffix(event.Name, ".yaml") && !strings.HasSuffix(event.Name, ".yml") {
				continue
			}
			pendingFile = event.Name
			debounce.Reset(200 * time.Millisecond)

		case <-debounce.C:
			if pendingFile != "" {
				hr.handleChange(pendingFile)
				pendingFile = ""
			}

		case err, ok := <-hr.watcher.Errors:
			if !ok {
				return
			}
			logger.Error("hot reload watcher error", zap.Error(err))
		}
	}
}

func (hr *HotReloader) handleChange(path string) {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	logger.Info("workflow file changed", zap.String("workflow", name), zap.String("path", path))

	// Stop existing runner if any
	if r, ok := hr.runners[name]; ok {
		if err := r.Stop(hr.ctx); err != nil {
			logger.Error("failed to stop runner during reload", zap.String("workflow", name), zap.Error(err))
		}
		delete(hr.runners, name)
	}

	// Try to load and start new runner
	w, err := models.LoadWorkflow(name)
	if err != nil {
		logger.Warn("workflow removed or unreadable", zap.String("workflow", name), zap.Error(err))
		return
	}

	if err := w.Validate(); err != nil {
		logger.Error("reloaded workflow invalid", zap.String("workflow", name), zap.Error(err))
		return
	}

	var r Runner
	switch w.Trigger.Type {
	case models.TriggerScheduled:
		r = &CronRunner{Workflow: w}
	case models.TriggerWebhook:
		r = &WebhookRunner{Workflow: w}
	case models.TriggerWatch:
		r = &WatchRunner{Workflow: w}
	default:
		return
	}

	if err := r.Run(hr.ctx); err != nil {
		logger.Error("failed to start reloaded runner", zap.String("workflow", name), zap.Error(err))
		return
	}

	hr.runners[name] = r
	logger.Info("workflow reloaded", zap.String("workflow", name))
}
