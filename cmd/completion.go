package cmd

import (
	"github.com/spf13/cobra"
	"github.com/y0anfa/rhino/internal/models"
)

func completeWorkflowNames(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	workflows, err := models.ListWorkflows()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return workflows, cobra.ShellCompDirectiveNoFileComp
}
