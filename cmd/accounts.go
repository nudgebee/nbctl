package cmd

import (
	"github.com/spf13/cobra"
)

var accountsCmd = &cobra.Command{
	Use:   "accounts",
	Short: "Manage accounts",
	Long:  `Create, list, update, and delete accounts.`,
}

func init() {
	rootCmd.AddCommand(accountsCmd)
	accountsCmd.AddCommand(accountsCreateCmd)
	accountsCmd.AddCommand(accountsEnableCmd)
	accountsCmd.AddCommand(accountsDisableCmd)
	accountsCmd.AddCommand(accountsCreateAgentTokenCmd)
}
