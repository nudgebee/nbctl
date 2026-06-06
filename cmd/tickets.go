package cmd

import (
	"github.com/spf13/cobra"
)

var ticketsCmd = &cobra.Command{
	Use:   "tickets",
	Short: "Manage tickets",
	Long:  `List ticket integration configurations and create tickets in external systems (Jira, ServiceNow, PagerDuty, GitHub, GitLab, Zenduty).`,
}

func init() {
	rootCmd.AddCommand(ticketsCmd)
}
