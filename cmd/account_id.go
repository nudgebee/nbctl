package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// resolveAccountID returns the account-id for the current invocation. It
// prefers an explicit --account-id flag, falls back to the configured
// profile via viper, and returns an error if neither is set.
func resolveAccountID(cmd *cobra.Command) (string, error) {
	accountID, _ := cmd.Flags().GetString("account-id")
	if accountID == "" {
		accountID = viper.GetString("account-id")
	}
	if accountID == "" {
		return "", fmt.Errorf("account-id is required; set it in your profile with `nbctl configure add` or pass --account-id")
	}
	return accountID, nil
}
