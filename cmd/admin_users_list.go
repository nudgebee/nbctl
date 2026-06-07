package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var adminUsersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all users",
	RunE: func(cmd *cobra.Command, args []string) error {
		graphqlClient := client.NewClient()

		offset, _ := cmd.Flags().GetInt("offset")
		limit, _ := cmd.Flags().GetInt("limit")
		name, _ := cmd.Flags().GetString("name")
		username, _ := cmd.Flags().GetString("username")
		status, _ := cmd.Flags().GetString("status")
		id, _ := cmd.Flags().GetString("id")

		req := client.NewRequest(`
			query GetUsersByTenant($offset: Int, $limit: Int, $where: UsersByTenantWhereRequest!) {
				users: users_list_by_tenant(where: $where, limit: $limit, offset: $offset, order_by: [{column: "username", order: asc}]) {
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

		type row struct {
			ID             string
			DisplayName    string
			Username       string
			Status         string
			CreatedAt      string
			LastAccessedAt string
			Roles          string
			Groups         string
		}
		var rows []row
		for _, r := range respData.Users.Rows {
			rows = append(rows, row{
				ID:             r.ID,
				DisplayName:    r.DisplayName,
				Username:       r.Username,
				Status:         r.Status,
				CreatedAt:      r.CreatedAt,
				LastAccessedAt: r.LastAccessedAt,
				Roles:          joinJSONNames(r.UserRoles, "role"),
				Groups:         joinJSONNames(r.UserGroups, "name"),
			})
		}

		table := format.TabularData{
			Data: rows,
			Fields: []format.TableField{
				{Header: "ID", Field: "ID"},
				{Header: "Display Name", Field: "DisplayName"},
				{Header: "Username", Field: "Username"},
				{Header: "Status", Field: "Status"},
				{Header: "Roles", Field: "Roles"},
				{Header: "Groups", Field: "Groups"},
				{Header: "Last Login", Field: "LastAccessedAt"},
			},
		}
		format.GetFormat().Print(table)

		return nil
	},
}

func init() {
	adminUsersCmd.AddCommand(adminUsersListCmd)
	adminUsersListCmd.Flags().Int("offset", 0, "Offset for pagination")
	adminUsersListCmd.Flags().Int("limit", 20, "Limit for pagination")
	adminUsersListCmd.Flags().String("name", "", "Filter by name (ilike)")
	adminUsersListCmd.Flags().String("username", "", "Filter by username (ilike)")
	adminUsersListCmd.Flags().String("status", "", "Filter by status (eq)")
	adminUsersListCmd.Flags().String("id", "", "Filter by id (eq)")
}

func joinJSONNames(raw json.RawMessage, key string) string {
	if len(raw) == 0 {
		return ""
	}
	payload := []byte(raw)
	// jsonb fields are sometimes returned as a JSON-encoded string; peel one layer.
	if len(payload) > 0 && payload[0] == '"' {
		var s string
		if err := json.Unmarshal(payload, &s); err == nil {
			payload = []byte(s)
		}
	}
	var items []map[string]any
	if err := json.Unmarshal(payload, &items); err != nil {
		return ""
	}
	var names []string
	for _, it := range items {
		if v, ok := it[key]; ok {
			if s, ok := v.(string); ok && s != "" {
				names = append(names, s)
			}
		}
	}
	return strings.Join(names, ",")
}
