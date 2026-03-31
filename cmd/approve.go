package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/y0anfa/rhino/internal/providers"
)

var approveCmd = &cobra.Command{
	Use:   "approve <approval-id>",
	Short: "Approve a pending approval gate",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := providers.Approve(args[0]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Approval '%s' approved.\n", args[0])
	},
}

var rejectCmd = &cobra.Command{
	Use:   "reject <approval-id>",
	Short: "Reject a pending approval gate",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := providers.Reject(args[0]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Approval '%s' rejected.\n", args[0])
	},
}

func init() {
	rootCmd.AddCommand(approveCmd)
	rootCmd.AddCommand(rejectCmd)
}
