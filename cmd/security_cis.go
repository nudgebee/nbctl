package cmd

import (
	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var securityCisCmd = &cobra.Command{
	Use:   "cis",
	Short: "Show summary of CIS benchmark security findings grouped by severity",
	RunE: func(cmd *cobra.Command, args []string) error {
		graphqlClient := client.NewClient()

		req := client.NewRequest(`
			query RecommendationSecurityCisGroupings {
				recommendation_security_cis_groupings_v2 {
					rows {
						rule_name
						severity
						status
						count
					}
				}
			}
		`)

		var respData struct {
			RecommendationSecurityCisGroupingsV2 struct {
				Rows []struct {
					RuleName string `json:"rule_name"`
					Severity string `json:"severity"`
					Status   string `json:"status"`
					Count    int    `json:"count"`
				} `json:"rows"`
			} `json:"recommendation_security_cis_groupings_v2"`
		}

		if err := graphqlClient.Run(cmd.Context(), req, &respData); err != nil {
			return err
		}

		table := format.TabularData{
			Data: respData.RecommendationSecurityCisGroupingsV2.Rows,
			Fields: []format.TableField{
				{Header: "Rule Name", Field: "RuleName"},
				{Header: "Severity", Field: "Severity"},
				{Header: "Status", Field: "Status"},
				{Header: "Count", Field: "Count"},
			},
		}
		format.GetFormat().Print(table)

		return nil
	},
}

func init() {
	securityCmd.AddCommand(securityCisCmd)
}
