package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var workflowApplyCmd = &cobra.Command{
	Use:   "apply [file]",
	Short: "Create or update a workflow from a YAML file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]
		yamlContent, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		var workflowData map[string]interface{}
		if err := yaml.Unmarshal(yamlContent, &workflowData); err != nil {
			return fmt.Errorf("failed to parse YAML: %w", err)
		}

		workflowName, ok := workflowData["name"].(string)
		if !ok || workflowName == "" {
			return fmt.Errorf("workflow name is required in YAML")
		}

		// Ensure account_id is set if not present in the file
		accountId := viper.GetString("account-id")
		if _, ok := workflowData["account_id"]; !ok {
			workflowData["account_id"] = accountId
		}

		c := client.NewClient()

		existingID, err := findWorkflowByName(c, accountId, workflowName)
		if err != nil {
			return fmt.Errorf("failed to check existing workflows: %w", err)
		}

		if existingID != "" {
			// Update existing workflow
			query := `
mutation UpdateWorkflow($request: WorkflowUpdateRequest!) {
  workflow_update(request: $request) {
    id
  }
}
`
			req := client.NewRequest(query)

			requestVar := map[string]interface{}{
				"account_id": accountId,
				"id":         existingID,
				"workflow":   workflowData,
			}

			req.Var("request", requestVar)

			var respData struct {
				WorkflowUpdate struct {
					ID string `json:"id"`
				} `json:"workflow_update"`
			}

			if err := c.Run(context.Background(), req, &respData); err != nil {
				return err
			}

			if format.GetFormat().Get() == "json" {
				format.GetFormat().Print(respData.WorkflowUpdate)
			} else {
				_, _ = fmt.Fprintf(format.GetFormat().GetOutput(), "Workflow updated with ID: %s\n", respData.WorkflowUpdate.ID)
			}

		} else {
			// Create new workflow
			query := `
mutation CreateWorkflow($request: WorkflowCreateRequest!) {
  workflow_create(request: $request) {
    id
  }
}
`
			req := client.NewRequest(query)

			requestVar := map[string]interface{}{
				"account_id": accountId,
				"workflow":   workflowData,
			}

			req.Var("request", requestVar)

			var respData struct {
				WorkflowCreate struct {
					ID string `json:"id"`
				} `json:"workflow_create"`
			}

			if err := c.Run(context.Background(), req, &respData); err != nil {
				return err
			}

			if format.GetFormat().Get() == "json" {
				format.GetFormat().Print(respData.WorkflowCreate)
			} else {
				_, _ = fmt.Fprintf(format.GetFormat().GetOutput(), "Workflow created with ID: %s\n", respData.WorkflowCreate.ID)
			}
		}

		return nil
	},
}

func findWorkflowByName(c *client.Client, accountId, name string) (string, error) {
	query := `
query ListWorkflows($accountId:String!, $pageToken:String) {
  workflow_list(request: {account_id: $accountId, page_token: $pageToken}) {
    next_page_token
    workflows {
      id
      name
    }
  }
}
`
	var pageToken *string
	for {
		req := client.NewRequest(query)
		req.Var("accountId", accountId)
		if pageToken != nil {
			req.Var("pageToken", *pageToken)
		}

		var respData struct {
			WorkflowList struct {
				NextPageToken *string `json:"next_page_token"`
				Workflows     []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"workflows"`
			} `json:"workflow_list"`
		}

		if err := c.Run(context.Background(), req, &respData); err != nil {
			return "", err
		}

		for _, wf := range respData.WorkflowList.Workflows {
			if wf.Name == name {
				return wf.ID, nil
			}
		}

		if respData.WorkflowList.NextPageToken == nil || *respData.WorkflowList.NextPageToken == "" {
			break
		}
		pageToken = respData.WorkflowList.NextPageToken
	}
	return "", nil
}

func init() {
	workflowCmd.AddCommand(workflowApplyCmd)
}