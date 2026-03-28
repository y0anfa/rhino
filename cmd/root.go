package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/y0anfa/rhino/internal/config"
	"github.com/y0anfa/rhino/internal/logger"
	"go.uber.org/zap"
)

var rootCmd = &cobra.Command{
	Use:   "rhino",
	Short: "A lightweight workflow automation engine",
	Long:  `Rhino is a mini-Airflow that lets you define, schedule, and execute workflows using YAML definitions with cron and webhook triggers.`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	logger.Info("initializing config")
	if config.ConfigFileExists() {
		logger.Info("config file exists")
	} else {
		logger.Info("config file does not exist, creating default")
		err := config.SaveDefaultConfig()
		if err != nil {
			logger.Fatal("failed to save default config", zap.Error(err))
		}
	}
}