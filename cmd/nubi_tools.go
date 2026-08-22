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

var nubiToolsCmd = &cobra.Command{
	Use:   "tools [account-id]",
	Short: "List registered tools accessible by AI agents",
	Long:  `Display tools accessible by Nubi AI agents, including their status and tool type.`,
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

		tools, err := nubiClient.ListTools()
		if err != nil {
			return fmt.Errorf("failed to list tools: %w", err)
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
