package cmd

import (
	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
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

		type assignRoleResult struct {
			GroupID   string `json:"group_id"`
			Role      string `json:"role"`
			AccountID string `json:"account_id,omitempty"`
			Status    string `json:"status"`
			Message   string `json:"message,omitempty"`
		}

		if accountID != "" {
			// Account-scoped role assignment
			req := client.NewRequest(`
				mutation AssignAccountGroupRole($role: auth_account_group_roles_upsert_one_input!) {
					userroles_upsert_account_group(role: $role) {
						status
						message
					}
				}
			`)
			req.Var("role", map[string]any{
				"group_id": groupID,
				"account_roles": []map[string]string{
					{
						"account_id": accountID,
						"role":       role,
					},
				},
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
			format.GetFormat().Print(assignRoleResult{
				GroupID:   groupID,
				Role:      role,
				AccountID: accountID,
				Status:    respData.UserrolesUpsertAccountGroup.Status,
				Message:   respData.UserrolesUpsertAccountGroup.Message,
			})
		} else {
			// Tenant-level role assignment
			req := client.NewRequest(`
				mutation AssignGroupRole($role: auth_tenant_group_roles_upsert_one_input!) {
					userroles_upsert_group(role: $role) {
						status
						message
					}
				}
			`)
			req.Var("role", map[string]any{
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
			format.GetFormat().Print(assignRoleResult{
				GroupID: groupID,
				Role:    role,
				Status:  respData.UserrolesUpsertGroup.Status,
				Message: respData.UserrolesUpsertGroup.Message,
			})
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
