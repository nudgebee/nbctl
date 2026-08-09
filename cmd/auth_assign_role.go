package cmd

import (
	"fmt"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/spf13/cobra"
)

var authAssignRoleCmd = &cobra.Command{
	Use:   "assign-role",
	Short: "Assign a built-in or custom role to a user group",
	RunE: func(cmd *cobra.Command, args []string) error {
		groupID, _ := cmd.Flags().GetString("group-id")
		role, _ := cmd.Flags().GetString("role")
		accountID, _ := cmd.Flags().GetString("account-id")

		graphqlClient := client.NewClient()

		if accountID != "" {
			// Account-scoped role assignment
			req := client.NewRequest(`
				mutation AssignAccountGroupRole($request: UserRolesUpsertAccountGroupInput!) {
					userroles_upsert_account_group(request: $request) {
						status
						message
					}
				}
			`)
			req.Var("request", map[string]any{
				"group_id":   groupID,
				"role":       role,
				"account_id": accountID,
			})

			var respData struct {
				UserrolesUpsertAccountGroup struct {
					Status  string `json:"status"`
					Message string `json:"message"`
				} `json:"userroles_upsert_account_group"`
			}
			if err := graphqlClient.Run(cmd.Context(), req, &respData); err != nil {
				return err
			}
			fmt.Printf("Assigned role '%s' to group '%s' for account '%s' (Status: %s)\n", role, groupID, accountID, respData.UserrolesUpsertAccountGroup.Status)
		} else {
			// Tenant-level role assignment
			req := client.NewRequest(`
				mutation AssignGroupRole($request: UserRolesUpsertGroupInput!) {
					userroles_upsert_group(request: $request) {
						status
						message
					}
				}
			`)
			req.Var("request", map[string]any{
				"group_id": groupID,
				"role":     role,
			})

			var respData struct {
				UserrolesUpsertGroup struct {
					Status  string `json:"status"`
					Message string `json:"message"`
				} `json:"userroles_upsert_group"`
			}
			if err := graphqlClient.Run(cmd.Context(), req, &respData); err != nil {
				return err
			}
			fmt.Printf("Assigned role '%s' to group '%s' at tenant level (Status: %s)\n", role, groupID, respData.UserrolesUpsertGroup.Status)
		}

		return nil
	},
}

func init() {
	authCmd.AddCommand(authAssignRoleCmd)
	authAssignRoleCmd.Flags().String("group-id", "", "User Group ID (required)")
	authAssignRoleCmd.Flags().String("role", "", "Role name (e.g. tenant_admin, account_admin, or custom role ID) (required)")
	authAssignRoleCmd.Flags().String("account-id", "", "Account ID for account-scoped assignment (optional)")
	_ = authAssignRoleCmd.MarkFlagRequired("group-id")
	_ = authAssignRoleCmd.MarkFlagRequired("role")
}
