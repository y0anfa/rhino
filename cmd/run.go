package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/y0anfa/rhino/internal/models"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Manually run a specific workflow",
	Long:  `Load and execute a workflow by name. The workflow must exist in the configured workflows directory.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		w, err := models.LoadWorkflow(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading workflow: %v\n", err)
			os.Exit(1)
		}
		if err := w.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running workflow: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
