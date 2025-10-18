package cmd

import (
	"github.com/spf13/cobra"
)

var tracesCmd = &cobra.Command{
	Use:   "traces",
	Short: "Query traces",
	Long:  `Query traces from the Nudgebee API.`,
}

func init() {
	rootCmd.AddCommand(tracesCmd)
}
