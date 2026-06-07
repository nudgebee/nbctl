package cmd

import (
	"fmt"
	"slices"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var ticketCapableTypes = []string{"jira", "github", "gitlab", "servicenow", "pagerduty", "zenduty"}

var ticketsListConfigurationsCmd = &cobra.Command{
	Use:   "list-configurations",
	Short: "List ticket integration configurations",
	Long:  `List configured ticket integrations (Jira, ServiceNow, PagerDuty, GitHub, GitLab, Zenduty) for the current account.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		typeFilter, _ := cmd.Flags().GetString("type")
		statusFilter, _ := cmd.Flags().GetString("status")

		where := map[string]any{}
		if typeFilter != "" {
			if !slices.Contains(ticketCapableTypes, typeFilter) {
				return fmt.Errorf("invalid --type %q; must be one of %v", typeFilter, ticketCapableTypes)
			}
			where["type"] = map[string]any{"_eq": typeFilter}
		} else {
			where["type"] = map[string]any{"_in": ticketCapableTypes}
		}
		if statusFilter != "" {
			where["status"] = map[string]any{"_eq": statusFilter}
		}

		query := `
query ListTicketConfigurations($where: IntegrationWhereRequest!) {
  integrations: integrations_list(where: $where, order_by: [{column: "name", order: asc}]) {
    rows {
      id
      name
      type
      status
      created_by_display_name
      updated_at
    }
  }
}
`
		req := client.NewRequest(query)
		req.Var("where", where)

		var respData struct {
			Integrations struct {
				Rows []struct {
					ID                   string `json:"id"`
					Name                 string `json:"name"`
					Type                 string `json:"type"`
					Status               string `json:"status"`
					CreatedByDisplayName string `json:"created_by_display_name"`
					UpdatedAt            string `json:"updated_at"`
				} `json:"rows"`
			} `json:"integrations"`
		}

		c := client.NewClient()
		if err := c.Run(cmd.Context(), req, &respData); err != nil {
			return fmt.Errorf("failed to list ticket configurations: %w", err)
		}

		if format.GetFormat().Get() == "json" {
			format.GetFormat().Print(respData.Integrations.Rows)
			return nil
		}

		type row struct {
			ID        string
			Name      string
			Type      string
			Status    string
			CreatedBy string
			UpdatedAt string
		}
		var rows []row
		for _, r := range respData.Integrations.Rows {
			rows = append(rows, row{
				ID:        r.ID,
				Name:      r.Name,
				Type:      r.Type,
				Status:    r.Status,
				CreatedBy: r.CreatedByDisplayName,
				UpdatedAt: r.UpdatedAt,
			})
		}

		format.GetFormat().Print(format.TabularData{
			Data: rows,
			Fields: []format.TableField{
				{Header: "ID", Field: "ID"},
				{Header: "Name", Field: "Name"},
				{Header: "Type", Field: "Type"},
				{Header: "Status", Field: "Status"},
				{Header: "Created By", Field: "CreatedBy"},
				{Header: "Updated At", Field: "UpdatedAt"},
			},
		})

		return nil
	},
}

func init() {
	ticketsCmd.AddCommand(ticketsListConfigurationsCmd)
	ticketsListConfigurationsCmd.Flags().String("type", "", "Filter by a single integration type (jira, github, gitlab, servicenow, pagerduty, zenduty)")
	ticketsListConfigurationsCmd.Flags().String("status", "", "Filter by status (e.g. enabled)")
}
