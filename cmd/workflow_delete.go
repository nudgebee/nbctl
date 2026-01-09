package cmd

import (
	"context"
	"fmt"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var workflowDeleteCmd = &cobra.Command{
	Use:   "delete [id]",
	Short: "Delete a workflow",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workflowID := args[0]
		accountId := viper.GetString("account-id")

		query := `
mutation deleteWorkflow($accountId:String!, $id: String!) {
  workflow_delete(request: {account_id: $accountId, id: $id}){
    message
  }
}
`
		req := client.NewRequest(query)
		req.Var("accountId", accountId)
		req.Var("id", workflowID)

		var respData struct {
			WorkflowDelete struct {
				Message string `json:"message"`
			} `json:"workflow_delete"`
		}

		c := client.NewClient()
		if err := c.Run(context.Background(), req, &respData); err != nil {
			return fmt.Errorf("failed to delete workflow: %w", err)
		}

		if format.GetFormat().Get() == "json" {
			format.GetFormat().Print(respData.WorkflowDelete)
		} else {
			_, _ = fmt.Fprintf(format.GetFormat().GetOutput(), "%s\n", respData.WorkflowDelete.Message)
		}

		return nil
	},
}

func init() {
	workflowCmd.AddCommand(workflowDeleteCmd)
}