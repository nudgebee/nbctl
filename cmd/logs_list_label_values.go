package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/machinebox/graphql"
	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var logsListLabelValuesCmd = &cobra.Command{
	Use:   "list-label-values",
	Short: "List log label values",
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
		labelName, _ := cmd.Flags().GetString("label-name")

		var startTime, endTime time.Time
		var err error

		if startTimeStr == "" {
			startTime = time.Now().Add(-1 * time.Hour)
		} else {
			startTime, err = time.Parse(time.RFC3339, startTimeStr)
			if err != nil {
				return err
			}
		}

		if endTimeStr == "" {
			endTime = time.Now()
		} else {
			endTime, err = time.Parse(time.RFC3339, endTimeStr)
			if err != nil {
				return err
			}
		}

		query := fmt.Sprintf("start=%d&end=%d", startTime.UnixNano(), endTime.UnixNano())

		req := graphql.NewRequest(`
			mutation FetchLogLabelValues($accountId: String!, $labelName: String!, $query: String!) {
			  logs_list_label_values(request: {account_id: $accountId, label_name: $labelName, request: {query: $query}}) {
				value
			  }
			}
		`)

		req.Var("accountId", accountId)
		req.Var("labelName", labelName)
		req.Var("query", query)

		var respData struct {
			LogsListLabelValues []struct {
				Value string `json:"value"`
			} `json:"logs_list_label_values"`
		}

		if err := client.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		table := format.TabularData{
			Data: respData.LogsListLabelValues,
			Fields: []format.TableField{
				{Header: "Value", Field: "Value"},
			},
		}
		format.GetFormat().Print(table)

		return nil
	},
}

func init() {
	logsCmd.AddCommand(logsListLabelValuesCmd)
	logsListLabelValuesCmd.Flags().String("account-id", "", "Account ID")
	logsListLabelValuesCmd.Flags().String("start-time", "", "Start time (RFC3339)")
	logsListLabelValuesCmd.Flags().String("end-time", "", "End time (RFC3339)")
	logsListLabelValuesCmd.Flags().String("label-name", "", "Label name")
	if err := logsListLabelValuesCmd.MarkFlagRequired("label-name"); err != nil {
		panic(err)
	}
}
