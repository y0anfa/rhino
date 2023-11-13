/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.com/y0anfa/rhino/internal/workflow"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		w, err := workflow.LoadWorkflow("workflows", args[0])
		if err != nil {
			panic(err)
		} else {
			err := w.Run()
			if err != nil {
				panic(err)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
