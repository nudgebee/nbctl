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
	Cost         string `json:"cost"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
	CachedTokens int    `json:"cached_tokens"`
	HitRate      string `json:"hit_rate"`
	WallTime     string `json:"wall_time"`
}

var nubiStatsCmd = &cobra.Command{
	Use:     "stats <conversation-id>",
	Aliases: []string{"metrics", "usage"},
	Short:   "Get usage metrics and costs for a conversation",
	Long:    `Retrieve total cost, input tokens, output tokens, cache savings, and execution duration for a conversation.`,
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

		costVal := getFloat64FromMap(stats, "total_cost_usd", "total_cost")
		inputTokens := getIntFromMap(stats, "total_input_tokens")
		outputTokens := getIntFromMap(stats, "total_output_tokens")
		cachedTokens := getIntFromMap(stats, "total_cached_input_tokens")
		hitRateVal := getFloat64FromMap(stats, "total_cache_hit_rate_percentage")
		wallTimeVal := getFloat64FromMap(stats, "wall_time_seconds")

		wallTimeStr := "-"
		if wallTimeVal > 0 {
			wallTimeStr = fmt.Sprintf("%.2fs", wallTimeVal)
		}

		rows := []statsRowDisplay{
			{
				Cost:         fmt.Sprintf("$%.6f", costVal),
				InputTokens:  inputTokens,
				OutputTokens: outputTokens,
				CachedTokens: cachedTokens,
				HitRate:      fmt.Sprintf("%.1f%%", hitRateVal),
				WallTime:     wallTimeStr,
			},
		}

		format.GetFormat().Print(format.TabularData{
			Data: rows,
			Fields: []format.TableField{
				{Header: "COST", Field: "Cost"},
				{Header: "INPUT TOKENS", Field: "InputTokens"},
				{Header: "OUTPUT TOKENS", Field: "OutputTokens"},
				{Header: "CACHED TOKENS", Field: "CachedTokens"},
				{Header: "HIT RATE", Field: "HitRate"},
				{Header: "WALL TIME", Field: "WallTime"},
			},
		})
		return nil
	},
}

func getFloat64FromMap(m map[string]any, keys ...string) float64 {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			switch val := v.(type) {
			case float64:
				return val
			case int:
				return float64(val)
			case int64:
				return float64(val)
			}
		}
	}
	return 0
}

func getIntFromMap(m map[string]any, keys ...string) int {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			switch val := v.(type) {
			case float64:
				return int(val)
			case int:
				return val
			case int64:
				return int(val)
			}
		}
	}
	return 0
}

func init() {
	nubiCmd.AddCommand(nubiStatsCmd)
}
