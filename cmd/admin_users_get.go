package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var (
	userId   string
	userName string
)

var adminUsersGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a single user by ID or username",
	RunE: func(cmd *cobra.Command, args []string) error {
		if userId == "" && userName == "" {
			return fmt.Errorf("either --id or --username is required")
		}

		graphqlClient := client.NewClient()

		req := client.NewRequest(`
			query GetUsersByTenant($where: UsersByTenantWhereRequest!) {
				users: admin_get_users_by_tenant_v2(where: $where, limit: 1, offset: 0) {
					rows {
						display_name
						id
						status
						username
						created_at
						last_accessed_at
						user_roles
						user_groups
					}
				}
			}
		`)

		where := make(map[string]any)
		if userId != "" {
			where["id"] = map[string]any{"_eq": userId}
		}
		if userName != "" {
			where["username"] = map[string]any{"_eq": userName}
		}

		req.Var("where", where)

		var respData struct {
			Users struct {
				Rows []struct {
					DisplayName    string          `json:"display_name"`
					ID             string          `json:"id"`
					Status         string          `json:"status"`
					Username       string          `json:"username"`
					CreatedAt      string          `json:"created_at"`
					LastAccessedAt string          `json:"last_accessed_at"`
					UserRoles      json.RawMessage `json:"user_roles"`
					UserGroups     json.RawMessage `json:"user_groups"`
				} `json:"rows"`
			} `json:"users"`
		}
		if err := graphqlClient.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		if len(respData.Users.Rows) == 0 {
			fmt.Println("User not found.")
			return nil
		}

		user := respData.Users.Rows[0]
		output := struct {
			ID             string
			DisplayName    string
			Username       string
			Status         string
			CreatedAt      string
			LastAccessedAt string
			Roles          string
			Groups         string
		}{
			ID:             user.ID,
			DisplayName:    user.DisplayName,
			Username:       user.Username,
			Status:         user.Status,
			CreatedAt:      user.CreatedAt,
			LastAccessedAt: user.LastAccessedAt,
			Roles:          joinJSONNames(user.UserRoles, "role"),
			Groups:         joinJSONNames(user.UserGroups, "name"),
		}

		format.GetFormat().Print(output)

		return nil
	},
}

func init() {
	adminUsersCmd.AddCommand(adminUsersGetCmd)
	adminUsersGetCmd.Flags().StringVar(&userId, "id", "", "User ID")
	adminUsersGetCmd.Flags().StringVar(&userName, "username", "", "Username")
}
