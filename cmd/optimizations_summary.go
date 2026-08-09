package cmd

import (
	"fmt"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var optimizationsSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Show summary of optimization recommendations grouped by category and status",
	RunE: func(cmd *cobra.Command, args []string) error {
		graphqlClient := client.NewClient()

		req := client.NewRequest(`
			query RecommendationGroupings {
				recommendation_groupings_v2 {
					rows {
						category
						status
						count
						sum_estimated_savings
					}
				}
			}
		`)

		var respData struct {
			RecommendationGroupingsV2 struct {
				Rows []struct {
					Category            string  `json:"category"`
					Status              string  `json:"status"`
					Count               int     `json:"count"`
					SumEstimatedSavings float64 `json:"sum_estimated_savings"`
				} `json:"rows"`
			} `json:"recommendation_groupings_v2"`
		}

		if err := graphqlClient.Run(cmd.Context(), req, &respData); err != nil {
			// Fallback to recommendations_list grouping if v2 grouping table is unavailable
			reqFallback := client.NewRequest(`
				query ListRecommendations {
					recommendations: recommendations_list(limit: 100) {
						rows {
							category
							status
							estimated_monthly_savings
						}
					}
				}
			`)
			var fallbackRespData struct {
				Recommendations struct {
					Rows []struct {
						Category                string  `json:"category"`
						Status                  string  `json:"status"`
						EstimatedMonthlySavings float64 `json:"estimated_monthly_savings"`
					} `json:"rows"`
				} `json:"recommendations"`
			}
			if errFallback := graphqlClient.Run(cmd.Context(), reqFallback, &fallbackRespData); errFallback != nil {
				return fmt.Errorf("primary query failed (%v), fallback query failed: %w", err, errFallback)
			}

			type key struct {
				Category string
				Status   string
			}
			counts := make(map[key]int)
			savings := make(map[key]float64)

			for _, r := range fallbackRespData.Recommendations.Rows {
				k := key{Category: r.Category, Status: r.Status}
				counts[k]++
				savings[k] += r.EstimatedMonthlySavings
			}

			type row struct {
				Category     string  `json:"category"`
				Status       string  `json:"status"`
				TotalCount   int     `json:"total_count"`
				TotalSavings float64 `json:"total_savings"`
			}
			var rows []row
			for k, c := range counts {
				rows = append(rows, row{
					Category:     k.Category,
					Status:       k.Status,
					TotalCount:   c,
					TotalSavings: savings[k],
				})
			}

			table := format.TabularData{
				Data: rows,
				Fields: []format.TableField{
					{Header: "Category", Field: "Category"},
					{Header: "Status", Field: "Status"},
					{Header: "Count", Field: "TotalCount"},
					{Header: "Estimated Savings ($/mo)", Field: "TotalSavings"},
				},
			}
			format.GetFormat().Print(table)
			return nil
		}

		table := format.TabularData{
			Data: respData.RecommendationGroupingsV2.Rows,
			Fields: []format.TableField{
				{Header: "Category", Field: "Category"},
				{Header: "Status", Field: "Status"},
				{Header: "Count", Field: "Count"},
				{Header: "Estimated Savings ($/mo)", Field: "SumEstimatedSavings"},
			},
		}
		format.GetFormat().Print(table)

		return nil
	},
}

func init() {
	optimizationsCmd.AddCommand(optimizationsSummaryCmd)
}
