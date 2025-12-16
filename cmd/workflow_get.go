package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/machinebox/graphql"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"nudgebee.com/nbctl/pkg/client"
	"nudgebee.com/nbctl/pkg/format"
)

var workflowGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get workflow details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workflowId := args[0]
		client := client.NewClient()

		query := `
query GetWorkflowById($accountId:String!, $workflowId:String!) {
  workflow_get(request: {account_id: $accountId, id: $workflowId}) {
    definition {
      output
      timeout
      version
      inputs {
        default
        description
        id
        type
      }
      tasks {
        depends_on
        id
        if
        matrix
        outputs
        params
        timeout
        type
        hooks {
          always {
            params
            type
          }
          failure {
            params
            type
          }
          success {
            params
            type
          }
        }
      }
      retry_policy {
        backoff_coefficient
        initial_interval
        maximum_attempts
        maximum_interval
        non_retryable_error_types
      }
      triggers {
        params
        type
      }
      hooks {
        always {
          params
          type
        }
        failure {
          params
          type
        }
        success {
          params
          type
        }
      }
    }
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
  }
}
`
		req := graphql.NewRequest(query)
		accountId := viper.GetString("account-id")
		req.Var("accountId", accountId)
		req.Var("workflowId", workflowId)

		var respData struct {
			WorkflowGet struct {
				ID                  string          `json:"id"`
				Name                string          `json:"name"`
				Status              string          `json:"status"`
				LastExecutionStatus string          `json:"last_execution_status"`
				AccountID           string          `json:"account_id"`
				CreatedAt           string          `json:"created_at"`
				UpdatedAt           string          `json:"updated_at"`
				Definition          json.RawMessage `json:"definition"`
				Tags                json.RawMessage `json:"tags"`
			} `json:"workflow_get"`
		}

		if err := client.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		if format.GetFormat().Get() == "json" {
			format.GetFormat().Print(respData.WorkflowGet)
		} else {
			out := format.GetFormat().GetOutput()
			fmt.Fprintf(out, "ID: %s\n", respData.WorkflowGet.ID)
			fmt.Fprintf(out, "Name: %s\n", respData.WorkflowGet.Name)
			fmt.Fprintf(out, "Status: %s\n", respData.WorkflowGet.Status)
			fmt.Fprintf(out, "Last Execution Status: %s\n", respData.WorkflowGet.LastExecutionStatus)
			fmt.Fprintf(out, "Created At: %s\n", respData.WorkflowGet.CreatedAt)
			fmt.Fprintf(out, "Updated At: %s\n", respData.WorkflowGet.UpdatedAt)
			fmt.Fprintf(out, "Definition:\n")
			definitionIndented, _ := json.MarshalIndent(respData.WorkflowGet.Definition, "", "  ")
			fmt.Fprintln(out, string(definitionIndented))
		}

		return nil
	},
}

func init() {
	workflowCmd.AddCommand(workflowGetCmd)
}
