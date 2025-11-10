package cmd

import (
	"github.com/spf13/cobra"
)

var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Query metrics",
	Long:  `Query metrics from the Nudgebee API.`,
}

func init() {
	rootCmd.AddCommand(metricsCmd)
}
