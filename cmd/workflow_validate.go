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

var workflowValidateCmd = &cobra.Command{
	Use:   "validate [file]",
	Short: "Validate a workflow from a YAML file",
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

		query := `
mutation ValidateWorkflow($request: WorkflowCreateRequest!) {
  workflow_validate(request: $request) {
    message
  }
}
`
		requestVar := map[string]interface{}{
			"account_id": accountId,
			"workflow":   workflowData,
		}

		req := client.NewRequest(query)
		req.Var("request", requestVar)

		var respData struct {
			WorkflowValidate struct {
				Message string `json:"message"`
			} `json:"workflow_validate"`
		}

		if err := client.Run(context.Background(), req, &respData); err != nil {
			// client.Run now handles the Hasura/webhook error logic
			return err
		}

		message := respData.WorkflowValidate.Message

		if format.GetFormat().Get() == "json" {
			// Construct response object for JSON output
			respObj := map[string]interface{}{
				"message": message,
				"valid":   true,
			}
			format.GetFormat().Print(respObj)
		} else {
			if message != "" {
				_, _ = fmt.Fprintln(format.GetFormat().GetOutput(), message)
			} else {
				_, _ = fmt.Fprintln(format.GetFormat().GetOutput(), "Workflow is valid")
			}
		}

		return nil
	},
}

func init() {
	workflowCmd.AddCommand(workflowValidateCmd)
}
