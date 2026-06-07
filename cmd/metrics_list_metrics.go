package cmd

import (
	"context"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var metricsListMetricsCmd = &cobra.Command{
	Use:   "list-metrics",
	Short: "List metrics",
	RunE: func(cmd *cobra.Command, args []string) error {
		graphqlClient := client.NewClient()

		accountId, err := resolveAccountID(cmd)
		if err != nil {
			return err
		}

		req := client.NewRequest(`
			query MetricsList($accountId: String!) {
			  metrics_list(request: {account_id: $accountId}) {
				metric
			  }
			}
		`)

		req.Var("accountId", accountId)

		var respData struct {
			MetricsList []struct {
				Metric string `json:"metric"`
			} `json:"metrics_list"`
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
		format.GetFormat().Print(table)

		return nil
	},
}

func init() {
	metricsCmd.AddCommand(metricsListMetricsCmd)
	metricsListMetricsCmd.Flags().String("account-id", "", "Account ID")
}
