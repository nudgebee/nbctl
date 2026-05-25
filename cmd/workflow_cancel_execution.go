package cmd

import (
	"context"
	"fmt"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var workflowCancelExecutionCmd = &cobra.Command{
	Use:   "cancel-execution [workflow-id] [execution-id]",
	Short: "Cancel an in-progress workflow execution",
	Long:  `Cancel a specific running execution of a workflow.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		workflowID := args[0]
		executionID := args[1]
		accountId := viper.GetString("account-id")
		if accountId == "" {
			return fmt.Errorf("account-id is required; set it in your profile with `nbctl configure add` or pass --account-id")
		}

		query := `
mutation cancelWorkflowExecution($request: WorkflowCancelRequest!) {
  workflow_cancel_execution(request: $request) {
    message
  }
}
`
		req := client.NewRequest(query)
		req.Var("request", map[string]any{
			"account_id":   accountId,
			"id":           workflowID,
			"execution_id": executionID,
		})

		var respData struct {
			WorkflowCancelExecution struct {
				Message string `json:"message"`
			} `json:"workflow_cancel_execution"`
		}

		c := client.NewClient()
		if err := c.Run(context.Background(), req, &respData); err != nil {
			return fmt.Errorf("failed to cancel workflow execution: %w", err)
		}

		if format.GetFormat().Get() == "json" {
			format.GetFormat().Print(respData.WorkflowCancelExecution)
		} else {
			msg := respData.WorkflowCancelExecution.Message
			if msg == "" {
				msg = fmt.Sprintf("Execution %s of workflow %s cancelled.", executionID, workflowID)
			}
			_, _ = fmt.Fprintf(format.GetFormat().GetOutput(), "%s\n", msg)
		}

		return nil
	},
}

func init() {
	workflowCmd.AddCommand(workflowCancelExecutionCmd)
}
