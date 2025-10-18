package cmd

import (
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Query logs",
	Long:  `Query logs from the Nudgebee API.`,
}

func init() {
	rootCmd.AddCommand(logsCmd)
}
