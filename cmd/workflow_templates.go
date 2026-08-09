package cmd

import (
	"fmt"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var workflowTemplatesCmd = &cobra.Command{
	Use:   "templates",
	Short: "Browse and inspect pre-built workflow templates",
}

var workflowTemplatesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List pre-built workflow templates",
	RunE: func(cmd *cobra.Command, args []string) error {
		typeFlag, _ := cmd.Flags().GetString("type")
		category, _ := cmd.Flags().GetString("category")
		limit, _ := cmd.Flags().GetInt("limit")

		if typeFlag != "system" && typeFlag != "custom" && typeFlag != "all" {
			return fmt.Errorf("invalid template type '%s': must be 'system', 'custom', or 'all'", typeFlag)
		}

		graphqlClient := client.NewClient()

		req := client.NewRequest(`
			query ListWorkflowTemplates($request: WorkflowListTemplateRequest!) {
				workflow_list_template(request: $request) {
					total_count
					templates {
						id
						name
						description
						category
						is_system
						status
					}
				}
			}
		`)

		input := map[string]any{
			"type":  typeFlag,
			"limit": limit,
		}
		if category != "" {
			input["category"] = category
		}
		req.Var("request", input)

		var respData struct {
			WorkflowListTemplate struct {
				TotalCount int `json:"total_count"`
				Templates  []struct {
					ID          string `json:"id"`
					Name        string `json:"name"`
					Description string `json:"description"`
					Category    string `json:"category"`
					IsSystem    bool   `json:"is_system"`
					Status      string `json:"status"`
				} `json:"templates"`
			} `json:"workflow_list_template"`
		}

		if err := graphqlClient.Run(cmd.Context(), req, &respData); err != nil {
			return err
		}

		table := format.TabularData{
			Data: respData.WorkflowListTemplate.Templates,
			Fields: []format.TableField{
				{Header: "Template ID", Field: "ID"},
				{Header: "Template Name", Field: "Name"},
				{Header: "Category", Field: "Category"},
				{Header: "System", Field: "IsSystem"},
				{Header: "Description", Field: "Description"},
			},
		}
		format.GetFormat().Print(table)

		return nil
	},
}

var workflowTemplatesGetCmd = &cobra.Command{
	Use:   "get <template-id>",
	Short: "Get details for a specific workflow template",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		templateID := args[0]
		typeFlag, _ := cmd.Flags().GetString("type")

		if typeFlag != "" && typeFlag != "system" && typeFlag != "custom" && typeFlag != "all" {
			return fmt.Errorf("invalid template type '%s': must be 'system', 'custom', or 'all'", typeFlag)
		}

		typeVal := typeFlag
		if typeVal == "" {
			typeVal = "all"
		}

		graphqlClient := client.NewClient()

		req := client.NewRequest(`
			query GetWorkflowTemplate($request: WorkflowGetTemplateRequest!) {
				workflow_get_template(request: $request) {
					id
					name
					description
					category
					is_system
					status
				}
			}
		`)
		req.Var("request", map[string]any{
			"type": typeVal,
			"id":   templateID,
		})

		var respData struct {
			WorkflowGetTemplate struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Description string `json:"description"`
				Category    string `json:"category"`
				IsSystem    bool   `json:"is_system"`
				Status      string `json:"status"`
			} `json:"workflow_get_template"`
		}

		if err := graphqlClient.Run(cmd.Context(), req, &respData); err != nil {
			return err
		}

		if respData.WorkflowGetTemplate.ID == "" {
			return fmt.Errorf("workflow template '%s' not found", templateID)
		}

		format.GetFormat().Print(respData.WorkflowGetTemplate)
		return nil
	},
}

func init() {
	workflowCmd.AddCommand(workflowTemplatesCmd)
	workflowTemplatesCmd.AddCommand(workflowTemplatesListCmd)
	workflowTemplatesCmd.AddCommand(workflowTemplatesGetCmd)

	workflowTemplatesListCmd.Flags().String("type", "system", "Template type (system, custom, or all)")
	workflowTemplatesListCmd.Flags().String("category", "", "Filter templates by category")
	workflowTemplatesListCmd.Flags().Int("limit", 50, "Maximum number of templates to return")

	workflowTemplatesGetCmd.Flags().String("type", "", "Optional template type filter (system, custom, or all)")
}
