package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/guptarohit/asciigraph"
	"github.com/spf13/cobra"

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

// MetricsQueryQuery runs a metrics query (PromQL or the account's own language)
// through the metrics_list action. results is passed through untouched.
const MetricsQueryQuery = `query MetricsQuery($request: FetchMetricsRequest!) {
  metrics_list(request: $request) {
    results
  }
}`

type MetricsQueryResponse struct {
	MetricsQuery struct {
		Results []MetricsResponse `json:"results"`
	} `json:"metrics_list"`
}

// metricsQueryRawResponse keeps results as sent by the API, for -o json.
type metricsQueryRawResponse struct {
	MetricsQuery struct {
		Results json.RawMessage `json:"results"`
	} `json:"metrics_list"`
}

type MetricsResponse struct {
	QueryKey string          `json:"query_key"`
	Payload  []MetricsResult `json:"payload"`
	Error    *string         `json:"error,omitempty"`
	Note     string          `json:"note,omitempty"`
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

		accountId, err := resolveAccountID(cmd)
		if err != nil {
			return err
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

		step, _ := cmd.Flags().GetDuration("step")
		if step < 0 {
			return fmt.Errorf("invalid step: must not be negative")
		}

		request := map[string]any{
			"account_id": accountId,
			"queries":    queries,
			"instant":    instant,
			"start_time": startTime.UnixMilli(),
			"end_time":   endTime.UnixMilli(),
		}
		if step > 0 {
			// step_interval is whole seconds; round a sub-second step up to 1s.
			request["step_interval"] = max(1, int(step.Round(time.Second)/time.Second))
		}

		req := client.NewRequest(MetricsQueryQuery)
		req.Var("request", request)

		var respData metricsQueryRawResponse
		if err := graphqlClient.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		raw := respData.MetricsQuery.Results
		if len(raw) == 0 || string(raw) == "null" {
			raw = json.RawMessage("[]")
		}

		// JSON output passes results through, so only text output needs them to
		// decode; for JSON the decode is best-effort, for the warnings below.
		jsonOutput := format.GetFormat().Get() == "json"
		var results []MetricsResponse
		decodeErr := json.Unmarshal(raw, &results)
		if decodeErr != nil && !jsonOutput {
			return fmt.Errorf("failed to decode metrics results: %w", decodeErr)
		}
		series, failed := 0, false
		for _, r := range results {
			series += len(r.Payload)
			if r.Error != nil && *r.Error != "" {
				failed = true
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: query %q failed: %s\n", r.QueryKey, *r.Error)
			} else if r.Note != "" {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Note: %s\n", r.Note)
			}
		}

		// Only when the results decoded: otherwise series is unknown, not zero.
		if decodeErr == nil && series == 0 && !failed {
			when := fmt.Sprintf("between %s and %s", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
			if instant {
				when = "at " + endTime.Format(time.RFC3339)
			}
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "No data: the query returned no series %s.\n", when)
		}

		// JSON output is the backend's results, unchanged, so scripts can use it as is.
		if jsonOutput {
			return format.GetFormat().PrintRawJSON(raw)
		}

		if len(results) == 0 {
			return nil
		}

		var displayPayload []DisplayMetricsResult
		for _, r := range results[0].Payload {
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
			renderChart(results[0].Payload)
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
	metricsQueryCmd.Flags().Duration("step", 0, "Resolution step for range queries, e.g. 30s, 5m (default: chosen by the backend)")
}
