package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/guptarohit/asciigraph"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
)

func renderChart(payload []MetricsResult) {
	if len(payload) == 0 {
		format.GetFormat().Print("No data to plot")
		return
	}

	for _, p := range payload {
		graph := asciigraph.Plot(p.Values, asciigraph.Caption(fmt.Sprintf("%v", p.Metric)))
		format.GetFormat().Print(graph)
		fmt.Println()
	}
}

type MetricsQueryResponse struct {
	MetricsQuery struct {
		Results []MetricsResponse `json:"results"`
	} `json:"metrics_list"`
}

type MetricsResponse struct {
	QueryKey string          `json:"query_key"`
	Payload  []MetricsResult `json:"payload"`
}

type MetricsResult struct {
	Metric     map[string]string `json:"metric"`
	Timestamps []float64         `json:"timestamps"`
	Values     []float64         `json:"values"`
}

type DisplayMetricsResult struct {
	Metric     string
	Timestamps string
	Values     string
}

var metricsQueryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query metrics",
	RunE: func(cmd *cobra.Command, args []string) error {
		graphqlClient := client.NewClient()

		accountId, _ := cmd.Flags().GetString("account-id")
		if accountId == "" {
			accountId = viper.GetString("account-id")
		}
		if accountId == "" {
			return fmt.Errorf("account-id is required")
		}

		queriesStr, _ := cmd.Flags().GetString("query")
		if queriesStr == "" {
			return fmt.Errorf("query is required")
		}

		queries := map[string]any{
			"query": queriesStr,
		}

		startTimeStr, _ := cmd.Flags().GetString("start-time")
		endTimeStr, _ := cmd.Flags().GetString("end-time")
		instant, _ := cmd.Flags().GetBool("instant")
		chart, _ := cmd.Flags().GetBool("chart")

		if startTimeStr == "" {
			startTimeStr = time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
		}
		if endTimeStr == "" {
			endTimeStr = time.Now().Format(time.RFC3339)
		}

		startTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			return fmt.Errorf("invalid start-time format: %w", err)
		}
		endTime, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			return fmt.Errorf("invalid end-time format: %w", err)
		}

		// Convert to Unix milliseconds
		startTimeMs := float64(startTime.UnixNano() / int64(time.Millisecond))
		endTimeMs := float64(endTime.UnixNano() / int64(time.Millisecond))

		req := client.NewRequest(`
			query MetricsQuery(
				$account_id: String!
				$queries: jsonb!
				$instant: Boolean!
				$start_time: Float!
				$end_time: Float!
			) {
				metrics_list(
					request: {
						account_id: $account_id
						queries: $queries
						instant: $instant
						end_time: $end_time
						start_time: $start_time
					}
				) {
					results
				}
			}
		`)

		req.Var("account_id", accountId)
		req.Var("queries", queries)
		req.Var("instant", instant)
		req.Var("start_time", startTimeMs)
		req.Var("end_time", endTimeMs)

		var respData MetricsQueryResponse
		if err := graphqlClient.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		if len(respData.MetricsQuery.Results) == 0 {
			format.GetFormat().Print("No Data")
			return nil
		}

		var displayPayload []DisplayMetricsResult
		for _, r := range respData.MetricsQuery.Results[0].Payload {
			metricJSON, err := json.Marshal(r.Metric)
			if err != nil {
				return fmt.Errorf("failed to marshal metric to JSON: %w", err)
			}

			timestampsJSON, err := json.Marshal(r.Timestamps)
			if err != nil {
				return fmt.Errorf("failed to marshal timestamps to JSON: %w", err)
			}

			valuesJSON, err := json.Marshal(r.Values)
			if err != nil {
				return fmt.Errorf("failed to marshal values to JSON: %w", err)
			}

			displayPayload = append(displayPayload, DisplayMetricsResult{
				Metric:     string(metricJSON),
				Timestamps: string(timestampsJSON),
				Values:     string(valuesJSON),
			})
		}

		table := format.TabularData{
			Data: displayPayload,
			Fields: []format.TableField{
				{Header: "Metric", Field: "Metric"},
				{Header: "Timestamps", Field: "Timestamps"},
				{Header: "Values", Field: "Values"},
			},
		}
		if chart {
			renderChart(respData.MetricsQuery.Results[0].Payload)
		} else {
			format.GetFormat().Print(table)
		}
		return nil
	},
}

func init() {
	metricsCmd.AddCommand(metricsQueryCmd)
	metricsQueryCmd.Flags().String("query", "", "Metrics Query")
	metricsQueryCmd.Flags().String("start-time", "", "Start time (RFC3339)")
	metricsQueryCmd.Flags().String("end-time", "", "End time (RFC3339)")
	metricsQueryCmd.Flags().String("account-id", "", "Account ID")
	metricsQueryCmd.Flags().Bool("instant", false, "Instant query")
	metricsQueryCmd.Flags().Bool("chart", false, "Display data as a chart")
}
