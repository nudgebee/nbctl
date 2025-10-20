package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/machinebox/graphql"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"nudgebee.com/nbctl/pkg/client"
	"nudgebee.com/nbctl/pkg/format"
)

var logsQueryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query logs",
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
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")

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
		startTimeMs := startTime.UnixNano() / int64(time.Millisecond)
		endTimeMs := endTime.UnixNano() / int64(time.Millisecond)

		// Construct the inner query string
		innerQuery := fmt.Sprintf("query=%s&start=%d&end=%d&limit=%d", queryStr, startTime.UnixNano(), endTime.UnixNano(), limit)

		req := graphql.NewRequest(`
			mutation FetchLogs($request: FetchLogRequest!) {
				logs_query(request: $request) {
					timestamp
					severity
					message
					labels
				}
			}
		`)

		requestVars := map[string]any{
			"account_id": accountId,
			"end_time":   endTimeMs,
			"start_time": startTimeMs,
			"query":      innerQuery,
			"limit":      limit,
			"offset":     offset,
		}
		req.Var("request", requestVars)

		var respData struct {
			LogsList []struct {
				Timestamp string          `json:"timestamp"`
				Severity  string          `json:"severity"`
				Message   string          `json:"message"`
				Labels    json.RawMessage `json:"labels"`
			} `json:"logs_list"`
		}

		if err := client.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		table := format.TabularData{
			Data: respData.LogsList,
			Fields: []format.TableField{
				{Header: "Timestamp", Field: "Timestamp"},
				{Header: "Severity", Field: "Severity"},
				{Header: "Message", Field: "Message"},
				{Header: "Labels", Field: "Labels"},
			},
		}
		format.GetFormat().Print(table)

		return nil
	}}

func init() {
	logsCmd.AddCommand(logsQueryCmd)
	logsQueryCmd.Flags().String("account-id", "", "Account ID")
	logsQueryCmd.Flags().String("start-time", "", "Start time (RFC3339)")
	logsQueryCmd.Flags().String("end-time", "", "End time (RFC3339)")
	logsQueryCmd.Flags().String("query", "", "Log query")
	logsQueryCmd.Flags().Int("limit", 100, "Limit")
	logsQueryCmd.Flags().Int("offset", 0, "Offset")
	logsQueryCmd.Flags().Bool("only-message", false, "Show only log messages")
}
