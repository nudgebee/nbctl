package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/machinebox/graphql"
	"github.com/spf13/cobra"
	"nudgebee.com/nbctl/pkg/client"
	"nudgebee.com/nbctl/pkg/format"
)

var (
	userId      string
	userName    string
	getTenantId string
)

var adminUsersGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a single user by ID or username",
	RunE: func(cmd *cobra.Command, args []string) error {
		if userId == "" && userName == "" {
			return fmt.Errorf("either --id or --username is required")
		}

		client := client.NewClient()

		req := graphql.NewRequest(`
			query GetUsersByTenant($offset: Int, $limit: Int, $tenantId: uuid, $where: users_bool_exp) {
				users(limit: $limit, offset: $offset, order_by: {display_name:asc}, where: $where) {
					display_name
					id
					status
					username
					created_at
					user_roles(where: {entity_id: {_eq: $tenantId}}) {
						id
						role
						entity_type
						entity_id
						roleByRole {
							display_name
						}
					}
					tenants:tenantUsersByUser(where: {tenant: {_eq: $tenantId}}) {
						id:tenant
						created_at
					}
					usergroupUsersByUser {
						user_group {
							name
							id
							group_roles {
								role
							}
						}
					}
					user_auths(limit: 1, order_by: {accessed_at: desc},where: {tenant_id: {_eq: $tenantId}}) {
						accessed_at
						tenant_id
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
		req.Var("limit", 1)
		req.Var("tenantId", getTenantId)

		var respData struct {
			Users []struct {
				DisplayName string `json:"display_name"`
				ID          string `json:"id"`
				Status      string `json:"status"`
				Username    string `json:"username"`
				CreatedAt   string `json:"created_at"`
				UserRoles   []struct {
					ID         string `json:"id"`
					Role       string `json:"role"`
					EntityType string `json:"entity_type"`
					EntityID   string `json:"entity_id"`
					RoleByRole struct {
						DisplayName string `json:"display_name"`
					} `json:"roleByRole"`
				} `json:"user_roles"`
				Tenants []struct {
					ID        string `json:"id"`
					CreatedAt string `json:"created_at"`
				} `json:"tenants"`
				UsergroupUsersByUser []struct {
					UserGroup struct {
						Name       string `json:"name"`
						ID         string `json:"id"`
						GroupRoles []struct {
							Role string `json:"role"`
						} `json:"group_roles"`
					} `json:"user_group"`
				} `json:"usergroupUsersByUser"`
				UserAuths []struct {
					AccessedAt string `json:"accessed_at"`
					TenantID   string `json:"tenant_id"`
				} `json:"user_auths"`
			} `json:"users"`
		}
		if err := client.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		if len(respData.Users) == 0 {
			fmt.Println("User not found.")
			return nil
		}

		user := respData.Users[0]

		var roles []string
		for _, r := range user.UserRoles {
			roles = append(roles, r.RoleByRole.DisplayName)
		}

		var groups []string
		for _, g := range user.UsergroupUsersByUser {
			groups = append(groups, g.UserGroup.Name)
		}

		lastLogin := "never"
		if len(user.UserAuths) > 0 {
			lastLogin = user.UserAuths[0].AccessedAt
		}

		outputData := struct {
			ID          string
			DisplayName string
			Username    string
			Status      string
			Roles       string
			Groups      string
			LastLogin   string
		}{
			ID:          user.ID,
			DisplayName: user.DisplayName,
			Username:    user.Username,
			Status:      user.Status,
			Roles:       strings.Join(roles, ","),
			Groups:      strings.Join(groups, ","),
			LastLogin:   lastLogin,
		}

		format.GetFormat().Print(outputData)

		return nil
	},
}

func init() {
	adminUsersCmd.AddCommand(adminUsersGetCmd)
	adminUsersGetCmd.Flags().StringVar(&userId, "id", "", "User ID")
	adminUsersGetCmd.Flags().StringVar(&userName, "username", "", "Username")
	adminUsersGetCmd.Flags().StringVar(&getTenantId, "tenant-id", "", "Tenant ID")
}
