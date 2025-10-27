package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/machinebox/graphql"
	"github.com/spf13/cobra"
	"nudgebee.com/nbctl/pkg/client"
	"nudgebee.com/nbctl/pkg/format"
)

var optimizationsGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get an optimization by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := client.NewClient()

		id := args[0]

		req := graphql.NewRequest(`
			query GetRecommendationById($id: String!) {
			  recommendations_v2(where: {id: {_eq: $id}}) {
				rows {
					account_object_id
					category
					created_at
					dismissed_reason
					estimated_savings
					id
					is_dismissed
											note
											recommendation_action
											resource_cloud_service
											resource_id
											resource_k8s_namespace
											resource_meta
											resource_name
											resource_type
											rule_name
											severity
											status
											tenant_id
											updated_at
											updated_by
											recommendation					resource_id
					resource_k8s_namespace
					resource_meta
					resource_name
					resource_type
					rule_name
					severity
					status
					tenant_id
					updated_at
					updated_by
				}
			  }
			}
		`)

		req.Var("id", id)

		var respData struct {
			RecommendationsV2 struct {
				Rows []struct {
					AccountObjectID      string          `json:"account_object_id"`
					Category             string          `json:"category"`
					CreatedAt            string          `json:"created_at"`
					DismissedReason      string          `json:"dismissed_reason"`
					EstimatedSavings     float64         `json:"estimated_savings"`
					ID                   string          `json:"id"`
					IsDismissed          bool            `json:"is_dismissed"`
					Note                 string          `json:"note"`
					RecommendationAction string          `json:"recommendation_action"`
					ResourceCloudService string          `json:"resource_cloud_service"`
					ResourceID           string          `json:"resource_id"`
					ResourceK8sNamespace string          `json:"resource_k8s_namespace"`
					ResourceMeta         json.RawMessage `json:"resource_meta"`
					ResourceName         string          `json:"resource_name"`
					ResourceType         string          `json:"resource_type"`
					RuleName             string          `json:"rule_name"`
					Severity             string          `json:"severity"`
					Status               string          `json:"status"`
					TenantID             string          `json:"tenant_id"`
					UpdatedAt            string          `json:"updated_at"`
					UpdatedBy            string          `json:"updated_by"`
					Evidence             string          `json:"recommendation"`
				} `json:"rows"`
			} `json:"recommendations_v2"`
		}

		if err := client.Run(context.Background(), req, &respData); err != nil {
			fmt.Printf("GraphQL error: %v\n", err)
			return err
		}

		if len(respData.RecommendationsV2.Rows) == 0 {
			fmt.Println("Optimization not found.")
			return nil
		}

		// Get the recommendation object
		rec := respData.RecommendationsV2.Rows[0]

		// Unmarshal and pretty-print the Evidence field
		var evidenceData any
		if err := json.Unmarshal([]byte(rec.Evidence), &evidenceData); err != nil {
			return fmt.Errorf("failed to unmarshal evidence JSON: %w", err)
		}
		prettyEvidence, err := json.MarshalIndent(evidenceData, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal evidence JSON: %w", err)
		}
		rec.Evidence = string(prettyEvidence) // Assign pretty-printed JSON back to the field

		format.GetFormat().Print(rec)

		return nil
	},
}

func init() {
	optimizationsCmd.AddCommand(optimizationsGetCmd)
}
