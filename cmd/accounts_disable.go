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
			mutation UpdateCloudAccount($object: cloud_account_update_input!) {
				cloud_account_update(object: $object) {
					affected_rows
				}
			}
		`)

		req.Var("object", map[string]any{
			"id":     accountID,
			"status": "disabled",
		})

		var respData struct {
			CloudAccountUpdate struct {
				AffectedRows int `json:"affected_rows"`
			} `json:"cloud_account_update"`
		}

		if err := graphqlClient.Run(context.Background(), req, &respData); err != nil {
			return fmt.Errorf("failed to disable account: %w", err)
		}

		if respData.CloudAccountUpdate.AffectedRows == 0 {
			return fmt.Errorf("account %s not found or not updated", accountID)
		}

		output := struct {
			Message string `json:"message"`
		}{
			Message: fmt.Sprintf("Account %s disabled", accountID),
		}

		format.GetFormat().Print(output)

		return nil
	},
}

func init() {
}
