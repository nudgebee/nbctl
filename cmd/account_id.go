package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// resolveAccountID returns the account-id for the current invocation. It
// prefers an explicit --account-id flag, falls back to the configured
// profile via viper, and returns an error if neither is set.
func resolveAccountID(cmd *cobra.Command) (string, error) {
	accountID, err := cmd.Flags().GetString("account-id")
	if err != nil {
		accountID = ""
	}
	if accountID == "" {
		accountID = viper.GetString("account-id")
	}
	if accountID == "" {
		return "", fmt.Errorf("account-id is required; set it in your profile with `nbctl configure add` or pass --account-id")
	}
	return accountID, nil
}

// resolveAccountIDWithPositional returns the account-id prioritizing an explicit
// --account-id flag (if set), followed by a positional argument, and falling
// back to profile config via viper.
func resolveAccountIDWithPositional(cmd *cobra.Command, args []string) (string, error) {
	if cmd.Flags().Changed("account-id") {
		flagVal, err := cmd.Flags().GetString("account-id")
		if err == nil && strings.TrimSpace(flagVal) != "" {
			return strings.TrimSpace(flagVal), nil
		}
	}
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		return strings.TrimSpace(args[0]), nil
	}
	accountID, err := resolveAccountID(cmd)
	if err != nil {
		return "", fmt.Errorf("resolving account ID: %w", err)
	}
	return accountID, nil
}
