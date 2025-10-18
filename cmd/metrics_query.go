package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/machinebox/graphql"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"nudgebee.com/nbctl/pkg/client"
	"nudgebee.com/nbctl/pkg/format"
)

var metricsQueryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query metrics",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := client.NewClient()

		accountId, _ := cmd.Flags().GetString("account-id")
		if accountId == "" {
			accountId = viper.GetString("account-id")
		}
		if accountId == "" {
			return fmt.Errorf("account-id is required")
		}

		startTimeStr, _ := cmd.Flags().GetString("start-time")
		endTimeStr, _ := cmd.Flags().GetString("end-time")
		queryStr, _ := cmd.Flags().GetString("query")
		metricProvider, _ := cmd.Flags().GetString("metric-provider")
		onlyMetric, _ := cmd.Flags().GetBool("only-metric")

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

		req := graphql.NewRequest(`
			mutation FetchMetrics($request: FetchMetricsListRequest!) {
				metrics_list(request: $request) {
					metric
					attributes
				}
			}
		`)

		requestVars := map[string]interface{}{
			"account_id": accountId,
			"start_time": startTimeMs,
			"end_time":   endTimeMs,
			"request":    json.RawMessage(fmt.Sprintf("{\"query\":%s}", strconv.Quote(queryStr))),
		}

		if metricProvider != "" {
			requestVars["metric_provider"] = metricProvider
		}

		req.Var("request", requestVars)

		var respData struct {
			MetricsList []struct {
				Metric     string          `json:"metric"`
				Attributes json.RawMessage `json:"attributes"`
			} `json:"metrics_list"`
		}

		if err := client.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		// Define a struct for display purposes
		type MetricEntryDisplay struct {
			Metric     string `json:"metric"`
			Attributes string `json:"attributes"` // Attributes will be pretty-printed string
		}

		var displayMetrics []MetricEntryDisplay
		for _, metricEntry := range respData.MetricsList {
			var attributesData interface{}
			prettyAttributes := ""
			if len(metricEntry.Attributes) > 0 {
				if err := json.Unmarshal(metricEntry.Attributes, &attributesData); err != nil {
					prettyAttributes = string(metricEntry.Attributes) // Fallback to raw if unmarshal fails
				} else {
					prettyBytes, err := json.MarshalIndent(attributesData, "", "  ")
					if err != nil {
						prettyAttributes = string(metricEntry.Attributes) // Fallback to raw if marshal fails
					} else {
						prettyAttributes = string(prettyBytes)
					}
				}
			}

			displayMetrics = append(displayMetrics, MetricEntryDisplay{
				Metric:     metricEntry.Metric,
				Attributes: prettyAttributes,
			})
		}

		if onlyMetric {
			for _, metricEntry := range displayMetrics {
				fmt.Println(metricEntry.Metric)
			}
		} else {
			table := format.TabularData{
				Data: displayMetrics,
				Fields: []format.TableField{
					{Header: "Metric", Field: "Metric"},
					{Header: "Attributes", Field: "Attributes"},
				},
			}
			format.GetFormat().Print(table)
		}

		return nil
	},
}

func init() {
	metricsCmd.AddCommand(metricsQueryCmd)
	metricsQueryCmd.Flags().String("query", "", "Metric query")
	metricsQueryCmd.Flags().String("start-time", "", "Start time (RFC3339)")
	metricsQueryCmd.Flags().String("end-time", "", "End time (RFC3339)")
	metricsQueryCmd.Flags().String("account-id", "", "Account ID")
	metricsQueryCmd.Flags().String("metric-provider", "", "Metric Provider")
	metricsQueryCmd.Flags().Bool("only-metric", false, "Show only metric names")
}
