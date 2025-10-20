package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/machinebox/graphql"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"nudgebee.com/nbctl/pkg/client"
	"nudgebee.com/nbctl/pkg/format"
)

var optimizationsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List optimizations",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := client.NewClient()

		accountId, _ := cmd.Flags().GetString("account-id")
		if accountId == "" {
			accountId = viper.GetString("account-id")
		}
		if accountId == "" {
			return fmt.Errorf("account-id is required")
		}

		category, _ := cmd.Flags().GetString("category")
		ruleName, _ := cmd.Flags().GetString("rule-name")
		status, _ := cmd.Flags().GetString("status")
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")

		where := map[string]any{
			"account_id": map[string]any{"_eq": accountId},
		}

		if category != "" {
			where["category"] = map[string]any{"_eq": category}
		}
		if ruleName != "" {
			where["rule_name"] = map[string]any{"_eq": ruleName}
		}
		if status != "" {
			where["status"] = map[string]any{"_in": []string{status}}
		}

		req := graphql.NewRequest(`
			query list_k8_recommendation($where: RecommendationWhereRequest!, $limit: Int, $offset: Int) {
			  recommendation: recommendations_v2(where: $where, limit: $limit, offset: $offset, order_by: {column: "estimated_savings", order: desc}) {
				rows{
				  resource_k8s_namespace
				  resource_meta
				  resource_name
				  resource_type
				  resource_id
				  severity
				  category
				  rule_name
				  recommendation
				  estimated_savings
				  account_object_id
				  updated_at
				  id
				}
			  }
			}
		`)

		req.Var("where", where)
		req.Var("limit", limit)
		req.Var("offset", offset)

		var respData struct {
			Recommendation struct {
				Rows []struct {
					ResourceK8sNamespace string          `json:"resource_k8s_namespace"`
					ResourceMeta         json.RawMessage `json:"resource_meta"`
					ResourceName         string          `json:"resource_name"`
					ResourceType         string          `json:"resource_type"`
					ResourceID           string          `json:"resource_id"`
					Severity             string          `json:"severity"`
					Category             string          `json:"category"`
					RuleName             string          `json:"rule_name"`
					Recommendation       string          `json:"recommendation"`
					EstimatedSavings     float64         `json:"estimated_savings"`
					AccountObjectID      string          `json:"account_object_id"`
					UpdatedAt            string          `json:"updated_at"`
					ID                   string          `json:"id"`
				} `json:"rows"`
			} `json:"recommendation"`
		}

		if err := client.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		table := format.TabularData{
			Data: respData.Recommendation.Rows,
			Fields: []format.TableField{
				{Header: "ID", Field: "ID"},
				{Header: "Category", Field: "Category"},
				{Header: "Rule Name", Field: "RuleName"},
				{Header: "Resource Type", Field: "ResourceType"},
				{Header: "Resource Name", Field: "ResourceName"},
				{Header: "Estimated Savings", Field: "EstimatedSavings"},
				{Header: "Severity", Field: "Severity"},
				{Header: "Updated At", Field: "UpdatedAt"},
			},
		}
		format.GetFormat().Print(table)

		return nil
	},
}

func init() {
	optimizationsCmd.AddCommand(optimizationsListCmd)
	optimizationsListCmd.Flags().String("account-id", "", "Account ID")
	optimizationsListCmd.Flags().String("category", "", "Filter by category")
	optimizationsListCmd.Flags().String("rule-name", "", "Filter by rule name")
	optimizationsListCmd.Flags().String("status", "", "Filter by status")
	optimizationsListCmd.Flags().Int("limit", 10, "Limit")
	optimizationsListCmd.Flags().Int("offset", 0, "Offset")
}
