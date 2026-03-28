package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/y0anfa/rhino/internal/models"
)

var dryRun bool

var runCmd = &cobra.Command{
	Use:   "run <workflow>",
	Short: "Manually run a specific workflow",
	Long:  `Load and execute a workflow by name. Use --dry-run to validate and preview the execution plan without running.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		w, err := models.LoadWorkflow(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading workflow: %v\n", err)
			os.Exit(1)
		}

		if dryRun {
			if err := w.Validate(); err != nil {
				fmt.Fprintf(os.Stderr, "Validation failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Workflow: %s\n", w.Name)
			fmt.Printf("Settings: max-tries=%d, timeout=%s\n", w.Settings.MaxTries, w.Settings.Timeout)
			fmt.Println("\nExecution plan:")
			for i, group := range w.Order {
				fmt.Printf("  Step %d: [%s]\n", i+1, strings.Join(group, ", "))
				for _, taskName := range group {
					task := w.GetTask(taskName)
					if task != nil {
						fmt.Printf("    - %s (provider: %s)\n", task.Name, task.Provider)
					}
				}
			}
			fmt.Println("\nDry run complete. No tasks were executed.")
			return
		}

		results, err := w.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error running workflow: %v\n", err)
			os.Exit(1)
		}
		for taskName, result := range results {
			if result != nil && result.Output != "" {
				fmt.Printf("[%s] %s", taskName, result.Output)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate and show execution plan without running")
}
