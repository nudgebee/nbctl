package cmd

import (
	"fmt"
	"strings"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var workflowApproveCmd = &cobra.Command{
	Use:   "approve <execution-id>",
	Short: "Complete a human approval gate for a pending workflow execution",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		executionID := strings.TrimSpace(args[0])
		if executionID == "" {
			return fmt.Errorf("execution-id cannot be empty")
		}
		taskID, _ := cmd.Flags().GetString("task")
		if strings.TrimSpace(taskID) == "" {
			return fmt.Errorf("task flag is required and cannot be empty")
		}

		reject, _ := cmd.Flags().GetBool("reject")
		comments, _ := cmd.Flags().GetString("comments")

		status := "approved"
		if reject {
			status = "rejected"
		}

		req := client.NewRequest(`
			mutation CompleteWorkflowApproval($request: WorkflowCompleteApprovalRequest!) {
				workflow_complete_approval(request: $request) {
					status
					message
				}
			}
		`)
		input := map[string]any{
			"execution_id": executionID,
			"task_id":      taskID,
			"status":       status,
		}
		if comments != "" {
			input["comments"] = comments
		}
		req.Var("request", input)

		var respData struct {
			WorkflowCompleteApproval struct {
				Status  string `json:"status"`
				Message string `json:"message"`
			} `json:"workflow_complete_approval"`
		}

		if err := client.Run(cmd.Context(), req, &respData); err != nil {
			return err
		}

		format.GetFormat().Print(respData.WorkflowCompleteApproval)
		return nil
	},
}

func init() {
	workflowCmd.AddCommand(workflowApproveCmd)
	workflowApproveCmd.Flags().String("task", "", "Task ID waiting for approval (required)")
	workflowApproveCmd.Flags().Bool("reject", false, "Reject the approval step instead of approving")
	workflowApproveCmd.Flags().String("comments", "", "Optional comments for the approval decision")
	_ = workflowApproveCmd.MarkFlagRequired("task")
}
