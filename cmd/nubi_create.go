package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var nubiCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Start a new Nubi conversation",
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "nubi create called"); err != nil {
			_ = err
		}
		return nil
	},
}

func init() {
	nubiCmd.AddCommand(nubiCreateCmd)
}
