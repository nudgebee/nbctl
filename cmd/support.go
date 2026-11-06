package cmd

import (
	"github.com/spf13/cobra"
)

var supportCmd = &cobra.Command{
	Use:   "support",
	Short: "Collect information for Nudgebee support",
	Long:  `Collect diagnostics from a self-hosted Nudgebee install to send to support.`,
}

func init() {
	rootCmd.AddCommand(supportCmd)
}
