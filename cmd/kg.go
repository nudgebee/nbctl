package cmd

import (
	"github.com/spf13/cobra"
)

var kgCmd = &cobra.Command{
	Use:   "kg",
	Short: "Knowledge Graph service topology and manual dependency management",
	Long:  `Query service topology, explore dependency graphs, and manage manual dependency overrides in Nudgebee Knowledge Graph (KG).`,
}

func init() {
	rootCmd.AddCommand(kgCmd)
}
