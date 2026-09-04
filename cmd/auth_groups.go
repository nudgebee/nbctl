package cmd

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var authGroupsCmd = &cobra.Command{
	Use:   "groups",
	Short: "Manage tenant user groups",
}

type groupRoleItem struct {
	Role       string `json:"role"`
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
}

type groupRolesField []groupRoleItem

func (g *groupRolesField) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || string(trimmed) == "null" || string(trimmed) == `""` {
		*g = nil
		return nil
	}

	var items []groupRoleItem
	err := json.Unmarshal(trimmed, &items)
	if err == nil {
		*g = items
		return nil
	}

	var str string
	if errStr := json.Unmarshal(trimmed, &str); errStr != nil {
		return err
	}

	str = strings.TrimSpace(str)
	if str == "" || str == "null" {
		*g = nil
		return nil
	}

	if errArr := json.Unmarshal([]byte(str), &items); errArr != nil {
		return errArr
	}

	*g = items
	return nil
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
						group_roles
						member_count
						created_at
					}
				}
			}
		`)

		var respData struct {
			UsergroupsList struct {
				Rows []struct {
					ID          string          `json:"id"`
					Name        string          `json:"name"`
					Description string          `json:"description"`
					GroupRoles  groupRolesField `json:"group_roles"`
					MemberCount int             `json:"member_count"`
					CreatedAt   string          `json:"created_at"`
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
			var roles []string
			for _, item := range r.GroupRoles {
				if item.Role != "" {
					roles = append(roles, item.Role)
				}
			}

			rolesStr := "-"
			if len(roles) > 0 {
				rolesStr = strings.Join(roles, ", ")
			}
			rows = append(rows, groupRow{
				ID:          r.ID,
				Name:        r.Name,
				Description: r.Description,
				Roles:       rolesStr,
				UserCount:   r.MemberCount,
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
			mutation CreateUserGroup($name: String!, $description: String) {
				usergroup_create(name: $name, description: $description) {
					id
				}
			}
		`)
		req.Var("name", groupName)
		if desc != "" {
			req.Var("description", desc)
		} else {
			req.Var("description", nil)
		}

		var respData struct {
			UsergroupCreate struct {
				ID string `json:"id"`
			} `json:"usergroup_create"`
		}

		if err := graphqlClient.Run(cmd.Context(), req, &respData); err != nil {
			return err
		}

		format.GetFormat().Print(struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Status string `json:"status"`
		}{
			ID:     respData.UsergroupCreate.ID,
			Name:   groupName,
			Status: "created",
		})
		return nil
	},
}

func init() {
	authCmd.AddCommand(authGroupsCmd)
	authGroupsCmd.AddCommand(authGroupsListCmd)
	authGroupsCmd.AddCommand(authGroupsCreateCmd)

	authGroupsCreateCmd.Flags().String("description", "", "Description of the user group")
}
