package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/y0anfa/rhino/internal/models"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available workflows",
	Long:  `List all workflow names found in the configured workflows directory.`,
	Run: func(cmd *cobra.Command, args []string) {
		workflows, err := models.ListWorkflows()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing workflows: %v\n", err)
			os.Exit(1)
		}
		for _, w := range workflows {
			fmt.Println(w)
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
