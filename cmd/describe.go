package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/y0anfa/rhino/internal/models"
)

var describeCmd = &cobra.Command{
	Use:   "describe <workflow>",
	Short: "Show workflow details",
	Long:  `Display detailed information about a specific workflow including its trigger, tasks, and execution order.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		w, err := models.LoadWorkflow(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading workflow: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(w.Describe())
	},
}

func init() {
	rootCmd.AddCommand(describeCmd)
}
