package cmd

import (
	"context"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var metricsListLabelValuesCmd = &cobra.Command{
	Use:   "list-label-values",
	Short: "List metric label values",
	RunE: func(cmd *cobra.Command, args []string) error {
		graphqlClient := client.NewClient()

		accountId, err := resolveAccountID(cmd)
		if err != nil {
			return err
		}

		label, _ := cmd.Flags().GetString("label")

		req := client.NewRequest(`
			query MetricsLabelValueList($accountId: String!, $labelName: String!) {
			  metrics_list_label_values(request: {account_id: $accountId, label: $labelName}) {
				value
			  }
			}
		`)

		req.Var("accountId", accountId)
		req.Var("labelName", label)

		var respData struct {
			MetricsListLabelValues []struct {
				Value string `json:"value"`
			} `json:"metrics_list_label_values"`
		}

		if err := graphqlClient.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		table := format.TabularData{
			Data: respData.MetricsListLabelValues,
			Fields: []format.TableField{
				{Header: "Value", Field: "Value"},
			},
		}
		format.GetFormat().Print(table)

		return nil
	},
}

func init() {
	metricsCmd.AddCommand(metricsListLabelValuesCmd)
	metricsListLabelValuesCmd.Flags().String("account-id", "", "Account ID")
	metricsListLabelValuesCmd.Flags().String("label", "", "Label name")
	if err := metricsListLabelValuesCmd.MarkFlagRequired("label"); err != nil {
		panic(err)
	}
}
