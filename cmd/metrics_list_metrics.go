package cmd

import (
	"context"
	"fmt"

	"github.com/machinebox/graphql"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"nudgebee.com/nbctl/pkg/client"
	"nudgebee.com/nbctl/pkg/format"
)

var metricsListMetricsCmd = &cobra.Command{
	Use:   "list-metrics",
	Short: "List metrics",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := client.NewClient()

		accountId, _ := cmd.Flags().GetString("account-id")
		if accountId == "" {
			accountId = viper.GetString("account-id")
		}
		if accountId == "" {
			return fmt.Errorf("account-id is required")
		}

		req := graphql.NewRequest(`
			mutation MetricsList($accountId: String!) {
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

		if err := client.Run(context.Background(), req, &respData); err != nil {
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
