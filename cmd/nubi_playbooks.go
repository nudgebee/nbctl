package cmd

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/nudgebee/nbctl/pkg/nubi"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var nubiPlaybooksCmd = &cobra.Command{
	Use:   "playbooks [account-id]",
	Short: "List investigation playbooks for the account",
	Long:  `Display agent automation investigation playbooks configured for the account.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var accountID string
		if len(args) > 0 {
			accountID = args[0]
		} else {
			accountID = viper.GetString("account-id")
		}

		if accountID == "" {
			return fmt.Errorf("account-id is required, please provide it as an argument or set it in your config file")
		}

		username := viper.GetString("username")
		if username == "" {
			return fmt.Errorf("username is required, please set it in your config file")
		}

		endpoint := viper.GetString("endpoint")
		sessionID := uuid.New().String()
		nubiClient := nubi.New(client.NewClient(), accountID, username, sessionID, endpoint)

		playbooks, err := nubiClient.ListPlaybooks(cmd.Context())
		if err != nil {
			return fmt.Errorf("failed to list playbooks: %w", err)
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
