package cmd

import (
	"fmt"
	"strings"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var authRolesCmd = &cobra.Command{
	Use:   "roles",
	Short: "Manage built-in and custom roles",
}

var authRolesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List built-in and custom roles",
	RunE: func(cmd *cobra.Command, args []string) error {
		graphqlClient := client.NewClient()

		req := client.NewRequest(`
			query ListRoles {
				roles_list {
					display_name
					value
				}
				customroles_list {
					roles {
						id
						name
						description
					}
				}
			}
		`)

		var respData struct {
			RolesList []struct {
				DisplayName string `json:"display_name"`
				Value       string `json:"value"`
			} `json:"roles_list"`
			CustomrolesList struct {
				Roles []struct {
					ID          string `json:"id"`
					Name        string `json:"name"`
					Description string `json:"description"`
				} `json:"roles"`
			} `json:"customroles_list"`
		}

		if err := graphqlClient.Run(cmd.Context(), req, &respData); err != nil {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Warning: failed to fetch custom roles, falling back to built-in roles:", err)
			reqFallback := client.NewRequest(`
				query ListBuiltInRoles {
					roles_list {
						display_name
						value
					}
				}
			`)
			if errFallback := graphqlClient.Run(cmd.Context(), reqFallback, &respData); errFallback != nil {
				return fmt.Errorf("primary query failed (%v), fallback query failed: %w", err, errFallback)
			}
		}

		type roleRow struct {
			Type        string `json:"type"`
			Name        string `json:"name"`
			RoleKey     string `json:"role_key"`
			Description string `json:"description"`
		}
		var rows []roleRow

		for _, r := range respData.RolesList {
			rows = append(rows, roleRow{
				Type:        "Built-in",
				Name:        r.DisplayName,
				RoleKey:     r.Value,
				Description: "-",
			})
		}
		for _, cr := range respData.CustomrolesList.Roles {
			rows = append(rows, roleRow{
				Type:        "Custom",
				Name:        cr.Name,
				RoleKey:     cr.ID,
				Description: cr.Description,
			})
		}

		table := format.TabularData{
			Data: rows,
			Fields: []format.TableField{
				{Header: "Type", Field: "Type"},
				{Header: "Display Name", Field: "Name"},
				{Header: "Role Key", Field: "RoleKey"},
				{Header: "Description", Field: "Description"},
			},
		}
		format.GetFormat().Print(table)

		return nil
	},
}

var authRolesCreateCmd = &cobra.Command{
	Use:   "create <role-name>",
	Short: "Create a custom role",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		roleName := args[0]
		desc, _ := cmd.Flags().GetString("description")
		permissions, _ := cmd.Flags().GetStringArray("permission")

		graphqlClient := client.NewClient()

		type customRolePermissionInput struct {
			Module string `json:"module"`
			Class  string `json:"class,omitempty"`
		}

		var permInputs []customRolePermissionInput
		for _, p := range permissions {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			parts := strings.SplitN(p, ":", 2)
			if len(parts) == 2 {
				module := strings.TrimSpace(parts[0])
				class := strings.TrimSpace(parts[1])
				if module == "" {
					continue
				}
				permInputs = append(permInputs, customRolePermissionInput{
					Module: module,
					Class:  class,
				})
			} else {
				permInputs = append(permInputs, customRolePermissionInput{
					Module: p,
				})
			}
		}

		req := client.NewRequest(`
			mutation CreateCustomRole($name: String!, $description: String, $permissions: [CustomRolePermissionInput!]) {
				customroles_create(name: $name, description: $description, permissions: $permissions) {
					id
				}
			}
		`)
		req.Var("name", roleName)
		if desc != "" {
			req.Var("description", desc)
		}
		if len(permInputs) > 0 {
			req.Var("permissions", permInputs)
		}

		var respData struct {
			CustomrolesCreate struct {
				ID string `json:"id"`
			} `json:"customroles_create"`
		}

		if err := graphqlClient.Run(cmd.Context(), req, &respData); err != nil {
			return err
		}

		format.GetFormat().Print(struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Status string `json:"status"`
		}{
			ID:     respData.CustomrolesCreate.ID,
			Name:   roleName,
			Status: "created",
		})
		return nil
	},
}

var authRolesDeleteCmd = &cobra.Command{
	Use:   "delete <role-id>",
	Short: "Delete a custom role",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		roleID := args[0]

		graphqlClient := client.NewClient()

		req := client.NewRequest(`
			mutation DeleteCustomRole($id: String!) {
				customroles_delete(id: $id) {
					status
					message
				}
			}
		`)
		req.Var("id", roleID)

		var respData struct {
			CustomrolesDelete struct {
				Status  string `json:"status"`
				Message string `json:"message"`
			} `json:"customroles_delete"`
		}

		if err := graphqlClient.Run(cmd.Context(), req, &respData); err != nil {
			return err
		}

		format.GetFormat().Print(struct {
			RoleID  string `json:"role_id"`
			Status  string `json:"status"`
			Message string `json:"message,omitempty"`
		}{
			RoleID:  roleID,
			Status:  respData.CustomrolesDelete.Status,
			Message: respData.CustomrolesDelete.Message,
		})
		return nil
	},
}

func init() {
	authCmd.AddCommand(authRolesCmd)
	authRolesCmd.AddCommand(authRolesListCmd)
	authRolesCmd.AddCommand(authRolesCreateCmd)
	authRolesCmd.AddCommand(authRolesDeleteCmd)

	authRolesCreateCmd.Flags().String("description", "", "Description of the custom role")
	authRolesCreateCmd.Flags().StringArray("permission", []string{}, "Permission key to assign (can specify multiple)")
}
