package cmd

import (
	"context"
	"fmt"

	"github.com/machinebox/graphql"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"nudgebee.com/nbctl/pkg/client"
	"nudgebee.com/nbctl/pkg/format"
)

var workflowTriggerCmd = &cobra.Command{
	Use:   "trigger [id]",
	Short: "Trigger a workflow",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workflowId := args[0]
		client := client.NewClient()

		query := `
mutation triggerWorkflow($request: WorkflowTriggerRequest!) {
  workflow_trigger(request: $request) {
    id
    execution_id
  }
}
`
		req := graphql.NewRequest(query)
		accountId := viper.GetString("account-id")

		requestVar := map[string]interface{}{
			"account_id": accountId,
			"id":         workflowId,
			"inputs":     map[string]interface{}{},
		}

		req.Var("request", requestVar)

		var respData struct {
			WorkflowTrigger struct {
				ID          string `json:"id"`
				ExecutionID string `json:"execution_id"`
			} `json:"workflow_trigger"`
		}

		if err := client.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		if format.GetFormat().Get() == "json" {
			format.GetFormat().Print(respData.WorkflowTrigger)
		} else {
			fmt.Fprintf(format.GetFormat().GetOutput(), "Workflow triggered. Execution ID: %s\n", respData.WorkflowTrigger.ExecutionID)
		}

		return nil
	},
}

func init() {
	workflowCmd.AddCommand(workflowTriggerCmd)
}
