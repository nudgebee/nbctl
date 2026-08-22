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

var nubiListLimit int

var nubiListCmd = &cobra.Command{
	Use:     "list [account-id]",
	Aliases: []string{"history"},
	Short:   "List conversation history",
	Long:    `Display recent conversation history with IDs, titles, and update timestamps.`,
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

		limit, _ := cmd.Flags().GetInt("limit")
		if limit <= 0 {
			limit = 10
		}

		history, err := nubiClient.ShowHistory(limit)
		if err != nil {
			return fmt.Errorf("failed to list conversations: %w", err)
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
