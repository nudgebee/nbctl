package cmd

import (
	"sort"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var accountsSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Show summary of cloud accounts grouped by cloud provider and status",
	RunE: func(cmd *cobra.Command, args []string) error {
		graphqlClient := client.NewClient()

		req := client.NewRequest(`
			query AggregateAccounts {
				accounts_list {
					rows {
						cloud_provider
						status
					}
				}
			}
		`)

		var respData struct {
			AccountsList struct {
				Rows []struct {
					CloudProvider string `json:"cloud_provider"`
					Status        string `json:"status"`
				} `json:"rows"`
			} `json:"accounts_list"`
		}

		if err := graphqlClient.Run(cmd.Context(), req, &respData); err != nil {
			return err
		}

		counts := make(map[string]map[string]int)
		for _, r := range respData.AccountsList.Rows {
			provider := r.CloudProvider
			if provider == "" {
				provider = "unknown"
			}
			status := r.Status
			if status == "" {
				status = "unknown"
			}
			if _, ok := counts[provider]; !ok {
				counts[provider] = make(map[string]int)
			}
			counts[provider][status]++
		}

		type summaryRow struct {
			CloudProvider string `json:"cloud_provider"`
			Status        string `json:"status"`
			Count         int    `json:"count"`
		}

		var summaryRows []summaryRow
		for provider, stMap := range counts {
			for status, count := range stMap {
				summaryRows = append(summaryRows, summaryRow{
					CloudProvider: provider,
					Status:        status,
					Count:         count,
				})
			}
		}

		sort.Slice(summaryRows, func(i, j int) bool {
			if summaryRows[i].CloudProvider != summaryRows[j].CloudProvider {
				return summaryRows[i].CloudProvider < summaryRows[j].CloudProvider
			}
			return summaryRows[i].Status < summaryRows[j].Status
		})

		table := format.TabularData{
			Data: summaryRows,
			Fields: []format.TableField{
				{Header: "Cloud Provider", Field: "CloudProvider"},
				{Header: "Status", Field: "Status"},
				{Header: "Count", Field: "Count"},
			},
		}
		format.GetFormat().Print(table)

		return nil
	},
}

func init() {
	accountsCmd.AddCommand(accountsSummaryCmd)
}
