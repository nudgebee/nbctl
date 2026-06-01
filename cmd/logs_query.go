package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var logsQueryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query logs",
	RunE: func(cmd *cobra.Command, args []string) error {
		graphqlClient := client.NewClient()

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

		req := client.NewRequest(`
			query FetchLogs($request: FetchLogRequest!) {
				logs_list(request: $request) {
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
			"query":      queryStr,
			"limit":      limit,
			"offset":     offset,
		}
		req.Var("request", requestVars)

		var respData struct {
			LogsQuery []struct {
				Timestamp string          `json:"timestamp"`
				Severity  string          `json:"severity"`
				Message   string          `json:"message"`
				Labels    json.RawMessage `json:"labels"`
			} `json:"logs_list"`
		}

		if err := graphqlClient.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		if len(respData.LogsQuery) == 0 {
			fmt.Println("No logs found.")
			return nil
		}

		table := format.TabularData{
			Data: respData.LogsQuery,
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
