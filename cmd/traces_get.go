package cmd

import (
	"context"
	"fmt"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var tracesGetCmd = &cobra.Command{
	Use:   "get [trace_id]",
	Short: "Get a trace by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		graphqlClient := client.NewClient()
		accountId, err := resolveAccountID(cmd)
		if err != nil {
			return err
		}

		req := client.NewRequest(`
			query TraceGet($account_id: String!, $trace_id: String!) {
				traces_list(request: {account_id: $account_id, query:"", start_time:0, end_time:0, query_request:{where:{_binary:{trace_id:{_eq:$trace_id}}}, having:{}, limit:1, offset:0, order_by:[{column:"timestamp", order:"desc"}]}}) {
					trace_id
					span_id
					parent_span_id
					workload_name
					workload_namespace
					timestamp
					status_code
					span_name
					resource
					duration_ns
					http_status_code
				}
			}
		`)

		req.Var("account_id", accountId)
		req.Var("trace_id", args[0])

		var respData struct {
			TracesList []struct {
				TraceID           string `json:"trace_id"`
				SpanID            string `json:"span_id"`
				ParentSpanID      string `json:"parent_span_id"`
				WorkloadName      string `json:"workload_name"`
				WorkloadNamespace string `json:"workload_namespace"`
				Timestamp         string `json:"timestamp"`
				StatusCode        string `json:"status_code"`
				SpanName          string `json:"span_name"`
				Resource          string `json:"resource"`
				DurationNs        int    `json:"duration_ns"`
				HTTPStatusCode    string `json:"http_status_code"`
			} `json:"traces_list"`
		}

		if err := graphqlClient.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		if len(respData.TracesList) == 0 {
			fmt.Println("Trace not found.")
			return nil
		}

		format.GetFormat().Print(respData.TracesList[0])

		return nil
	},
}

func init() {
	tracesCmd.AddCommand(tracesGetCmd)
	tracesGetCmd.Flags().String("account-id", "", "Account ID (overrides profile)")
}
