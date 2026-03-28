package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/y0anfa/rhino/internal/models"
)

var newCmd = &cobra.Command{
	Use:   "new <name>",
	Short: "Create a new workflow",
	Long:  `Create a new workflow with a default template including a cron trigger and a shell task.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		workflow := models.NewWorkflow(args[0], "A new workflow")
		trigger := models.NewTrigger("trigger1", "A new trigger", models.TriggerScheduled, "*/5 * * * *")
		task := models.NewTask("task1", "A new task", "shell", map[string]interface{}{"command": "echo", "args": []interface{}{"Hello world!"}})
		workflow.SetTrigger(*trigger)
		workflow.AddTask(*task)
		workflow.Order = [][]string{{"task1"}}
		if err := workflow.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating workflow: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(newCmd)
}
