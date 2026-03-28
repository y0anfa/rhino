package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/y0anfa/rhino/internal/models"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <workflow>",
	Short: "Delete a workflow",
	Long:  `Delete a workflow file from the configured workflows directory.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Deleting workflow " + args[0])
		if err := models.DeleteWorkflow(args[0]); err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting workflow: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
