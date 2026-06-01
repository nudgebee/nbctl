package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var workflowTriggerInputs []string

var workflowTriggerCmd = &cobra.Command{
	Use:   "trigger [id]",
	Short: "Trigger a workflow",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workflowId := args[0]
		graphqlClient := client.NewClient()

		query := `
mutation triggerWorkflow($request: WorkflowTriggerRequest!) {
  workflow_execute(request: $request) {
    id
    execution_id
  }
}
`
		req := client.NewRequest(query)
		accountId, _ := cmd.Flags().GetString("account-id")
		if accountId == "" {
			accountId = viper.GetString("account-id")
		}

		inputs := make(map[string]interface{})
		for _, input := range workflowTriggerInputs {
			parts := strings.SplitN(input, "=", 2)
			if len(parts) == 2 {
				inputs[parts[0]] = parts[1]
			} else {
				return fmt.Errorf("invalid input format: %s. Expected key=value", input)
			}
		}

		requestVar := map[string]interface{}{
			"account_id": accountId,
			"id":         workflowId,
			"inputs":     inputs,
		}

		req.Var("request", requestVar)

		var respData struct {
			WorkflowTrigger struct {
				ID          string `json:"id"`
				ExecutionID string `json:"execution_id"`
			} `json:"workflow_execute"`
		}

		if err := graphqlClient.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		if format.GetFormat().Get() == "json" {
			format.GetFormat().Print(respData.WorkflowTrigger)
		} else {
			_, _ = fmt.Fprintf(format.GetFormat().GetOutput(), "Workflow triggered. Execution ID: %s\n", respData.WorkflowTrigger.ExecutionID)
		}

		return nil
	},
}

func init() {
	workflowCmd.AddCommand(workflowTriggerCmd)
	workflowTriggerCmd.Flags().StringSliceVarP(&workflowTriggerInputs, "input", "i", []string{}, "Workflow inputs in key=value format")
	workflowTriggerCmd.Flags().String("account-id", "", "Account ID")
}
