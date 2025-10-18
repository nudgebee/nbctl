package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var suggestCmd = &cobra.Command{
	Use:   "suggest",
	Short: "Get suggestions for a Nubi conversation",
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "nubi suggest called"); err != nil {
			_ = err
		}
		return nil
	},
}

func init() {
	nubiCmd.AddCommand(suggestCmd)
}
