package cmd

import (
	"github.com/spf13/cobra"
)

var workflowCmd = &cobra.Command{
	Use:   "workflow",
	Short: "Manage workflows",
	Long:  `Manage workflows including listing, getting details, applying configurations, and triggering executions.`,
}

func init() {
	rootCmd.AddCommand(workflowCmd)
}
