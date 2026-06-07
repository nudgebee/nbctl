package cmd

import (
	"context"
	"fmt"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var integrationsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List integrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		graphqlClient := client.NewClient()

		status, _ := cmd.Flags().GetString("status")
		integrationType, _ := cmd.Flags().GetString("type")
		name, _ := cmd.Flags().GetString("name")
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")

		req := client.NewRequest(`
			query ListIntegrations($where: IntegrationWhereRequest!, $limit: Int, $offset: Int) {
				integrations: integrations_list(where: $where, limit: $limit, offset: $offset, order_by: [{column: "name", order: asc}]) {
					rows {
						id
						name
						type
						source
						status
						created_by_display_name
						updated_at
					}
				}
			}
		`)

		where := make(map[string]any)
		if status != "" {
			where["status"] = map[string]any{"_eq": status}
		}
		if integrationType != "" {
			where["type"] = map[string]any{"_eq": integrationType}
		}
		if name != "" {
			where["name"] = map[string]any{"_ilike": fmt.Sprintf("%%%s%%", name)}
		}

		req.Var("where", where)
		req.Var("limit", limit)
		req.Var("offset", offset)

		var respData struct {
			Integrations struct {
				Rows []struct {
					ID                   string `json:"id"`
					Name                 string `json:"name"`
					Type                 string `json:"type"`
					Source               string `json:"source"`
					Status               string `json:"status"`
					CreatedByDisplayName string `json:"created_by_display_name"`
					UpdatedAt            string `json:"updated_at"`
				} `json:"rows"`
			} `json:"integrations"`
		}

		if err := graphqlClient.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		table := format.TabularData{
			Data: respData.Integrations.Rows,
			Fields: []format.TableField{
				{Header: "ID", Field: "ID"},
				{Header: "Name", Field: "Name"},
				{Header: "Type", Field: "Type"},
				{Header: "Source", Field: "Source"},
				{Header: "Status", Field: "Status"},
				{Header: "Created By", Field: "CreatedByDisplayName"},
				{Header: "Updated At", Field: "UpdatedAt"},
			},
		}
		format.GetFormat().Print(table)

		return nil
	},
}

func init() {
	integrationsCmd.AddCommand(integrationsListCmd)
	integrationsListCmd.Flags().String("status", "", "Filter by status (eq)")
	integrationsListCmd.Flags().String("type", "", "Filter by type (eq)")
	integrationsListCmd.Flags().String("name", "", "Filter by name (ilike)")
	integrationsListCmd.Flags().Int("limit", 20, "Limit for pagination")
	integrationsListCmd.Flags().Int("offset", 0, "Offset for pagination")
}
