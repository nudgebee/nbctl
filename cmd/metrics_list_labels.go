package cmd

import (
	"context"
	"fmt"

	"github.com/machinebox/graphql"
	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var metricsListLabelsCmd = &cobra.Command{
	Use:   "list-labels",
	Short: "List metric labels",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := client.NewClient()

		accountId, _ := cmd.Flags().GetString("account-id")
		if accountId == "" {
			accountId = viper.GetString("account-id")
		}
		if accountId == "" {
			return fmt.Errorf("account-id is required")
		}

		metric, _ := cmd.Flags().GetString("metric")

		req := graphql.NewRequest(`
			query MetricsLabelList($accountId: String!, $metricName: String!) {
			  metrics_list_labels(request: {account_id: $accountId, metric: $metricName}) {
				label
			  }
			}
		`)

		req.Var("accountId", accountId)
		req.Var("metricName", metric)

		var respData struct {
			MetricsListLabels []struct {
				Label string `json:"label"`
			} `json:"metrics_list_labels"`
		}

		if err := client.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		table := format.TabularData{
			Data: respData.MetricsListLabels,
			Fields: []format.TableField{
				{Header: "Label", Field: "Label"},
			},
		}
		format.GetFormat().Print(table)

		return nil
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
