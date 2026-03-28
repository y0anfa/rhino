package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/y0anfa/rhino/internal/models"
)

var validateCmd = &cobra.Command{
	Use:   "validate <workflow>",
	Short: "Validate a workflow definition",
	Long:  `Load and validate a workflow YAML file without executing it. Reports success or detailed validation errors.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		w, err := models.LoadWorkflow(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading workflow: %v\n", err)
			os.Exit(1)
		}
		if err := w.Validate(); err != nil {
			fmt.Fprintf(os.Stderr, "Validation failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Workflow '%s' is valid.\n", w.Name)
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
