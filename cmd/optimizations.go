package cmd

import (
	"github.com/spf13/cobra"
)

var optimizationsCmd = &cobra.Command{
	Use:   "optimizations",
	Short: "Manage optimizations",
	Long:  `List, search, and describe optimizations.`,
}

func init() {
	rootCmd.AddCommand(optimizationsCmd)
}
