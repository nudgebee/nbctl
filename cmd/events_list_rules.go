package cmd

import (
	"fmt"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var eventsListRulesCmd = &cobra.Command{
	Use:   "list-rules",
	Short: "List event-manager rules",
	Long:  `List Prometheus / event-manager rules that determine which conditions raise an event.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		accountID, err := resolveAccountID(cmd)
		if err != nil {
			return err
		}

		name, _ := cmd.Flags().GetString("name")
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")

		where := map[string]any{
			"account_id": map[string]any{"_eq": accountID},
		}
		if name != "" {
			where["alert"] = map[string]any{"_ilike": "%" + name + "%"}
		}

		query := `
query ListEventRules($where: EventRuleWhereRequest!, $limit: Int, $offset: Int) {
  event_rules_v2(where: $where, order_by: [{column: "updated_at", order: desc}], limit: $limit, offset: $offset) {
    rows {
      id
      account_id
      alert
      group
      namespace
      severity
      category
      source
      enabled
      is_editable
      expr
      duration
      created_at
      updated_at
    }
  }
}
`
		req := client.NewRequest(query)
		req.Var("where", where)
		req.Var("limit", limit)
		req.Var("offset", offset)

		var respData struct {
			EventRulesV2 struct {
				Rows []struct {
					ID         string `json:"id"`
					AccountID  string `json:"account_id"`
					Alert      string `json:"alert"`
					Group      string `json:"group"`
					Namespace  string `json:"namespace"`
					Severity   string `json:"severity"`
					Category   string `json:"category"`
					Source     string `json:"source"`
					Enabled    bool   `json:"enabled"`
					IsEditable bool   `json:"is_editable"`
					Expr       string `json:"expr"`
					Duration   string `json:"duration"`
					CreatedAt  string `json:"created_at"`
					UpdatedAt  string `json:"updated_at"`
				} `json:"rows"`
			} `json:"event_rules_v2"`
		}

		c := client.NewClient()
		if err := c.Run(cmd.Context(), req, &respData); err != nil {
			return fmt.Errorf("failed to list event rules: %w", err)
		}

		if format.GetFormat().Get() == "json" {
			format.GetFormat().Print(respData.EventRulesV2.Rows)
			return nil
		}

		format.GetFormat().Print(format.TabularData{
			Data: respData.EventRulesV2.Rows,
			Fields: []format.TableField{
				{Header: "ID", Field: "ID"},
				{Header: "Alert", Field: "Alert"},
				{Header: "Severity", Field: "Severity"},
				{Header: "Category", Field: "Category"},
				{Header: "Source", Field: "Source"},
				{Header: "Group", Field: "Group"},
				{Header: "Enabled", Field: "Enabled"},
				{Header: "Updated At", Field: "UpdatedAt"},
			},
		})

		return nil
	},
}

func init() {
	eventsCmd.AddCommand(eventsListRulesCmd)
	eventsListRulesCmd.Flags().String("account-id", "", "Account ID (overrides profile)")
	eventsListRulesCmd.Flags().String("name", "", "Filter by alert name (ilike)")
	eventsListRulesCmd.Flags().Int("limit", 25, "Maximum number of rules to return")
	eventsListRulesCmd.Flags().Int("offset", 0, "Pagination offset")
}
