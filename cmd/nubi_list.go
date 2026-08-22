package cmd

import (
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var nubiListLimit int

var nubiListCmd = &cobra.Command{
	Use:     "list [account-id]",
	Aliases: []string{"history"},
	Short:   "List conversation history",
	Long:    `Display recent conversation history with IDs, titles, and update timestamps.`,
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		nubiClient, err := initNubiClient(args)
		if err != nil {
			return err
		}

		limit, _ := cmd.Flags().GetInt("limit")
		if limit <= 0 {
			limit = 10
		}

		history, err := nubiClient.ShowHistory(limit)
		if err != nil {
			return err
		}

		if format.GetFormat().Get() == "json" {
			format.GetFormat().Print(history)
			return nil
		}

		format.GetFormat().Print(format.TabularData{
			Data: history,
			Fields: []format.TableField{
				{Header: "ID", Field: "ID"},
				{Header: "UPDATED AT", Field: "UpdatedAt"},
				{Header: "TITLE", Field: "Title"},
			},
		})
		return nil
	},
}

func init() {
	nubiListCmd.Flags().IntVarP(&nubiListLimit, "limit", "l", 10, "Maximum number of conversations to return")
	nubiCmd.AddCommand(nubiListCmd)
}
