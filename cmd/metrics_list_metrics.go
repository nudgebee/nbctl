package cmd

import (
	"context"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

// MetricsListNamesQuery lists metric names. metrics_list is the series query
// action (see metrics query), so names come from metrics_list_names.
const MetricsListNamesQuery = `query MetricsListNames($accountId: String!) {
  metrics_list_names(request: {account_id: $accountId}) {
    metric
  }
}`

var metricsListMetricsCmd = &cobra.Command{
	Use:   "list-metrics",
	Short: "List metrics",
	RunE: func(cmd *cobra.Command, args []string) error {
		graphqlClient := client.NewClient()

		accountId, err := resolveAccountID(cmd)
		if err != nil {
			return err
		}

		req := client.NewRequest(MetricsListNamesQuery)
		req.Var("accountId", accountId)

		var respData struct {
			MetricsList []struct {
				Metric string `json:"metric"`
			} `json:"metrics_list_names"`
		}

		if err := graphqlClient.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		table := format.TabularData{
			Data: respData.MetricsList,
			Fields: []format.TableField{
				{Header: "Metric", Field: "Metric"},
			},
		}
		return printRows(cmd, table, len(respData.MetricsList), "No metrics found for this account.")
	},
}

func init() {
	metricsCmd.AddCommand(metricsListMetricsCmd)
	metricsListMetricsCmd.Flags().String("account-id", "", "Account ID")
}
