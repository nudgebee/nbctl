package cmd

import (
	"github.com/spf13/cobra"
)

var optimizationsCmd = &cobra.Command{
	Use:   "optimizations",
	Short: "Manage optimizations",
	Long:  `List, search, and describe optimizations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return optimizationsListCmd.RunE(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(optimizationsCmd)
}
