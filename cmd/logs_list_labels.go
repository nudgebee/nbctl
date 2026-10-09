package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

// LogsListLabelsQuery lists log labels in a time window ($query is "start=<ns>&end=<ns>").
const LogsListLabelsQuery = `query FetchLogLabels($accountId: String!, $query: String!) {
  logs_list_labels(request: {account_id: $accountId, request: {query: $query}}) {
    label
  }
}`

// LogsListLabelsWithIndexQuery is LogsListLabelsQuery with an index, for
// providers that read labels from an index mapping (Elasticsearch).
const LogsListLabelsWithIndexQuery = `query FetchLogLabels($accountId: String!, $query: String!, $index: String!) {
  logs_list_labels(request: {account_id: $accountId, request: {index: $index, query: $query}}) {
    label
  }
}`

var logsListLabelsCmd = &cobra.Command{
	Use:   "list-labels",
	Short: "List log labels",
	RunE: func(cmd *cobra.Command, args []string) error {
		graphqlClient := client.NewClient()

		accountId, err := resolveAccountID(cmd)
		if err != nil {
			return err
		}

		startTimeStr, _ := cmd.Flags().GetString("start-time")
		endTimeStr, _ := cmd.Flags().GetString("end-time")

		var startTime, endTime time.Time

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

		req := client.NewRequest(LogsListLabelsQuery)
		if index, _ := cmd.Flags().GetString("index"); index != "" {
			req = client.NewRequest(LogsListLabelsWithIndexQuery)
			req.Var("index", index)
		}

		req.Var("accountId", accountId)
		req.Var("query", query)

		var respData struct {
			LogsListLabels []struct {
				Label string `json:"label"`
			} `json:"logs_list_labels"`
		}

		if err := graphqlClient.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		table := format.TabularData{
			Data: respData.LogsListLabels,
			Fields: []format.TableField{
				{Header: "Label", Field: "Label"},
			},
		}
		return printRows(cmd, table, len(respData.LogsListLabels), fmt.Sprintf("No log labels found between %s and %s.", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339)))
	},
}

func init() {
	logsCmd.AddCommand(logsListLabelsCmd)
	logsListLabelsCmd.Flags().String("account-id", "", "Account ID")
	logsListLabelsCmd.Flags().String("start-time", "", "Start time (RFC3339)")
	logsListLabelsCmd.Flags().String("end-time", "", "End time (RFC3339)")
	logsListLabelsCmd.Flags().String("index", "", "Elasticsearch/OpenSearch only: index whose fields to list (other providers ignore it)")
}
