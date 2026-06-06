package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var workflowGetExecutionCmd = &cobra.Command{
	Use:   "get-execution [workflow-id] [execution-id]",
	Short: "Get details of a workflow execution",
	Long:  `Fetch a single workflow execution, including its tasks, inputs, outputs, and any errors.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		workflowID := args[0]
		executionID := args[1]
		accountID, _ := cmd.Flags().GetString("account-id")
		if accountID == "" {
			accountID = viper.GetString("account-id")
		}
		if accountID == "" {
			return fmt.Errorf("account-id is required; set it in your profile with `nbctl configure add` or pass --account-id")
		}

		query := `
query getWorkflowExecution($request: WorkflowExecutionGetRequest!) {
  workflow_get_execution(request: $request) {
    id
    workflow_id
    parent_workflow_id
    triggered_by
    status
    start_time
    close_time
    inputs
    workflow_result
    error
    definition
    definition_source
    version_id
    version_number
    version_name
    tasks {
      id
      type
      status
      start_time
      end_time
      input
      output
      error
      attempt
      children
    }
  }
}
`
		req := client.NewRequest(query)
		req.Var("request", map[string]any{
			"account_id":   accountID,
			"id":           executionID,
			"workflow_id":  workflowID,
		})

		var respData struct {
			WorkflowGetExecution struct {
				ID               string          `json:"id"`
				WorkflowID       string          `json:"workflow_id"`
				ParentWorkflowID *string         `json:"parent_workflow_id"`
				TriggeredBy      string          `json:"triggered_by"`
				Status           string          `json:"status"`
				StartTime        *string         `json:"start_time"`
				CloseTime        *string         `json:"close_time"`
				Inputs           json.RawMessage `json:"inputs"`
				WorkflowResult   json.RawMessage `json:"workflow_result"`
				Error            *string         `json:"error"`
				Definition       json.RawMessage `json:"definition"`
				DefinitionSource *string         `json:"definition_source"`
				VersionID        *string         `json:"version_id"`
				VersionNumber    *int            `json:"version_number"`
				VersionName      *string         `json:"version_name"`
				Tasks            []struct {
					ID        string          `json:"id"`
					Type      string          `json:"type"`
					Status    string          `json:"status"`
					StartTime *string         `json:"start_time"`
					EndTime   *string         `json:"end_time"`
					Input     json.RawMessage `json:"input"`
					Output    json.RawMessage `json:"output"`
					Error     *string         `json:"error"`
					Attempt   *int            `json:"attempt"`
					Children  json.RawMessage `json:"children"`
				} `json:"tasks"`
			} `json:"workflow_get_execution"`
		}

		c := client.NewClient()
		if err := c.Run(cmd.Context(), req, &respData); err != nil {
			return fmt.Errorf("failed to get workflow execution: %w", err)
		}

		exec := respData.WorkflowGetExecution
		if format.GetFormat().Get() == "json" {
			format.GetFormat().Print(exec)
			return nil
		}

		out := format.GetFormat().GetOutput()
		_, _ = fmt.Fprintf(out, "Execution ID: %s\n", exec.ID)
		_, _ = fmt.Fprintf(out, "Workflow ID:  %s\n", exec.WorkflowID)
		_, _ = fmt.Fprintf(out, "Status:       %s\n", exec.Status)
		_, _ = fmt.Fprintf(out, "Triggered By: %s\n", exec.TriggeredBy)
		_, _ = fmt.Fprintf(out, "Start Time:   %s\n", formatExecutionTime(exec.StartTime))
		_, _ = fmt.Fprintf(out, "Close Time:   %s\n", formatExecutionTime(exec.CloseTime))
		if exec.VersionNumber != nil {
			versionName := ""
			if exec.VersionName != nil {
				versionName = " (" + *exec.VersionName + ")"
			}
			_, _ = fmt.Fprintf(out, "Version:      %d%s\n", *exec.VersionNumber, versionName)
		}
		if exec.Error != nil && *exec.Error != "" {
			_, _ = fmt.Fprintf(out, "Error:        %s\n", *exec.Error)
		}

		if len(exec.Inputs) > 0 && string(exec.Inputs) != "null" {
			_, _ = fmt.Fprintf(out, "\nInputs:\n")
			pretty, _ := json.MarshalIndent(exec.Inputs, "", "  ")
			_, _ = fmt.Fprintln(out, string(pretty))
		}

		if len(exec.WorkflowResult) > 0 && string(exec.WorkflowResult) != "null" {
			_, _ = fmt.Fprintf(out, "\nResult:\n")
			pretty, _ := json.MarshalIndent(exec.WorkflowResult, "", "  ")
			_, _ = fmt.Fprintln(out, string(pretty))
		}

		if len(exec.Tasks) > 0 {
			_, _ = fmt.Fprintf(out, "\nTasks (%d):\n", len(exec.Tasks))
			type taskRow struct {
				ID        string
				Type      string
				Status    string
				StartTime string
				EndTime   string
				Attempt   string
			}
			var rows []taskRow
			for _, t := range exec.Tasks {
				attempt := "-"
				if t.Attempt != nil {
					attempt = fmt.Sprintf("%d", *t.Attempt)
				}
				rows = append(rows, taskRow{
					ID:        t.ID,
					Type:      t.Type,
					Status:    t.Status,
					StartTime: formatExecutionTime(t.StartTime),
					EndTime:   formatExecutionTime(t.EndTime),
					Attempt:   attempt,
				})
			}
			format.GetFormat().Print(format.TabularData{
				Data: rows,
				Fields: []format.TableField{
					{Header: "Task ID", Field: "ID"},
					{Header: "Type", Field: "Type"},
					{Header: "Status", Field: "Status"},
					{Header: "Start Time", Field: "StartTime"},
					{Header: "End Time", Field: "EndTime"},
					{Header: "Attempt", Field: "Attempt"},
				},
			})
		}

		return nil
	},
}

func init() {
	workflowCmd.AddCommand(workflowGetExecutionCmd)
	workflowGetExecutionCmd.Flags().String("account-id", "", "Account ID (overrides profile)")
}
