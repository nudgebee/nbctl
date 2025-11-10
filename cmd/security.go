package cmd

import (
	"github.com/spf13/cobra"
)

var securityCmd = &cobra.Command{
	Use:   "security",
	Short: "Manage security-related resources",
	Long:  `Provides commands for managing and querying security-related resources like image vulnerabilities and recommendations.`,
}

func init() {
	rootCmd.AddCommand(securityCmd)
}
