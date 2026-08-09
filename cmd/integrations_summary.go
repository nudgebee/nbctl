package cmd

import (
	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var integrationsSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Show summary of integrations grouped by type and status",
	RunE: func(cmd *cobra.Command, args []string) error {
		graphqlClient := client.NewClient()

		req := client.NewRequest(`
			query AggregateIntegrations {
				integrations_aggregate {
					rows {
						type
						status
						count
					}
				}
			}
		`)

		var respData struct {
			IntegrationsAggregate struct {
				Rows []struct {
					Type   string `json:"type"`
					Status string `json:"status"`
					Count  int    `json:"count"`
				} `json:"rows"`
			} `json:"integrations_aggregate"`
		}

		if err := graphqlClient.Run(cmd.Context(), req, &respData); err != nil {
			return err
		}

		table := format.TabularData{
			Data: respData.IntegrationsAggregate.Rows,
			Fields: []format.TableField{
				{Header: "Type", Field: "Type"},
				{Header: "Status", Field: "Status"},
				{Header: "Count", Field: "Count"},
			},
		}
		format.GetFormat().Print(table)

		return nil
	},
}

func init() {
	integrationsCmd.AddCommand(integrationsSummaryCmd)
}
