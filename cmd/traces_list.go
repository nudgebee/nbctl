package cmd

import (
	"context"

	"github.com/machinebox/graphql"
	"github.com/spf13/cobra"
	"nudgebee.com/nbctl/pkg/client"
	"nudgebee.com/nbctl/pkg/format"
)

var tracesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List traces",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := client.NewClient()

		req := graphql.NewRequest(`
			query {
				traces {
					trace_id
					service_name
					operation_name
					start_time
					duration
					status_code
				}
			}
		`)

		var respData struct {
			Traces []struct {
				TraceID       string `json:"trace_id"`
				ServiceName   string `json:"service_name"`
				OperationName string `json:"operation_name"`
				StartTime     string `json:"start_time"`
				Duration      int    `json:"duration"`
				StatusCode    string `json:"status_code"`
			} `json:"traces"`
		}

		if err := client.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		table := format.TabularData{
			Data: respData.Traces,
			Fields: []format.TableField{
				{Header: "Trace ID", Field: "TraceID"},
				{Header: "Service Name", Field: "ServiceName"},
				{Header: "Operation Name", Field: "OperationName"},
				{Header: "Start Time", Field: "StartTime"},
				{Header: "Duration", Field: "Duration"},
				{Header: "Status Code", Field: "StatusCode"},
			},
		}
		format.GetFormat().Print(table)

		return nil
	},
}

func init() {
	tracesCmd.AddCommand(tracesListCmd)
}
