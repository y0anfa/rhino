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

// reloadDebounce coalesces the burst of events an editor emits per save.
const reloadDebounce = 200 * time.Millisecond

// HotReloader watches the workflows directory and swaps runners in the manager
// when a workflow file changes. The manager is the single source of truth for
// which runner is active, so a reload never leaves the previous runner alive.
type HotReloader struct {
	workflowsDir string
	manager      *RunnerManager
	watcher      *fsnotify.Watcher
	ctx          context.Context
	cancel       context.CancelFunc
}

func NewHotReloader(workflowsDir string, manager *RunnerManager) *HotReloader {
	return &HotReloader{
		workflowsDir: workflowsDir,
		manager:      manager,
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
	pending := make(map[string]struct{})

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
			pending[event.Name] = struct{}{}
			debounce.Reset(reloadDebounce)

		case <-debounce.C:
			for path := range pending {
				hr.handleChange(path)
			}
			pending = make(map[string]struct{})

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

	// Stop and forget the runner that is currently serving this workflow.
	if existing := hr.manager.RunnerFor(name); existing != nil {
		if err := existing.Stop(hr.ctx); err != nil {
			logger.Error("failed to stop runner during reload", zap.String("workflow", name), zap.Error(err))
		}
		hr.manager.RemoveRunner(existing)
	}

	w, err := models.LoadWorkflow(name)
	if err != nil {
		logger.Warn("workflow removed or unreadable", zap.String("workflow", name), zap.Error(err))
		return
	}

	if err := w.Validate(); err != nil {
		logger.Error("reloaded workflow invalid", zap.String("workflow", name), zap.Error(err))
		return
	}

	r, err := NewRunnerFor(w)
	if err != nil {
		logger.Error("reloaded workflow has no runner", zap.String("workflow", name), zap.Error(err))
		return
	}
	if r == nil {
		logger.Info("workflow reloaded (manual trigger, no runner)", zap.String("workflow", name))
		return
	}

	if err := r.Run(hr.ctx); err != nil {
		logger.Error("failed to start reloaded runner", zap.String("workflow", name), zap.Error(err))
		return
	}

	hr.manager.AddRunner(r)
	logger.Info("workflow reloaded", zap.String("workflow", name))
}
