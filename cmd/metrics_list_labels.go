package cmd

import (
	"context"
	"fmt"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

// MetricsListLabelsQuery lists the labels of a metric.
const MetricsListLabelsQuery = `query MetricsLabelList($accountId: String!, $metricName: String!) {
  metrics_list_labels(request: {account_id: $accountId, metric: $metricName}) {
    label
  }
}`

var metricsListLabelsCmd = &cobra.Command{
	Use:   "list-labels",
	Short: "List metric labels",
	RunE: func(cmd *cobra.Command, args []string) error {
		graphqlClient := client.NewClient()

		accountId, err := resolveAccountID(cmd)
		if err != nil {
			return err
		}

		metric, _ := cmd.Flags().GetString("metric")

		req := client.NewRequest(MetricsListLabelsQuery)

		req.Var("accountId", accountId)
		req.Var("metricName", metric)

		var respData struct {
			MetricsListLabels []struct {
				Label string `json:"label"`
			} `json:"metrics_list_labels"`
		}

		if err := graphqlClient.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		table := format.TabularData{
			Data: respData.MetricsListLabels,
			Fields: []format.TableField{
				{Header: "Label", Field: "Label"},
			},
		}
		return printRows(cmd, table, len(respData.MetricsListLabels), fmt.Sprintf("No labels found for metric %q.", metric))
	},
}

func init() {
	metricsCmd.AddCommand(metricsListLabelsCmd)
	metricsListLabelsCmd.Flags().String("account-id", "", "Account ID")
	metricsListLabelsCmd.Flags().String("metric", "", "Metric name")
	if err := metricsListLabelsCmd.MarkFlagRequired("metric"); err != nil {
		panic(err)
	}
}
