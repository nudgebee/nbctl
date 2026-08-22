package cmd

import (
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var nubiPlaybooksCmd = &cobra.Command{
	Use:   "playbooks [account-id]",
	Short: "List investigation playbooks for the account",
	Long:  `Display agent automation investigation playbooks configured for the account.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		nubiClient, err := initNubiClient(args)
		if err != nil {
			return err
		}

		playbooks, err := nubiClient.ListPlaybooks(cmd.Context())
		if err != nil {
			return err
		}

		if format.GetFormat().Get() == "json" {
			format.GetFormat().Print(playbooks)
			return nil
		}

		format.GetFormat().Print(format.TabularData{
			Data: playbooks,
			Fields: []format.TableField{
				{Header: "ID", Field: "ID"},
				{Header: "ALERT NAME", Field: "AlertName"},
				{Header: "SOURCE", Field: "Source"},
				{Header: "PROCESSOR", Field: "Processor"},
			},
		})
		return nil
	},
}

func init() {
	nubiCmd.AddCommand(nubiPlaybooksCmd)
}
