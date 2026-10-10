package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

// LogsQueryQuery fetches log lines through the logs_list action, with the
// result's counts and truncation. truncated, total, total_relation, series,
// series_note and
// the *_raw fragments are absent on an older api-server, and the *_raw fields
// are only filled when the request sets include_raw (-o raw).
const LogsQueryQuery = `query FetchLogs($request: FetchLogRequest!) {
  logs_list(request: $request) {
    logs {
      timestamp
      severity
      message
      labels
    }
    query
    provider
    suggestion
    truncated
    total
    total_relation
    series
    series_note
    partial
    aggregations_raw
    total_raw
  }
}`

// defaultLogsLimit matches llm-server's logs default, so nbctl and Nubi's
// tools page the same way.
const defaultLogsLimit = 1000

var logsQueryCmd = &cobra.Command{
	Use:         "query",
	Short:       "Query logs",
	Annotations: map[string]string{rawOutputAnnotation: "true"},
	Long: `Query logs in the account's log provider.

--query is written in the provider's own language: LogQL for Loki,
Elasticsearch Query DSL JSON, and so on.

--index applies to Elasticsearch/OpenSearch only; other providers ignore it.

Output:
  text      log lines as a table, then count series if the query returned any
  -o json   {"logs": [...], "series": [...], "truncated": bool, "partial": bool,
             "total": N, "total_relation": "eq"|"gte", "series_note": "...",
             "suggestion": "..."}; keys the api-server did not send are left
            out. Check truncated and partial before trusting the result.
  -o raw    the whole result, including the executed query, the provider and
            the provider's own fragments (aggregations_raw, total_raw), unchanged

stderr also says when the result may be cut off, how many lines matched when the
provider reports it (e.g. a counts-only query), and why series may be
incomplete. Elasticsearch counts past 10,000 only with "track_total_hits": true.

Counts as series (Elasticsearch): zero or more nested terms aggregations,
optionally ending in one date_histogram. Use real field names from
"nbctl logs list-labels --fields-only"; terms needs a keyword field.`,
	Example: `  # Loki
  nbctl logs query --query '{namespace="api"} |= "error"' --start-time 2026-10-01T00:00:00Z

  # Elasticsearch, Query DSL (in-cluster Elasticsearch needs --index)
  nbctl logs query --index 'logs-*' --query '{"query":{"match":{"level":"error"}}}' -o json > logs.json

  # Elasticsearch, error lines per pod per hour (counts come back as series)
  nbctl logs query -o json > counts.json --query '{"size":0,"track_total_hits":true,
    "query":{"match":{"log":"error"}},
    "aggs":{"pod":{"terms":{"field":"kubernetes.pod_name","size":500},
    "aggs":{"hour":{"date_histogram":{"field":"@timestamp","fixed_interval":"1h"}}}}}}'`,
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
		// Always send a positive limit: with none (or 0) the api-server falls back
		// to each provider's own default (10 to 5,000 lines), and a cut-off
		// result could not be told from a complete one.
		if limit < 1 {
			return fmt.Errorf("invalid --limit %d: must be at least 1", limit)
		}
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
		params, err := providerParams(cmd, map[string]string{"index": index})
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
		rawOutput := format.GetFormat().Get() == "raw"
		if rawOutput {
			requestVars["include_raw"] = true
		}
		req.Var("request", requestVars)

		var respData struct {
			LogsList json.RawMessage `json:"logs_list"`
		}
		if err := graphqlClient.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		var result struct {
			Logs          json.RawMessage `json:"logs"`
			Suggestion    string          `json:"suggestion"`
			Truncated     *bool           `json:"truncated"`
			Total         *int64          `json:"total"`
			TotalRelation string          `json:"total_relation"`
			Series        json.RawMessage `json:"series"`
			SeriesNote    string          `json:"series_note"`
			Provider      string          `json:"provider"`
			Partial       bool            `json:"partial"`
		}
		if len(respData.LogsList) > 0 {
			if err := json.Unmarshal(respData.LogsList, &result); err != nil {
				return fmt.Errorf("failed to decode logs result: %w", err)
			}
		}

		if result.Suggestion != "" {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Suggestion: %s\n", result.Suggestion)
		}
		if result.Partial {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Warning: partial result: the provider did not finish (e.g. a time budget ran out), so lines or counts are missing. Narrow --start-time/--end-time or the query.")
		}
		// Why counts are partial or missing from series (e.g. a terms grouping
		// left groups out, or the counts are only in aggregations_raw).
		if result.SeriesNote != "" {
			note := result.SeriesNote
			if strings.Contains(note, "include_raw") && !rawOutput {
				note += " (in nbctl: -o raw)"
			}
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Note: %s\n", note)
		}

		raw := orEmptyArray(result.Logs)
		series := orEmptyArray(result.Series)

		var logs []struct {
			Timestamp string          `json:"timestamp"`
			Severity  string          `json:"severity"`
			Message   string          `json:"message"`
			Labels    json.RawMessage `json:"labels"`
		}
		// JSON and raw output pass entries through, so for those only count them.
		textOutput := format.GetFormat().Get() != "json" && !rawOutput
		count := -1 // unknown until decoded
		if textOutput {
			if err := json.Unmarshal(raw, &logs); err != nil {
				return fmt.Errorf("failed to decode logs: %w", err)
			}
			count = len(logs)
		} else {
			var entries []json.RawMessage
			if json.Unmarshal(raw, &entries) == nil {
				count = len(entries)
			}
		}
		var seriesEntries []json.RawMessage
		_ = json.Unmarshal(series, &seriesEntries)

		matched := result.Total != nil && *result.Total > 0
		switch {
		case count == 0 && len(seriesEntries) == 0 && !matched:
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "No logs found between %s and %s.\n", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
		case count <= 0 && result.Total != nil:
			// A counts-only query returns no lines; the match count lets a script
			// check its series sums.
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Matched %s lines.%s\n", describeTotal(*result.Total, result.TotalRelation), exactCountHint(result.Provider, result.TotalRelation))
		default:
			if msg := truncationWarning(result.Truncated, result.Total, result.TotalRelation, count, limit, offset); msg != "" {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), msg+exactCountHint(result.Provider, result.TotalRelation))
			}
		}

		switch {
		case rawOutput:
			// The whole result as returned, provider fragments included.
			return format.GetFormat().PrintRawJSON(orEmptyObject(respData.LogsList))
		case !textOutput:
			// One shape for every result: the entries and counts as returned,
			// plus whether the result is complete, so a script that drops stderr
			// still sees it.
			out := map[string]any{"logs": raw, "series": series, "partial": result.Partial}
			if result.Truncated != nil {
				out["truncated"] = *result.Truncated
			}
			if result.Total != nil {
				out["total"] = *result.Total
				out["total_relation"] = result.TotalRelation
			}
			if result.SeriesNote != "" {
				out["series_note"] = result.SeriesNote
			}
			if result.Suggestion != "" {
				out["suggestion"] = result.Suggestion
			}
			combined, err := json.Marshal(out)
			if err != nil {
				return err
			}
			return format.GetFormat().PrintRawJSON(combined)
		}

		if count > 0 {
			format.GetFormat().Print(format.TabularData{
				Data: logs,
				Fields: []format.TableField{
					{Header: "Timestamp", Field: "Timestamp"},
					{Header: "Severity", Field: "Severity"},
					{Header: "Message", Field: "Message"},
					{Header: "Labels", Field: "Labels"},
				},
			})
		}
		if len(seriesEntries) > 0 {
			if err := printSeriesTable(series); err != nil {
				return err
			}
		}
		return nil
	},
}

// truncationWarning explains on stderr that a result may be incomplete. It
// trusts the api-server's truncated/total when present; an older api-server
// sends neither, and then a page of exactly --limit lines is the only hint.
// truncated can over-report (exactly --limit matches), so without a total the
// wording is "may be cut off".
func truncationWarning(truncated *bool, total *int64, relation string, count, limit, offset int) string {
	if count <= 0 {
		return ""
	}
	cutOff := truncated != nil && *truncated
	if truncated == nil {
		// Exactly --limit, not more: a Loki metric query returns one entry per
		// series point, unbounded by --limit, and is not cut off.
		cutOff = limit > 0 && count == limit
	}
	if !cutOff {
		return ""
	}
	next := fmt.Sprintf("Narrow --start-time/--end-time or the query, or page with --offset %d.", offset+count)
	if total != nil {
		return fmt.Sprintf("Returned %d of %s matching lines. %s", count, describeTotal(*total, relation), next)
	}
	return fmt.Sprintf("Returned %d lines = --limit %d; results may be cut off. %s", count, limit, next)
}

// describeTotal renders a match count, "at least N" when it is a lower bound.
func describeTotal(total int64, relation string) string {
	if relation == "gte" {
		return fmt.Sprintf("at least %d", total)
	}
	return fmt.Sprintf("%d", total)
}

// exactCountHint tells how to get an exact count when Elasticsearch reported
// only a lower bound (it stops counting at 10,000 by default).
func exactCountHint(provider, relation string) string {
	if relation == "gte" && provider == "ES" {
		return ` For an exact count, add "track_total_hits": true to the query body.`
	}
	return ""
}

func orEmptyArray(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 || string(raw) == "null" {
		return json.RawMessage("[]")
	}
	return raw
}

func orEmptyObject(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 || string(raw) == "null" {
		return json.RawMessage("{}")
	}
	return raw
}

// printSeriesTable prints count series (metrics_list's Result shape: metric
// labels, timestamps, values) the way metrics query prints its series.
func printSeriesTable(series json.RawMessage) error {
	var results []MetricsResult
	if err := json.Unmarshal(series, &results); err != nil {
		return fmt.Errorf("failed to decode series: %w", err)
	}
	rows := make([]DisplayMetricsResult, 0, len(results))
	for _, r := range results {
		metric, _ := json.Marshal(r.Metric)
		timestamps, _ := json.Marshal(r.Timestamps)
		values, _ := json.Marshal(r.Values)
		rows = append(rows, DisplayMetricsResult{Metric: string(metric), Timestamps: string(timestamps), Values: string(values)})
	}
	format.GetFormat().Print(format.TabularData{
		Data: rows,
		Fields: []format.TableField{
			{Header: "Series", Field: "Metric"},
			{Header: "Timestamps", Field: "Timestamps"},
			{Header: "Counts", Field: "Values"},
		},
	})
	return nil
}

func init() {
	logsCmd.AddCommand(logsQueryCmd)
	logsQueryCmd.Flags().String("account-id", "", "Account ID")
	logsQueryCmd.Flags().String("start-time", "", "Start time (RFC3339)")
	logsQueryCmd.Flags().String("end-time", "", "End time (RFC3339)")
	logsQueryCmd.Flags().String("query", "", "Log query in the provider's language (LogQL for Loki; DSL, KQL or PPL for Elasticsearch)")
	logsQueryCmd.Flags().Int("limit", defaultLogsLimit, "Maximum log lines to return")
	logsQueryCmd.Flags().Int("offset", 0, "Offset")
	logsQueryCmd.Flags().Bool("only-message", false, "Show only log messages")
	logsQueryCmd.Flags().String("index", "", "Elasticsearch/OpenSearch only: index to search (required for in-cluster Elasticsearch; hosted defaults to the account's log index)")
	addParamFlag(logsQueryCmd)
}
