package cmd

import (
	"github.com/spf13/cobra"
)

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Manage events",
	Long:  `List, search, and describe events.`,
}

func init() {
	rootCmd.AddCommand(eventsCmd)
}
