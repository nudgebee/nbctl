package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/machinebox/graphql"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"nudgebee.com/nbctl/pkg/client"
	"nudgebee.com/nbctl/pkg/format"
)

var logsListLabelsCmd = &cobra.Command{
	Use:   "list-labels",
	Short: "List log labels",
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
			mutation FetchLogLabels($accountId: String!, $query: String!) {
			  logs_list_labels(request: {account_id: $accountId, request: {query: $query}}) {
				label
			  }
			}
		`)

		req.Var("accountId", accountId)
		req.Var("query", query)

		var respData struct {
			LogsListLabels []struct {
				Label string `json:"label"`
			} `json:"logs_list_labels"`
		}

		if err := client.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		table := format.TabularData{
			Data: respData.LogsListLabels,
			Fields: []format.TableField{
				{Header: "Label", Field: "Label"},
			},
		}
		format.GetFormat().Print(table)

		return nil
	},
}

func init() {
	logsCmd.AddCommand(logsListLabelsCmd)
	logsListLabelsCmd.Flags().String("account-id", "", "Account ID")
	logsListLabelsCmd.Flags().String("start-time", "", "Start time (RFC3339)")
	logsListLabelsCmd.Flags().String("end-time", "", "End time (RFC3339)")
}
