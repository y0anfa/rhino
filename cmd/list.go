package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/y0anfa/rhino/internal/models"
)

var listOutputFormat string

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

		switch listOutputFormat {
		case "json":
			data, _ := json.MarshalIndent(workflows, "", "  ")
			fmt.Println(string(data))
		default:
			for _, w := range workflows {
				fmt.Println(w)
			}
		}
	},
}

func init() {
	listCmd.Flags().StringVarP(&listOutputFormat, "output", "o", "text", "Output format: text, json")
	rootCmd.AddCommand(listCmd)
}
