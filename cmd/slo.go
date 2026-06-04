package cmd

import (
	"github.com/spf13/cobra"
)

var sloCmd = &cobra.Command{
	Use:   "slo",
	Short: "Manage Service Level Objectives",
	Long:  `Query Service Level Objective (SLO) configurations and reports.`,
}

func init() {
	rootCmd.AddCommand(sloCmd)
}
