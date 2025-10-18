package cmd

import (
	"github.com/spf13/cobra"
)

var usersCmd = &cobra.Command{
	Use:   "users",
	Short: "Manage users",
	Long:  `Add and update users.`,
}

func init() {
	rootCmd.AddCommand(usersCmd)
}
