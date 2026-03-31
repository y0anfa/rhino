package worker

import (
	"context"
	"sync"
	"time"

	"github.com/y0anfa/rhino/internal/logger"
	"github.com/y0anfa/rhino/internal/providers"
	"github.com/y0anfa/rhino/internal/queue"
	"github.com/y0anfa/rhino/internal/store"
	"go.uber.org/zap"
)

type Worker struct {
	Name        string
	Concurrency int
	Queue       queue.Queue
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

func New(name string, concurrency int, q queue.Queue) *Worker {
	if concurrency <= 0 {
		concurrency = 1
	}
	return &Worker{
		Name:        name,
		Concurrency: concurrency,
		Queue:       q,
	}
}

func (w *Worker) Start(ctx context.Context) {
	w.ctx, w.cancel = context.WithCancel(ctx)

	for i := 0; i < w.Concurrency; i++ {
		w.wg.Add(1)
		go w.loop(i)
	}

	logger.Info("worker started",
		zap.String("name", w.Name),
		zap.Int("concurrency", w.Concurrency))
}

func (w *Worker) Stop() {
	logger.Info("stopping worker", zap.String("name", w.Name))
	w.cancel()
	w.wg.Wait()
	logger.Info("worker stopped", zap.String("name", w.Name))
}

func (w *Worker) loop(id int) {
	defer w.wg.Done()

	for {
		msg, err := w.Queue.Dequeue(w.ctx)
		if err != nil {
			if w.ctx.Err() != nil {
				return // shutting down
			}
			logger.Error("worker dequeue error",
				zap.String("worker", w.Name),
				zap.Int("goroutine", id),
				zap.Error(err))
			continue
		}

		w.execute(msg)
	}
}

func (w *Worker) execute(msg *queue.TaskMessage) {
	logger.Info("worker executing task",
		zap.String("worker", w.Name),
		zap.String("task", msg.TaskName),
		zap.String("run_id", msg.RunID))

	startedAt := time.Now()

	provider, err := providers.Get(msg.Provider)
	if err != nil {
		w.fail(msg, startedAt, err)
		return
	}

	if err := provider.Validate(msg.Params); err != nil {
		w.fail(msg, startedAt, err)
		return
	}

	timeout := msg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	taskCtx, cancel := context.WithTimeout(w.ctx, timeout)
	defer cancel()

	result, err := provider.Run(taskCtx, msg.Params)
	completedAt := time.Now()
	duration := completedAt.Sub(startedAt)

	if err != nil {
		w.fail(msg, startedAt, err)
		return
	}

	// Record success
	if s := store.Global(); s != nil {
		output := ""
		if result != nil {
			output = result.Output
		}
		s.SaveTaskExecution(&store.TaskExecution{
			ID:          store.NewID(),
			RunID:       msg.RunID,
			TaskName:    msg.TaskName,
			Provider:    msg.Provider,
			Status:      store.TaskStatusSuccess,
			StartedAt:   startedAt,
			CompletedAt: completedAt,
			Output:      output,
			DurationMs:  duration.Milliseconds(),
		})
	}

	w.Queue.Ack(msg.ID)
	logger.Info("worker task completed",
		zap.String("task", msg.TaskName),
		zap.Duration("duration", duration))
}

func (w *Worker) fail(msg *queue.TaskMessage, startedAt time.Time, err error) {
	completedAt := time.Now()

	if s := store.Global(); s != nil {
		s.SaveTaskExecution(&store.TaskExecution{
			ID:          store.NewID(),
			RunID:       msg.RunID,
			TaskName:    msg.TaskName,
			Provider:    msg.Provider,
			Status:      store.TaskStatusFailed,
			StartedAt:   startedAt,
			CompletedAt: completedAt,
			Error:       err.Error(),
			DurationMs:  completedAt.Sub(startedAt).Milliseconds(),
		})
	}

	w.Queue.Ack(msg.ID) // Don't re-enqueue failed tasks (retries handled at workflow level)

	logger.Error("worker task failed",
		zap.String("task", msg.TaskName),
		zap.Error(err))
}
