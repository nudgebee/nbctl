package cmd

import (
	"github.com/spf13/cobra"
)

var adminUsersCmd = &cobra.Command{
	Use:   "users",
	Short: "Manage users",
	Long:  `Add, update, and remove users.`,
}

func init() {
	adminCmd.AddCommand(adminUsersCmd)
}
