package cmd

import (
	"strings"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var authGroupsCmd = &cobra.Command{
	Use:   "groups",
	Short: "Manage tenant user groups",
}

var authGroupsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List user groups with assigned roles and member count",
	RunE: func(cmd *cobra.Command, args []string) error {
		graphqlClient := client.NewClient()

		req := client.NewRequest(`
			query ListUserGroups {
				usergroups_list {
					rows {
						id
						name
						description
						roles
						user_count
						created_at
					}
				}
			}
		`)

		var respData struct {
			UsergroupsList struct {
				Rows []struct {
					ID          string   `json:"id"`
					Name        string   `json:"name"`
					Description string   `json:"description"`
					Roles       []string `json:"roles"`
					UserCount   int      `json:"user_count"`
					CreatedAt   string   `json:"created_at"`
				} `json:"rows"`
			} `json:"usergroups_list"`
		}

		if err := graphqlClient.Run(cmd.Context(), req, &respData); err != nil {
			return err
		}

		type groupRow struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			Roles       string `json:"roles"`
			UserCount   int    `json:"user_count"`
			CreatedAt   string `json:"created_at"`
		}
		var rows []groupRow
		for _, r := range respData.UsergroupsList.Rows {
			rolesStr := "-"
			if len(r.Roles) > 0 {
				rolesStr = strings.Join(r.Roles, ", ")
			}
			rows = append(rows, groupRow{
				ID:          r.ID,
				Name:        r.Name,
				Description: r.Description,
				Roles:       rolesStr,
				UserCount:   r.UserCount,
				CreatedAt:   r.CreatedAt,
			})
		}

		table := format.TabularData{
			Data: rows,
			Fields: []format.TableField{
				{Header: "ID", Field: "ID"},
				{Header: "Group Name", Field: "Name"},
				{Header: "Description", Field: "Description"},
				{Header: "Assigned Roles", Field: "Roles"},
				{Header: "Users", Field: "UserCount"},
				{Header: "Created At", Field: "CreatedAt"},
			},
		}
		format.GetFormat().Print(table)

		return nil
	},
}

var authGroupsCreateCmd = &cobra.Command{
	Use:   "create <group-name>",
	Short: "Create a new user group",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		groupName := args[0]
		desc, _ := cmd.Flags().GetString("description")

		graphqlClient := client.NewClient()

		req := client.NewRequest(`
			mutation CreateUserGroup($request: UserGroupCreateInput!) {
				usergroup_create(request: $request) {
					id
					name
					status
				}
			}
		`)
		req.Var("request", map[string]any{
			"name":        groupName,
			"description": desc,
		})

		var respData struct {
			UsergroupCreate struct {
				ID     string `json:"id"`
				Name   string `json:"name"`
				Status string `json:"status"`
			} `json:"usergroup_create"`
		}

		if err := graphqlClient.Run(cmd.Context(), req, &respData); err != nil {
			return err
		}

		format.GetFormat().Print(respData.UsergroupCreate)
		return nil
	},
}

func init() {
	authCmd.AddCommand(authGroupsCmd)
	authGroupsCmd.AddCommand(authGroupsListCmd)
	authGroupsCmd.AddCommand(authGroupsCreateCmd)

	authGroupsCreateCmd.Flags().String("description", "", "Description of the user group")
}
