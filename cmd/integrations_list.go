package cmd

import (
	"context"
	"fmt"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var (
	integrationsListStatus string
	integrationsListType   string
	integrationsListName   string
	integrationsListLimit  int
	integrationsListOffset int
)

var integrationsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List integrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		graphqlClient := client.NewClient()

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
		if integrationsListStatus != "" {
			where["status"] = map[string]any{"_eq": integrationsListStatus}
		}
		if integrationsListType != "" {
			where["type"] = map[string]any{"_eq": integrationsListType}
		}
		if integrationsListName != "" {
			where["name"] = map[string]any{"_ilike": fmt.Sprintf("%%%s%%", integrationsListName)}
		}

		req.Var("where", where)
		req.Var("limit", integrationsListLimit)
		req.Var("offset", integrationsListOffset)

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
	integrationsListCmd.Flags().StringVar(&integrationsListStatus, "status", "", "Filter by status (eq)")
	integrationsListCmd.Flags().StringVar(&integrationsListType, "type", "", "Filter by type (eq)")
	integrationsListCmd.Flags().StringVar(&integrationsListName, "name", "", "Filter by name (ilike)")
	integrationsListCmd.Flags().IntVar(&integrationsListLimit, "limit", 20, "Limit for pagination")
	integrationsListCmd.Flags().IntVar(&integrationsListOffset, "offset", 0, "Offset for pagination")
}
