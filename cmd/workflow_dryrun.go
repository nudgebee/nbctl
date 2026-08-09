package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/spf13/cobra"
)

var workflowDryRunCmd = &cobra.Command{
	Use:   "dry-run <workflow-file>",
	Short: "Simulate and validate workflow execution without making live changes",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("reading workflow file: %w", err)
		}

		graphqlClient := client.NewClient()

		req := client.NewRequest(`
			mutation WorkflowDryRunExecute($content: String!) {
				workflow_dryrun_execute(content: $content) {
					valid
					status
					simulated_tasks {
						task_name
						status
						message
					}
				}
			}
		`)
		req.Var("content", string(content))

		var respData struct {
			WorkflowDryrunExecute struct {
				Valid          bool   `json:"valid"`
				Status         string `json:"status"`
				SimulatedTasks []struct {
					TaskName string `json:"task_name"`
					Status   string `json:"status"`
					Message  string `json:"message"`
				} `json:"simulated_tasks"`
			} `json:"workflow_dryrun_execute"`
		}

		if err := graphqlClient.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		if respData.WorkflowDryrunExecute.Valid {
			fmt.Printf("Workflow dry-run PASSED for %s (Status: %s)\n", filePath, respData.WorkflowDryrunExecute.Status)
		} else {
			fmt.Printf("Workflow dry-run FAILED for %s (Status: %s)\n", filePath, respData.WorkflowDryrunExecute.Status)
		}

		for i, t := range respData.WorkflowDryrunExecute.SimulatedTasks {
			fmt.Printf("  Step %d: %s -> %s (%s)\n", i+1, t.TaskName, t.Status, t.Message)
		}

		return nil
	},
}

func init() {
	workflowCmd.AddCommand(workflowDryRunCmd)
}
