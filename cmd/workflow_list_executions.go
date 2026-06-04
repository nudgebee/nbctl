package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	workflowListExecutionsStatus      string
	workflowListExecutionsTriggerType string
	workflowListExecutionsLimit       int
	workflowListExecutionsPageToken   string
)

var workflowListExecutionsCmd = &cobra.Command{
	Use:   "list-executions [workflow-id]",
	Short: "List executions for a workflow",
	Long:  `List execution history for a specific workflow, with optional status and trigger-type filters.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workflowID := args[0]
		accountID := viper.GetString("account-id")
		if accountID == "" {
			return fmt.Errorf("account-id is required; set it in your profile with `nbctl configure add` or pass --account-id")
		}

		query := `
query listWorkflowExecutions($accountId:String!, $id:String!, $limit:Int, $next_page_token:String, $status:String, $trigger_type:String) {
  workflow_list_executions(request: {account_id: $accountId, id: $id, limit: $limit, next_page_token: $next_page_token, status: $status, trigger_type: $trigger_type}) {
    next_page_token
    executions {
      id
      workflow_id
      parent_workflow_id
      status
      trigger_type
      triggered_by
      start_time
      close_time
    }
  }
}
`
		req := client.NewRequest(query)
		req.Var("accountId", accountID)
		req.Var("id", workflowID)
		if workflowListExecutionsStatus != "" {
			req.Var("status", workflowListExecutionsStatus)
		}
		if workflowListExecutionsTriggerType != "" {
			req.Var("trigger_type", workflowListExecutionsTriggerType)
		}
		if workflowListExecutionsLimit > 0 {
			req.Var("limit", workflowListExecutionsLimit)
		}
		if workflowListExecutionsPageToken != "" {
			req.Var("next_page_token", workflowListExecutionsPageToken)
		}

		var respData struct {
			WorkflowListExecutions struct {
				NextPageToken *string `json:"next_page_token"`
				Executions    []struct {
					ID               string  `json:"id"`
					WorkflowID       string  `json:"workflow_id"`
					ParentWorkflowID *string `json:"parent_workflow_id"`
					Status           string  `json:"status"`
					TriggerType      string  `json:"trigger_type"`
					TriggeredBy      string  `json:"triggered_by"`
					StartTime        *string `json:"start_time"`
					CloseTime        *string `json:"close_time"`
				} `json:"executions"`
			} `json:"workflow_list_executions"`
		}

		c := client.NewClient()
		if err := c.Run(context.Background(), req, &respData); err != nil {
			return fmt.Errorf("failed to list workflow executions: %w", err)
		}

		if format.GetFormat().Get() == "json" {
			format.GetFormat().Print(respData.WorkflowListExecutions)
			return nil
		}

		type row struct {
			ExecutionID string
			Status      string
			TriggerType string
			TriggeredBy string
			StartTime   string
			CloseTime   string
		}
		var rows []row
		for _, e := range respData.WorkflowListExecutions.Executions {
			rows = append(rows, row{
				ExecutionID: e.ID,
				Status:      e.Status,
				TriggerType: e.TriggerType,
				TriggeredBy: e.TriggeredBy,
				StartTime:   formatExecutionTime(e.StartTime),
				CloseTime:   formatExecutionTime(e.CloseTime),
			})
		}

		table := format.TabularData{
			Data: rows,
			Fields: []format.TableField{
				{Header: "Execution ID", Field: "ExecutionID"},
				{Header: "Status", Field: "Status"},
				{Header: "Trigger Type", Field: "TriggerType"},
				{Header: "Triggered By", Field: "TriggeredBy"},
				{Header: "Start Time", Field: "StartTime"},
				{Header: "Close Time", Field: "CloseTime"},
			},
		}
		format.GetFormat().Print(table)

		if respData.WorkflowListExecutions.NextPageToken != nil && *respData.WorkflowListExecutions.NextPageToken != "" {
			_, _ = fmt.Fprintf(format.GetFormat().GetOutput(), "\nNext page token: %s\n", *respData.WorkflowListExecutions.NextPageToken)
		}

		return nil
	},
}

func formatExecutionTime(ts *string) string {
	if ts == nil || *ts == "" {
		return "N/A"
	}
	if t, err := time.Parse(time.RFC3339, *ts); err == nil {
		return t.Format(time.RFC3339)
	}
	return *ts
}

func init() {
	workflowCmd.AddCommand(workflowListExecutionsCmd)
	workflowListExecutionsCmd.Flags().StringVar(&workflowListExecutionsStatus, "status", "", "Filter by execution status")
	workflowListExecutionsCmd.Flags().StringVar(&workflowListExecutionsTriggerType, "trigger-type", "", "Filter by trigger type")
	workflowListExecutionsCmd.Flags().IntVar(&workflowListExecutionsLimit, "limit", 0, "Maximum number of executions to return")
	workflowListExecutionsCmd.Flags().StringVar(&workflowListExecutionsPageToken, "page-token", "", "Pagination token from a previous response")
}
