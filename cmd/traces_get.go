package cmd

import (
	"context"
	"fmt"

	"github.com/machinebox/graphql"
	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var tracesGetCmd = &cobra.Command{
	Use:   "get [trace_id]",
	Short: "Get a trace by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := client.NewClient()

		req := graphql.NewRequest(`
			query($traceId: String!) {
				traces_by_pk(trace_id: $traceId) {
					trace_id
					service_name
					operation_name
					start_time
					duration
					status_code
				}
			}
		`)

		req.Var("traceId", args[0])

		var respData struct {
			Trace struct {
				TraceID       string `json:"trace_id"`
				ServiceName   string `json:"service_name"`
				OperationName string `json:"operation_name"`
				StartTime     string `json:"start_time"`
				Duration      int    `json:"duration"`
				StatusCode    string `json:"status_code"`
			} `json:"traces_by_pk"`
		}

		if err := client.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		if respData.Trace.TraceID == "" {
			fmt.Println("Trace not found.")
			return nil
		}

		format.GetFormat().Print(respData.Trace)

		return nil
	},
}

func init() {
	tracesCmd.AddCommand(tracesGetCmd)
}
