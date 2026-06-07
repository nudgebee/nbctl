package cmd

import (
	"context"
	"time"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var tracesQueryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query traces",
	RunE: func(cmd *cobra.Command, args []string) error {
		graphqlClient := client.NewClient()

		accountId, err := resolveAccountID(cmd)
		if err != nil {
			return err
		}

		workloadName, _ := cmd.Flags().GetStringSlice("workload-name")
		spanName, _ := cmd.Flags().GetStringSlice("span-name")
		traceId, _ := cmd.Flags().GetStringSlice("trace-id")
		startTime, _ := cmd.Flags().GetString("start-time")
		endTime, _ := cmd.Flags().GetString("end-time")
		resource, _ := cmd.Flags().GetString("resource")
		statusCode, _ := cmd.Flags().GetString("status-code")
		httpStatusCode, _ := cmd.Flags().GetStringSlice("http-status-code")

		st := time.Now().Add(-24 * time.Hour)
		if startTime != "" {
			parsed, err := time.Parse(time.RFC3339, startTime)
			if err != nil {
				return err
			}
			st = parsed
		}

		et := time.Now()
		if endTime != "" {
			parsed, err := time.Parse(time.RFC3339, endTime)
			if err != nil {
				return err
			}
			et = parsed
		}

		binary := map[string]any{
			"timestamp": map[string]any{
				"_between": map[string]any{
					"_gte": st.Format(time.RFC3339),
					"_lte": et.Format(time.RFC3339),
				},
			},
		}
		switch len(workloadName) {
		case 0:
		case 1:
			binary["workload_name"] = map[string]any{"_ilike": "%" + workloadName[0] + "%"}
		default:
			binary["workload_name"] = map[string]any{"_in": workloadName}
		}
		if len(spanName) > 0 {
			binary["span_name"] = map[string]any{"_in": spanName}
		}
		if len(traceId) > 0 {
			binary["trace_id"] = map[string]any{"_in": traceId}
		}
		if resource != "" {
			binary["resource"] = map[string]any{"_like": "%" + resource + "%"}
		}
		if statusCode != "" {
			binary["status_code"] = map[string]any{"_eq": statusCode}
		}
		if len(httpStatusCode) > 0 {
			binary["http_status_code"] = map[string]any{"_in": httpStatusCode}
		}

		req := client.NewRequest(`
			query TraceV3($request: TracesV3Input!) {
				traces_list(request: $request) {
					trace_id
					span_id
					parent_span_id
					workload_namespace
					workload_name
					timestamp
					status_code
					span_name
					resource
					duration_ns
					destination_workload_name
					destination_workload_namespace
					destination_name
					headers
					http_status_code
					request_payload
					http_response
					trace_source
					span_attributes
				}
			}
		`)

		req.Var("request", map[string]any{
			"account_id": accountId,
			"query":      "",
			"start_time": 0,
			"end_time":   0,
			"query_request": map[string]any{
				"where":    map[string]any{"_binary": binary},
				"having":   map[string]any{},
				"limit":    50,
				"offset":   0,
				"order_by": []map[string]any{{"column": "timestamp", "order": "desc"}},
			},
		})

		var respData struct {
			TracesList []struct {
				TraceID                      string         `json:"trace_id"`
				SpanID                       string         `json:"span_id"`
				ParentSpanID                 string         `json:"parent_span_id"`
				WorkloadNamespace            string         `json:"workload_namespace"`
				WorkloadName                 string         `json:"workload_name"`
				Timestamp                    string         `json:"timestamp"`
				StatusCode                   string         `json:"status_code"`
				SpanName                     string         `json:"span_name"`
				Resource                     string         `json:"resource"`
				DurationNs                   int            `json:"duration_ns"`
				DestinationWorkloadName      string         `json:"destination_workload_name"`
				DestinationWorkloadNamespace string         `json:"destination_workload_namespace"`
				DestinationName              string         `json:"destination_name"`
				Headers                      any            `json:"headers"`
				HTTPStatusCode               string         `json:"http_status_code"`
				RequestPayload               any            `json:"request_payload"`
				HTTPResponse                 any            `json:"http_response"`
				TraceSource                  string         `json:"trace_source"`
				SpanAttributes               map[string]any `json:"span_attributes"`
			} `json:"traces_list"`
		}

		if err := graphqlClient.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		table := format.TabularData{
			Data: respData.TracesList,
			Fields: []format.TableField{
				{Header: "Trace ID", Field: "TraceID"},
				{Header: "Span ID", Field: "SpanID"},
				{Header: "Workload Name", Field: "WorkloadName"},
				{Header: "Timestamp", Field: "Timestamp"},
				{Header: "Status Code", Field: "StatusCode"},
				{Header: "Span Name", Field: "SpanName"},
				{Header: "Duration (ns)", Field: "DurationNs"},
			},
		}
		format.GetFormat().Print(table)

		return nil
	},
}

func init() {
	tracesCmd.AddCommand(tracesQueryCmd)
	tracesQueryCmd.Flags().String("account-id", "", "Account ID (overrides profile)")
	tracesQueryCmd.Flags().StringSlice("workload-name", []string{}, "Filter by workload name")
	tracesQueryCmd.Flags().StringSlice("span-name", []string{}, "Filter by span name")
	tracesQueryCmd.Flags().StringSlice("trace-id", []string{}, "Filter by trace id")
	tracesQueryCmd.Flags().String("start-time", "", "Start time in RFC3339 format")
	tracesQueryCmd.Flags().String("end-time", "", "End time in RFC3339 format")
	tracesQueryCmd.Flags().String("resource", "", "Filter by resource")
	tracesQueryCmd.Flags().String("status-code", "", "Filter by status code")
	tracesQueryCmd.Flags().StringSlice("http-status-code", []string{}, "Filter by http status code")
}
