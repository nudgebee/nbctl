package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

// LogsQueryQuery fetches log lines through the logs_list action.
const LogsQueryQuery = `query FetchLogs($request: FetchLogRequest!) {
  logs_list(request: $request) {
    logs {
      timestamp
      severity
      message
      labels
    }
    suggestion
  }
}`

var logsQueryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query logs",
	RunE: func(cmd *cobra.Command, args []string) error {
		graphqlClient := client.NewClient()

		accountId, err := resolveAccountID(cmd)
		if err != nil {
			return err
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

		index, _ := cmd.Flags().GetString("index")
		queryType, _ := cmd.Flags().GetString("query-type")
		if err := validateQueryType(queryType); err != nil {
			return err
		}
		params, err := providerParams(cmd, map[string]string{"index": index, "query_type": queryType})
		if err != nil {
			return err
		}

		req := client.NewRequest(LogsQueryQuery)

		requestVars := map[string]any{
			"account_id": accountId,
			"end_time":   endTime.UnixMilli(),
			"start_time": startTime.UnixMilli(),
			"query":      queryStr,
			"limit":      limit,
			"offset":     offset,
		}
		if params != nil {
			requestVars["request"] = params
		}
		req.Var("request", requestVars)

		var respData struct {
			LogsList struct {
				Logs       json.RawMessage `json:"logs"`
				Suggestion string          `json:"suggestion"`
			} `json:"logs_list"`
		}

		if err := graphqlClient.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		if respData.LogsList.Suggestion != "" {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Suggestion: %s\n", respData.LogsList.Suggestion)
		}

		raw := respData.LogsList.Logs
		if len(raw) == 0 || string(raw) == "null" {
			raw = json.RawMessage("[]")
		}

		var logs []struct {
			Timestamp string          `json:"timestamp"`
			Severity  string          `json:"severity"`
			Message   string          `json:"message"`
			Labels    json.RawMessage `json:"labels"`
		}
		// JSON output passes entries through, so for JSON only count them.
		jsonOutput := format.GetFormat().Get() == "json"
		count := -1 // unknown until decoded
		if jsonOutput {
			var entries []json.RawMessage
			if json.Unmarshal(raw, &entries) == nil {
				count = len(entries)
			}
		} else {
			if err := json.Unmarshal(raw, &logs); err != nil {
				return fmt.Errorf("failed to decode logs: %w", err)
			}
			count = len(logs)
		}

		window := fmt.Sprintf("between %s and %s", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
		switch {
		case count == 0:
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "No logs found %s.\n", window)
		case limit > 0 && count >= limit:
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Returned %d lines = --limit; results are probably cut off. Narrow --start-time/--end-time or the query, or page with --offset %d.\n", count, offset+count)
		}

		// JSON output is the backend's log entries, unchanged.
		if jsonOutput {
			return format.GetFormat().PrintRawJSON(raw)
		}
		if count <= 0 {
			return nil
		}

		table := format.TabularData{
			Data: logs,
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
	logsQueryCmd.Flags().String("index", "", "Index to search (Elasticsearch/OpenSearch; required for in-cluster Elasticsearch)")
	logsQueryCmd.Flags().String("query-type", "", "Query language for Elasticsearch: dsl (default), kql (hosted only) or ppl (OpenSearch)")
	addParamFlag(logsQueryCmd)
}
