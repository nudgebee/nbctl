package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var workflowResumeCmd = &cobra.Command{
	Use:   "resume [id]",
	Short: "Resume a paused workflow",
	Long:  `Resume a previously paused workflow by ID.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workflowID := args[0]
		accountId := viper.GetString("account-id")
		if accountId == "" {
			return fmt.Errorf("account-id is required; set it in your profile with `nbctl configure add` or pass --account-id")
		}

		query := `
mutation resumeWorkflow($accountId: String!, $id: String!) {
  workflow_resume(request: {account_id: $accountId, id: $id}) {
    data
  }
}
`
		req := client.NewRequest(query)
		req.Var("accountId", accountId)
		req.Var("id", workflowID)

		var respData struct {
			WorkflowResume struct {
				Data json.RawMessage `json:"data"`
			} `json:"workflow_resume"`
		}

		c := client.NewClient()
		if err := c.Run(context.Background(), req, &respData); err != nil {
			return fmt.Errorf("failed to resume workflow: %w", err)
		}

		if format.GetFormat().Get() == "json" {
			format.GetFormat().Print(respData.WorkflowResume)
		} else {
			_, _ = fmt.Fprintf(format.GetFormat().GetOutput(), "Workflow %s resumed.\n", workflowID)
		}

		return nil
	},
}

func init() {
	workflowCmd.AddCommand(workflowResumeCmd)
}
