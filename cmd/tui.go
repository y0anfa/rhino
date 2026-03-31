package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/y0anfa/rhino/internal/tui"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Open the interactive TUI dashboard",
	Long:  `Launch a terminal-based dashboard for browsing workflows and execution history.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := tui.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
