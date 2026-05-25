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

var workflowPauseCmd = &cobra.Command{
	Use:   "pause [id]",
	Short: "Pause an active workflow",
	Long:  `Pause an active workflow by ID. Paused workflows stop accepting new triggers until resumed.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workflowID := args[0]
		accountId := viper.GetString("account-id")
		if accountId == "" {
			return fmt.Errorf("account-id is required; set it in your profile with `nbctl configure add` or pass --account-id")
		}

		query := `
mutation pauseWorkflow($accountId: String!, $id: String!) {
  workflow_pause(request: {account_id: $accountId, id: $id}) {
    data
  }
}
`
		req := client.NewRequest(query)
		req.Var("accountId", accountId)
		req.Var("id", workflowID)

		var respData struct {
			WorkflowPause struct {
				Data json.RawMessage `json:"data"`
			} `json:"workflow_pause"`
		}

		c := client.NewClient()
		if err := c.Run(context.Background(), req, &respData); err != nil {
			return fmt.Errorf("failed to pause workflow: %w", err)
		}

		if format.GetFormat().Get() == "json" {
			format.GetFormat().Print(respData.WorkflowPause)
		} else {
			_, _ = fmt.Fprintf(format.GetFormat().GetOutput(), "Workflow %s paused.\n", workflowID)
		}

		return nil
	},
}

func init() {
	workflowCmd.AddCommand(workflowPauseCmd)
}
