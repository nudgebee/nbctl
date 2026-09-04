package cmd

import (
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var nubiToolsCmd = &cobra.Command{
	Use:   "tools [account-id]",
	Short: "List registered tools accessible by AI agents",
	Long:  `Display registered tools, descriptions, status, and tool types.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		nubiClient, err := initNubiClientOptionalAccount(args)
		if err != nil {
			return err
		}

		tools, err := nubiClient.ListTools(cmd.Context())
		if err != nil {
			return err
		}

		if format.GetFormat().Get() == "json" {
			format.GetFormat().Print(tools)
			return nil
		}

		format.GetFormat().Print(format.TabularData{
			Data: tools,
			Fields: []format.TableField{
				{Header: "NAME", Field: "Name"},
				{Header: "STATUS", Field: "Status"},
				{Header: "TYPE", Field: "NBToolType"},
				{Header: "DESCRIPTION", Field: "Description"},
			},
		})
		return nil
	},
}

func init() {
	nubiCmd.AddCommand(nubiToolsCmd)
}
