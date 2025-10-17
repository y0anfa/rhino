/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
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
)

// runnerCmd represents the runner command
var runnerCmd = &cobra.Command{
	Use:   "runner",
	Short: "Start the workflow runner daemon",
	Long: `Start the runner daemon which will:
- Start cron schedulers for workflows with cron triggers
- Start webhook server for workflows with webhook triggers
- Monitor and execute workflows based on their triggers`,
	Run: func(cmd *cobra.Command, args []string) {
		logger.Info("starting runner...")

		// Create runner manager
		runnerManager := runner.NewRunnerManager()

		// Load all workflows
		workflows, err := models.LoadWorkflows()
		if err != nil {
			logger.Fatal("error while loading workflows: ", err)
		}

		// Register runners for each workflow
		for _, w := range workflows {
			switch w.Trigger.Type {
			case models.TriggerScheduled:
				logger.Info("registering cron runner for workflow ", w.Name)
				runnerManager.AddRunner(&runner.CronRunner{Workflow: w})
			case models.TriggerWebhook:
				logger.Info("registering webhook runner for workflow ", w.Name)
				runnerManager.AddRunner(&runner.WebhookRunner{Workflow: w})
			default:
				logger.Error("unknown trigger type for workflow ", w.Name, ": ", w.Trigger.Type)
			}
		}

		// Start all runners
		ctx := context.Background()
		runnerManager.Run(ctx)

		logger.Info("runner started successfully. Press Ctrl+C to stop.")

		// Wait for interrupt signal
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		// Graceful shutdown
		logger.Info("shutting down runner...")
		runnerManager.Stop(ctx)
		logger.Info("runner stopped")
	},
}

func init() {
	rootCmd.AddCommand(runnerCmd)
}
