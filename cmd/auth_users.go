package cmd

import (
	"context"
	"fmt"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var authUsersCmd = &cobra.Command{
	Use:   "users",
	Short: "Manage tenant users",
}

var authUsersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tenant users with status and assigned roles",
	RunE: func(cmd *cobra.Command, args []string) error {
		graphqlClient := client.NewClient()

		req := client.NewRequest(`
			query ListUsersByTenant {
				users_list_by_tenant {
					rows {
						id
						username
						display_name
						status
						created_at
					}
				}
			}
		`)

		var respData struct {
			UsersListByTenant struct {
				Rows []struct {
					ID          string `json:"id"`
					Username    string `json:"username"`
					DisplayName string `json:"display_name"`
					Status      string `json:"status"`
					CreatedAt   string `json:"created_at"`
				} `json:"rows"`
			} `json:"users_list_by_tenant"`
		}

		if err := graphqlClient.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		table := format.TabularData{
			Data: respData.UsersListByTenant.Rows,
			Fields: []format.TableField{
				{Header: "ID", Field: "ID"},
				{Header: "Username", Field: "Username"},
				{Header: "Display Name", Field: "DisplayName"},
				{Header: "Status", Field: "Status"},
				{Header: "Created At", Field: "CreatedAt"},
			},
		}
		format.GetFormat().Print(table)

		return nil
	},
}

var authUsersGetCmd = &cobra.Command{
	Use:   "get <username>",
	Short: "Get details for a specific user",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		username := args[0]
		graphqlClient := client.NewClient()

		req := client.NewRequest(`
			query GetUser($username: String!) {
				users_list_by_tenant(where: {username: {_eq: $username}}) {
					rows {
						id
						username
						display_name
						status
						created_at
					}
				}
			}
		`)
		req.Var("username", username)

		var respData struct {
			UsersListByTenant struct {
				Rows []struct {
					ID          string `json:"id"`
					Username    string `json:"username"`
					DisplayName string `json:"display_name"`
					Status      string `json:"status"`
					CreatedAt   string `json:"created_at"`
				} `json:"rows"`
			} `json:"users_list_by_tenant"`
		}

		if err := graphqlClient.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		if len(respData.UsersListByTenant.Rows) == 0 {
			return fmt.Errorf("user %s not found", username)
		}

		table := format.TabularData{
			Data: respData.UsersListByTenant.Rows,
			Fields: []format.TableField{
				{Header: "ID", Field: "ID"},
				{Header: "Username", Field: "Username"},
				{Header: "Display Name", Field: "DisplayName"},
				{Header: "Status", Field: "Status"},
				{Header: "Created At", Field: "CreatedAt"},
			},
		}
		format.GetFormat().Print(table)

		return nil
	},
}

func init() {
	authCmd.AddCommand(authUsersCmd)
	authUsersCmd.AddCommand(authUsersListCmd)
	authUsersCmd.AddCommand(authUsersGetCmd)
}
