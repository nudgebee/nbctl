package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/machinebox/graphql"
	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	workloadName   []string
	spanName       []string
	traceId        []string
	startTime      string
	endTime        string
	resource       string
	statusCode     string
	httpStatusCode []string
)

var tracesQueryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query traces",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := client.NewClient()

		accountId := viper.GetString("account-id")

		// Build the where clause
		var whereClause []string
		if len(workloadName) == 1 {
			whereClause = append(whereClause, fmt.Sprintf(`workload_name:{_ilike:"%%%s%%"}`, workloadName[0]))
		} else if len(workloadName) > 1 {
			whereClause = append(whereClause, fmt.Sprintf(`workload_name:{_in:["%s"]}`, strings.Join(workloadName, `","`)))
		}
		if len(spanName) > 0 {
			whereClause = append(whereClause, fmt.Sprintf(`span_name:{_in:["%s"]}`, strings.Join(spanName, `","`)))
		}
		if len(traceId) > 0 {
			whereClause = append(whereClause, fmt.Sprintf(`trace_id:{_in:["%s"]}`, strings.Join(traceId, `","`)))
		}
		if resource != "" {
			whereClause = append(whereClause, fmt.Sprintf(`resource:{_like:"%%%s%%"}`, resource))
		}
		if statusCode != "" {
			whereClause = append(whereClause, fmt.Sprintf(`status_code:{_eq:"%s"}`, statusCode))
		}
		if len(httpStatusCode) > 0 {
			whereClause = append(whereClause, fmt.Sprintf(`http_status_code:{_in:["%s"]}`, strings.Join(httpStatusCode, `","`)))
		}

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

		whereClause = append(whereClause, fmt.Sprintf(`timestamp:{_between:{_gte:"%s",_lte:"%s"}}`, st.Format(time.RFC3339), et.Format(time.RFC3339)))

		req := graphql.NewRequest(fmt.Sprintf(`
			query TraceV3($account_id: String!) {
				traces_v3(request: {account_id: $account_id, query:"",start_time:0,end_time:0,query_request:{where:{_binary:{%s}},having:{},limit:50,offset:0,order_by:[{column:"timestamp",order:"desc"}]}}) {
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
		`, strings.Join(whereClause, ",")))

		req.Var("account_id", accountId)

		var respData struct {
			TracesV3 []struct {
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
			} `json:"traces_v3"`
		}

		if err := client.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		table := format.TabularData{
			Data: respData.TracesV3,
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
	tracesQueryCmd.Flags().StringSliceVar(&workloadName, "workload-name", []string{}, "Filter by workload name")
	tracesQueryCmd.Flags().StringSliceVar(&spanName, "span-name", []string{}, "Filter by span name")
	tracesQueryCmd.Flags().StringSliceVar(&traceId, "trace-id", []string{}, "Filter by trace id")
	tracesQueryCmd.Flags().StringVar(&startTime, "start-time", "", "Start time in RFC3339 format")
	tracesQueryCmd.Flags().StringVar(&endTime, "end-time", "", "End time in RFC3339 format")
	tracesQueryCmd.Flags().StringVar(&resource, "resource", "", "Filter by resource")
	tracesQueryCmd.Flags().StringVar(&statusCode, "status-code", "", "Filter by status code")
	tracesQueryCmd.Flags().StringSliceVar(&httpStatusCode, "http-status-code", []string{}, "Filter by http status code")
}
