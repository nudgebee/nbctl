package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

var (
	offset   int
	limit    int
	name     string
	username string
	status   string
	id       string
)

var adminUsersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all users",
	RunE: func(cmd *cobra.Command, args []string) error {
		graphqlClient := client.NewClient()

		req := client.NewRequest(`
			query GetUsersByTenant($offset: Int, $limit: Int, $where: users_bool_exp) {
				users(limit: $limit, offset: $offset, order_by: {display_name:asc}, where: $where) {
					display_name
					id
					status
					username
					created_at
					user_roles(where: {}) {
						id
						role
						entity_type
						entity_id
						roleByRole {
							display_name
						}
					}
					tenants:tenantUsersByUser(where: {}) {
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
					user_auths(limit: 1, order_by: {accessed_at: desc},where: {}) {
						accessed_at
						tenant_id
					}
				}
				users_aggregate(where: $where) {
					aggregate {
						count
					}
				}
			}
		`)

		req.Var("offset", offset)
		req.Var("limit", limit)

		where := make(map[string]any)
		if name != "" {
			where["display_name"] = map[string]any{"_ilike": fmt.Sprintf("%%%s%%", name)}
		}
		if username != "" {
			where["username"] = map[string]any{"_ilike": fmt.Sprintf("%%%s%%", username)}
		}
		if status != "" {
			where["status"] = map[string]any{"_eq": status}
		}
		if id != "" {
			where["id"] = map[string]any{"_eq": id}
		}
		req.Var("where", where)

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
			UsersAggregate struct {
				Aggregate struct {
					Count int `json:"count"`
				} `json:"aggregate"`
			} `json:"users_aggregate"`
		}
		if err := graphqlClient.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		var outputData []struct {
			ID          string
			DisplayName string
			Username    string
			Status      string
			Roles       string
			Groups      string
			LastLogin   string
		}

		for _, user := range respData.Users {
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

			outputData = append(outputData, struct {
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
				Roles:       strings.Join(lo.Uniq(roles), ","),
				Groups:      strings.Join(lo.Uniq(groups), ","),
				LastLogin:   lastLogin,
			})
		}

		table := format.TabularData{
			Data: outputData,
			Fields: []format.TableField{
				{Header: "ID", Field: "ID"},
				{Header: "Display Name", Field: "DisplayName"},
				{Header: "Username", Field: "Username"},
				{Header: "Status", Field: "Status"},
				{Header: "Roles", Field: "Roles"},
				{Header: "Groups", Field: "Groups"},
				{Header: "Last Login", Field: "LastLogin"},
			},
		}
		format.GetFormat().Print(table)

		fmt.Printf("\nTotal users: %d\n", respData.UsersAggregate.Aggregate.Count)

		return nil
	},
}

func init() {
	adminUsersCmd.AddCommand(adminUsersListCmd)
	adminUsersListCmd.Flags().IntVar(&offset, "offset", 0, "Offset for pagination")
	adminUsersListCmd.Flags().IntVar(&limit, "limit", 20, "Limit for pagination")
	adminUsersListCmd.Flags().StringVar(&name, "name", "", "Filter by name (ilike)")
	adminUsersListCmd.Flags().StringVar(&username, "username", "", "Filter by username (ilike)")
	adminUsersListCmd.Flags().StringVar(&status, "status", "", "Filter by status (eq)")
	adminUsersListCmd.Flags().StringVar(&id, "id", "", "Filter by id (eq)")
}
