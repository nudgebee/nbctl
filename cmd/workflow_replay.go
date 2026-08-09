package cmd

import (
	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var workflowReplayCmd = &cobra.Command{
	Use:   "replay <execution-id>",
	Short: "Replay a previous or failed workflow execution",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		executionID := args[0]
		graphqlClient := client.NewClient()

		req := client.NewRequest(`
			mutation ReplayWorkflowExecution($request: WorkflowRetriggerRequest!) {
				workflow_replay_execution(request: $request) {
					execution_id
					status
					message
				}
			}
		`)
		req.Var("request", map[string]any{
			"execution_id": executionID,
		})

		var respData struct {
			WorkflowReplayExecution struct {
				ExecutionID string `json:"execution_id"`
				Status      string `json:"status"`
				Message     string `json:"message"`
			} `json:"workflow_replay_execution"`
		}

		if err := graphqlClient.Run(cmd.Context(), req, &respData); err != nil {
			return err
		}

		format.GetFormat().Print(respData.WorkflowReplayExecution)
		return nil
	},
}

func init() {
	workflowCmd.AddCommand(workflowReplayCmd)
}
