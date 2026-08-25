package cmd

import (
	"context"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/y0anfa/rhino/internal/logger"
	"github.com/y0anfa/rhino/internal/queue"
	"github.com/y0anfa/rhino/internal/worker"
	"go.uber.org/zap"
)

var (
	workerConcurrency int
	workerName        string
)

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Start a task worker process",
	Long:  `Start a worker that dequeues and executes tasks from the task queue.`,
	Run: func(cmd *cobra.Command, args []string) {
		logger.Info("starting worker",
			zap.String("name", workerName),
			zap.Int("concurrency", workerConcurrency))

		// The in-memory queue lives inside this process: it is only fed by
		// producers running here, not by other rhino processes.
		logger.Warn("using the in-memory task queue; only tasks enqueued in this process will be executed")
		q := queue.NewMemoryQueue()
		w := worker.New(workerName, workerConcurrency, q)

		ctx := context.Background()
		w.Start(ctx)

		logger.Info("worker running, press Ctrl+C to stop")

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		logger.Info("shutting down worker")
		w.Stop()
		q.Close()
		logger.Info("worker stopped")
	},
}

func init() {
	workerCmd.Flags().IntVar(&workerConcurrency, "concurrency", runtime.NumCPU(), "Number of concurrent task executors")
	workerCmd.Flags().StringVar(&workerName, "name", "worker-1", "Worker identifier")
	rootCmd.AddCommand(workerCmd)
}
