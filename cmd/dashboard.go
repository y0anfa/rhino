package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/y0anfa/rhino/internal/logger"
	"github.com/y0anfa/rhino/internal/web"
	"go.uber.org/zap"
)

var dashboardPort int

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Start the web monitoring dashboard",
	Long:  `Launch a browser-based dashboard for monitoring workflows and execution history.`,
	Run: func(cmd *cobra.Command, args []string) {
		srv := web.NewServer(dashboardPort)
		if err := srv.Start(); err != nil {
			logger.Fatal("failed to start dashboard", zap.Error(err))
		}

		logger.Info("dashboard running", zap.Int("port", dashboardPort))

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Stop(ctx)
	},
}

func init() {
	dashboardCmd.Flags().IntVar(&dashboardPort, "port", 9090, "Dashboard HTTP port")
	rootCmd.AddCommand(dashboardCmd)
}
