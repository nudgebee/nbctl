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

type statsRowDisplay struct {
	Cost        string `json:"cost"`
	InputTokens int    `json:"input_tokens"`
	OutputTokens int   `json:"output_tokens"`
}

var nubiStatsCmd = &cobra.Command{
	Use:     "stats <conversation-id>",
	Aliases: []string{"metrics", "usage"},
	Short:   "Get usage metrics and costs for a conversation",
	Long:    `Retrieve total cost, input tokens, and output tokens consumed by a specific conversation.`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		conversationID := args[0]

		accountID := viper.GetString("account-id")
		if accountID == "" {
			return fmt.Errorf("account-id is required, please set it in your config file or pass via flag")
		}

		username := viper.GetString("username")
		if username == "" {
			return fmt.Errorf("username is required, please set it in your config file")
		}

		endpoint := viper.GetString("endpoint")
		sessionID := uuid.New().String()
		nubiClient := nubi.New(client.NewClient(), accountID, username, sessionID, endpoint)

		stats, err := nubiClient.GetConversationStats(cmd.Context(), conversationID)
		if err != nil {
			return fmt.Errorf("failed to retrieve conversation stats: %w", err)
		}

		if format.GetFormat().Get() == "json" {
			format.GetFormat().Print(stats)
			return nil
		}

		rows := []statsRowDisplay{
			{
				Cost:         fmt.Sprintf("$%.6f", stats.TotalCost),
				InputTokens:  stats.TotalInputTokens,
				OutputTokens: stats.TotalOutputTokens,
			},
		}

		format.GetFormat().Print(format.TabularData{
			Data: rows,
			Fields: []format.TableField{
				{Header: "COST", Field: "Cost"},
				{Header: "INPUT TOKENS", Field: "InputTokens"},
				{Header: "OUTPUT TOKENS", Field: "OutputTokens"},
			},
		})
		return nil
	},
}

func init() {
	nubiCmd.AddCommand(nubiStatsCmd)
}
