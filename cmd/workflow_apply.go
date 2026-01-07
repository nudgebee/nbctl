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

		// Ensure account_id is set if not present in the file
		accountId := viper.GetString("account-id")
		if _, ok := workflowData["account_id"]; !ok {
			workflowData["account_id"] = accountId
		}

		graphqlClient := client.NewClient()

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

		if err := graphqlClient.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		if format.GetFormat().Get() == "json" {
			format.GetFormat().Print(respData.WorkflowCreate)
		} else {
			_, _ = fmt.Fprintf(format.GetFormat().GetOutput(), "Workflow created with ID: %s\n", respData.WorkflowCreate.ID)
		}

		return nil
	},
}

func init() {
	workflowCmd.AddCommand(workflowApplyCmd)
}
