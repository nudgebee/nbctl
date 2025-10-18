package cmd

import (
	"context"
	"fmt"

	"github.com/machinebox/graphql"
	"github.com/spf13/cobra"
	"nudgebee.com/nbctl/pkg/client"
	"nudgebee.com/nbctl/pkg/format"
)

var accountsCreateAgentTokenCmd = &cobra.Command{
	Use:   "create-agent-token [account-id]",
	Short: "Create an agent token for a Kubernetes account",
	Long:  `Create an agent token for a specified Kubernetes account. This is only applicable for accounts of type Kubernetes.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		accountID := args[0]
		client := client.NewClient()

		req := graphql.NewRequest(`
			mutation CreateAgentToken($accountId: String!) {
				agent_token_create(object: {account_id: $accountId}) {
					access_secret
					account_id
					access_key
				}
			}
		`)

		req.Var("accountId", accountID)

		var respData struct {
			AgentTokenCreate struct {
				AccessSecret string `json:"access_secret"`
				AccountID    string `json:"account_id"`
				AccessKey    string `json:"access_key"`
			} `json:"agent_token_create"`
		}

		if err := client.Run(context.Background(), req, &respData); err != nil {
			return fmt.Errorf("failed to create agent token: %w", err)
		}

		output := struct {
			AccessKey    string `json:"access_key"`
			AccessSecret string `json:"access_secret"`
			AccountID    string `json:"account_id"`
		}{
			AccessKey:    respData.AgentTokenCreate.AccessKey,
			AccessSecret: respData.AgentTokenCreate.AccessSecret,
			AccountID:    respData.AgentTokenCreate.AccountID,
		}

		format.GetFormat().Print(output)

		return nil
	},
}

func init() {
}
