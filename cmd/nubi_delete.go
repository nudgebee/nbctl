package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var nubiDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a Nubi conversation",
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "nubi delete called"); err != nil {
			_ = err
		}
		return nil
	},
}

func init() {
	nubiCmd.AddCommand(nubiDeleteCmd)
}
