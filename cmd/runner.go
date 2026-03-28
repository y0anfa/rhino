package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/y0anfa/rhino/internal/logger"
	"github.com/y0anfa/rhino/internal/models"
	"github.com/y0anfa/rhino/internal/runner"
	"go.uber.org/zap"
)

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
			default:
				logger.Error("unknown trigger type", zap.String("workflow", w.Name), zap.String("trigger", string(w.Trigger.Type)))
			}
		}

		ctx := context.Background()
		runnerManager.Run(ctx)

		logger.Info("runner started, press Ctrl+C to stop")

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		logger.Info("shutting down runner")
		runnerManager.Stop(ctx)
		logger.Info("runner stopped")
	},
}

func init() {
	rootCmd.AddCommand(runnerCmd)
}
