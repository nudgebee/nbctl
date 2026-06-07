package cmd

import (
	"context"
	"encoding/json"
	"time"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var (
	workflowListStatus              string
	workflowListLastExecutionStatus string
	workflowListType                string
)

var workflowListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all workflows",
	RunE: func(cmd *cobra.Command, args []string) error {
		graphqlClient := client.NewClient()

		query := `
query ListWorkflows($accountId:String!, $status:String, $last_execution_status:String, $type:String) {
  workflow_list(request: {account_id: $accountId, status: $status, last_execution_status: $last_execution_status, type: $type}) {
    next_page_token
    workflows {
      account_id
      created_at
      created_by
      id
      last_execution_status
      name
      status
      tags
      tenant_id
      updated_at
      updated_by
      last_execution_time
      definition {
        output
        timeout
        version
        triggers {
          params
          type
        }
      }
    }
  }
}
`
		req := client.NewRequest(query)

		accountId, err := resolveAccountID(cmd)
		if err != nil {
			return err
		}
		req.Var("accountId", accountId)

		if workflowListStatus != "" {
			req.Var("status", workflowListStatus)
		}
		if workflowListLastExecutionStatus != "" {
			req.Var("last_execution_status", workflowListLastExecutionStatus)
		}
		if workflowListType != "" {
			req.Var("type", workflowListType)
		}

		var respData struct {
			WorkflowList struct {
				NextPageToken *string `json:"next_page_token"`
				Workflows     []struct {
					ID                  string          `json:"id"`
					Name                string          `json:"name"`
					Status              string          `json:"status"`
					LastExecutionStatus string          `json:"last_execution_status"`
					LastExecutionTime   *string         `json:"last_execution_time"`
					CreatedAt           string          `json:"created_at"`
					UpdatedAt           string          `json:"updated_at"`
					Definition          json.RawMessage `json:"definition"`
				} `json:"workflows"`
			} `json:"workflow_list"`
		}

		if err := graphqlClient.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		var outputData []struct {
			ID                  string
			Name                string
			Status              string
			LastExecutionStatus string
			LastExecutionTime   string
			CreatedAt           string
		}

		for _, wf := range respData.WorkflowList.Workflows {
			lastExecTime := "N/A"
			if wf.LastExecutionTime != nil {
				// Try to format it if possible, or just print as is
				t, err := time.Parse(time.RFC3339, *wf.LastExecutionTime)
				if err == nil {
					lastExecTime = t.Format(time.RFC3339)
				} else {
					lastExecTime = *wf.LastExecutionTime
				}
			}

			outputData = append(outputData, struct {
				ID                  string
				Name                string
				Status              string
				LastExecutionStatus string
				LastExecutionTime   string
				CreatedAt           string
			}{
				ID:                  wf.ID,
				Name:                wf.Name,
				Status:              wf.Status,
				LastExecutionStatus: wf.LastExecutionStatus,
				LastExecutionTime:   lastExecTime,
				CreatedAt:           wf.CreatedAt,
			})
		}

		table := format.TabularData{
			Data: outputData,
			Fields: []format.TableField{
				{Header: "ID", Field: "ID"},
				{Header: "Name", Field: "Name"},
				{Header: "Status", Field: "Status"},
				{Header: "Last Execution Status", Field: "LastExecutionStatus"},
				{Header: "Last Execution Time", Field: "LastExecutionTime"},
				{Header: "Created At", Field: "CreatedAt"},
			},
		}
		if format.GetFormat().Get() == "json" {
			format.GetFormat().Print(respData.WorkflowList.Workflows)
		} else {
			format.GetFormat().Print(table)
		}

		return nil
	},
}

func init() {
	workflowCmd.AddCommand(workflowListCmd)
	workflowListCmd.Flags().String("account-id", "", "Account ID (overrides profile)")
	workflowListCmd.Flags().StringVar(&workflowListStatus, "status", "", "Filter by status")
	workflowListCmd.Flags().StringVar(&workflowListLastExecutionStatus, "last-execution-status", "", "Filter by last execution status")
	workflowListCmd.Flags().StringVar(&workflowListType, "type", "", "Filter by type")
}
