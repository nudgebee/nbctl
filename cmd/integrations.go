package cmd

import (
	"github.com/spf13/cobra"
)

var integrationsCmd = &cobra.Command{
	Use:   "integrations",
	Short: "Manage integrations",
	Long:  `List integrations configured for your tenant.`,
}

func init() {
	rootCmd.AddCommand(integrationsCmd)
}
