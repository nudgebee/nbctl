package cmd

import (
	"context"
	"fmt"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var accountsDisableCmd = &cobra.Command{
	Use:   "disable [id]",
	Short: "Disable a Nudgebee account",
	Long:  `Disable a Nudgebee account by setting its status to 'disabled'.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		accountID := args[0]
		graphqlClient := client.NewClient()

		req := client.NewRequest(`
			mutation UpdateCloudAccount($id: uuid!, $status: cloud_account_status_type_enum!) {
				update_cloud_accounts(where: {id:{_eq:$id}}, _set: {status:$status}) {
					affected_rows
				}
			}
		`)

		req.Var("id", accountID)
		req.Var("status", "disabled")

		var respData struct {
			UpdateCloudAccounts struct {
				AffectedRows int `json:"affected_rows"`
			} `json:"update_cloud_accounts"`
		}

		if err := graphqlClient.Run(context.Background(), req, &respData); err != nil {
			return fmt.Errorf("failed to disable account: %w", err)
		}

		output := struct {
			Message string `json:"message"`
		}{
			Message: fmt.Sprintf("Account %s disabled. Affected rows: %d", accountID, respData.UpdateCloudAccounts.AffectedRows),
		}

		format.GetFormat().Print(output)

		return nil
	},
}

func init() {
}
