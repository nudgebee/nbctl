package cmd

import (
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication, users, custom roles, user groups, and role assignments",
	Long:  `Manage tenant users, custom roles, user groups, and role assignments in Nudgebee.`,
}

func init() {
	rootCmd.AddCommand(authCmd)
}
