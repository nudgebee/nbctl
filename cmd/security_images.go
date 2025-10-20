package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/machinebox/graphql"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"nudgebee.com/nbctl/pkg/client"
	"nudgebee.com/nbctl/pkg/format"
)

type SecurityImageRecommendation struct {
	Severity       string          `json:"severity"`
	Recommendation json.RawMessage `json:"recommendation"`
	Image          string          `json:"image"`
	ID             string          `json:"id"`
	Namespace      string          `json:"namespace"`
	WorkloadName   string          `json:"workload_name"`
	PackageID      string          `json:"package_id"`
}

type SecurityImageResponse struct {
	Recommendation struct {
		Rows []SecurityImageRecommendation `json:"rows"`
	} `json:"recommendation"`
	RecommendationAggregate struct {
		Rows []struct {
			Count int `json:"count"`
		} `json:"rows"`
	} `json:"recommendation_aggregate"`
}

var securityImagesCmd = &cobra.Command{
	Use:   "list-image-cves",
	Short: "List security recommendations for images",
	Long:  `Retrieves and displays security recommendations related to container images from the Nudgebee API.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := client.NewClient()

		accountId, _ := cmd.Flags().GetString("account-id")
		if accountId == "" {
			accountId = viper.GetString("account-id")
		}
		if accountId == "" {
			return fmt.Errorf("account-id is required")
		}

		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")
		severity, _ := cmd.Flags().GetString("severity")
		namespace, _ := cmd.Flags().GetString("namespace")
		workload, _ := cmd.Flags().GetString("workload")
		cve, _ := cmd.Flags().GetString("cve")
		packageId, _ := cmd.Flags().GetString("package-id")
		image, _ := cmd.Flags().GetString("image")

		whereParts := []string{
			"account_id:{_eq:$accountId}",
			"status:{_in:[\"Open\"]}",
		}
		if severity != "" {
			whereParts = append(whereParts, "severity:{_eq:$severity}")
		}
		if namespace != "" {
			whereParts = append(whereParts, "namespace:{_ilike:$namespace}")
		}
		if workload != "" {
			whereParts = append(whereParts, "workload_name:{_ilike:$workload}")
		}
		if cve != "" {
			whereParts = append(whereParts, "vulnerability_id:{_ilike:$cve}")
		}
		if packageId != "" {
			whereParts = append(whereParts, "package_id:{_ilike:$packageId}")
		}
		if image != "" {
			whereParts = append(whereParts, "image:{_ilike:$image}")
		}
		whereClause := strings.Join(whereParts, ", ")

		// Create a separate whereClause for the aggregate query, excluding vulnerability_id filter
		wherePartsForAggregate := []string{
			"account_id:{_eq:$accountId}",
			"status:{_in:[\"Open\"]}",
		}
		if severity != "" {
			wherePartsForAggregate = append(wherePartsForAggregate, "severity:{_eq:$severity}")
		}
		if namespace != "" {
			wherePartsForAggregate = append(wherePartsForAggregate, "namespace:{_ilike:$namespace}")
		}
		if workload != "" {
			wherePartsForAggregate = append(wherePartsForAggregate, "workload_name:{_ilike:$workload}")
		}
		// Exclude cve filter from aggregate query
		if packageId != "" {
			wherePartsForAggregate = append(wherePartsForAggregate, "package_id:{_ilike:$packageId}")
		}
		if image != "" {
			wherePartsForAggregate = append(wherePartsForAggregate, "image:{_ilike:$image}")
		}
		whereClauseForAggregate := strings.Join(wherePartsForAggregate, ", ")

		// Build the arguments for recommendation_security_v2
		argsParts := []string{
			fmt.Sprintf("where: {%s}", whereClause),
			"limit: $limit",
			"offset:$offset",
			"order_by: [{column:\"severity_weight\", order: desc}]",
		}
		argsClause := strings.Join(argsParts, ", ")

		query := fmt.Sprintf(`
			query list_k8_recommendation($limit:Int, $offset:Int, $accountId: String!, $severity: String, $namespace: String, $workload: String, $cve: String, $packageId: String, $image: String) {
			  recommendation: recommendation_security_v2(%s) {
			rows{
				  severity
				  recommendation
				  image
				  id      
				  namespace
				  workload_name
				  package_id
				}
			  }
			  recommendation_aggregate: recommendation_security_groupings_v2(where: {%s}) {
			rows{
				  count
				}
			  }
			}
		`, argsClause, whereClauseForAggregate) // Use whereClauseForAggregate for aggregate query

		req := graphql.NewRequest(query)

		req.Var("limit", limit)
		req.Var("offset", offset)
		req.Var("accountId", accountId)
		if severity != "" {
			req.Var("severity", severity)
		}
		if namespace != "" {
			req.Var("namespace", namespace)
		}
		if workload != "" {
			req.Var("workload", workload)
		}
		if cve != "" {
			req.Var("cve", cve)
		}
		if packageId != "" {
			req.Var("packageId", packageId)
		}
		if image != "" {
			req.Var("image", image)
		}

		var respData SecurityImageResponse
		if err := client.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		// Print the recommendations
		format.GetFormat().Print("Security Recommendations:")
		if len(respData.Recommendation.Rows) > 0 {
			// Define the table headers and fields
			table := format.TabularData{
				Data: respData.Recommendation.Rows,
				Fields: []format.TableField{
					{Header: "Severity", Field: "Severity"},
					{Header: "Image", Field: "Image"},
					{Header: "Workload", Field: "WorkloadName"},
					{Header: "Namespace", Field: "Namespace"},
					{Header: "Package ID", Field: "PackageID"},
					{Header: "CVE", Field: "Recommendation"},
				},
			}
			format.GetFormat().Print(table)
		} else {
			format.GetFormat().Print("No security recommendations found.")
		}

		// Print the aggregate count
		if len(respData.RecommendationAggregate.Rows) > 0 {
			format.GetFormat().Print(fmt.Sprintf("\nTotal Recommendations: %d\n", respData.RecommendationAggregate.Rows[0].Count))
		}

		return nil
	},
}

func init() {
	securityCmd.AddCommand(securityImagesCmd)
	securityImagesCmd.Flags().String("account-id", "", "Account ID")
	securityImagesCmd.Flags().Int("limit", 10, "Number of recommendations to retrieve")
	securityImagesCmd.Flags().Int("offset", 0, "Offset for pagination")
	securityImagesCmd.Flags().String("severity", "", "Filter by severity (e.g., Critical, High, Medium, Low)")
	securityImagesCmd.Flags().String("namespace", "", "Filter by namespace (case-insensitive like)")
	securityImagesCmd.Flags().String("workload", "", "Filter by workload name (case-insensitive like)")
	securityImagesCmd.Flags().String("cve", "", "Filter by CVE ID (case-insensitive like)")
	securityImagesCmd.Flags().String("package-id", "", "Filter by package ID (case-insensitive like)")
	securityImagesCmd.Flags().String("image", "", "Filter by image name (case-insensitive like)")
}
