package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of nbctl",
	Long:  `All software has versions. This is nbctl's.`, 
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("nbctl version " + Version)
	},
}

func init() {
}
