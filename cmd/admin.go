package cmd

import (
	"github.com/spf13/cobra"
)

var adminCmd = &cobra.Command{
	Use:   "admin",
	Short: "Administer the platform",
	Long:  `Perform administrative tasks.`,
}

func init() {
	rootCmd.AddCommand(adminCmd)
}
