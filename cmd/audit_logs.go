package cmd

import (
	"github.com/spf13/cobra"
)

var auditLogsCmd = &cobra.Command{
	Use:   "audit-logs",
	Short: "Query audit events",
	Long:  `Query audit events recorded for the current account.`,
}

func init() {
	rootCmd.AddCommand(auditLogsCmd)
}
