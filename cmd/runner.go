package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/y0anfa/rhino/internal/config"
	"github.com/y0anfa/rhino/internal/logger"
	"github.com/y0anfa/rhino/internal/models"
	"github.com/y0anfa/rhino/internal/runner"
	"go.uber.org/zap"
)

const shutdownTimeout = 30 * time.Second

var runnerCmd = &cobra.Command{
	Use:   "runner",
	Short: "Start the workflow runner daemon",
	Long: `Start the runner daemon which will:
- Start cron schedulers for workflows with cron triggers
- Start webhook server for workflows with webhook triggers
- Monitor and execute workflows based on their triggers`,
	Run: func(cmd *cobra.Command, args []string) {
		logger.Info("starting runner")

		runnerManager := runner.NewRunnerManager()

		workflows, err := models.LoadWorkflows()
		if err != nil {
			logger.Fatal("failed to load workflows", zap.Error(err))
		}

		for _, w := range workflows {
			switch w.Trigger.Type {
			case models.TriggerScheduled:
				logger.Info("registering cron runner", zap.String("workflow", w.Name))
				runnerManager.AddRunner(&runner.CronRunner{Workflow: w})
			case models.TriggerWebhook:
				logger.Info("registering webhook runner", zap.String("workflow", w.Name))
				runnerManager.AddRunner(&runner.WebhookRunner{Workflow: w})
			case models.TriggerWatch:
				logger.Info("registering watch runner", zap.String("workflow", w.Name))
				runnerManager.AddRunner(&runner.WatchRunner{Workflow: w})
			default:
				logger.Error("unknown trigger type", zap.String("workflow", w.Name), zap.String("trigger", string(w.Trigger.Type)))
			}
		}

		ctx := context.Background()
		if err := runnerManager.Run(ctx); err != nil {
			logger.Error("some runners failed to start", zap.Error(err))
		}

		// Start hot reload
		reloader := runner.NewHotReloader(config.GetString("workflows-dir"), runnerManager)
		if err := reloader.Start(ctx); err != nil {
			logger.Error("failed to start hot reload", zap.Error(err))
		}

		logger.Info("runner started, press Ctrl+C to stop")

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		logger.Info("shutting down runner")
		if err := reloader.Stop(); err != nil {
			logger.Error("error stopping hot reload", zap.Error(err))
		}
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer shutdownCancel()
		if err := runnerManager.Stop(shutdownCtx); err != nil {
			logger.Error("errors during shutdown", zap.Error(err))
		}
		logger.Info("runner stopped")
	},
}

func init() {
	rootCmd.AddCommand(runnerCmd)
}
