package cmd

import (
	"github.com/spf13/cobra"
)

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Manage events",
	Long:  `List, search, and describe events.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return eventsListCmd.RunE(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(eventsCmd)
	eventsCmd.PersistentFlags().String("account-id", "", "Account ID")
	eventsCmd.PersistentFlags().String("start-time", "", "Start time (RFC3339)")
	eventsCmd.PersistentFlags().String("end-time", "", "End time (RFC3339)")
	eventsCmd.PersistentFlags().Int("limit", 10, "Limit")
	eventsCmd.PersistentFlags().Int("offset", 0, "Offset")
	eventsCmd.PersistentFlags().String("subject", "", "Filter by subject (ilike)")
	eventsCmd.PersistentFlags().String("status", "", "Filter by status")
	eventsCmd.PersistentFlags().String("title", "", "Filter by title (ilike)")
	eventsCmd.PersistentFlags().String("priority", "", "Filter by priority")
}
