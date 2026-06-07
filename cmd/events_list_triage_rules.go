package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var (
	eventsListTriageRulesType    string
	eventsListTriageRulesEnabled string
)

var eventsListTriageRulesCmd = &cobra.Command{
	Use:   "list-triage-rules",
	Short: "List event triage rules",
	Long:  `List event triage rules — both system rules and user-defined rules that classify or route incoming events.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		accountID, err := resolveAccountID(cmd)
		if err != nil {
			return err
		}

		vars := map[string]any{
			"cloud_account_id": accountID,
		}
		if eventsListTriageRulesType != "" {
			vars["rule_type"] = eventsListTriageRulesType
		}
		if eventsListTriageRulesEnabled != "" {
			if eventsListTriageRulesEnabled != "true" && eventsListTriageRulesEnabled != "false" {
				return fmt.Errorf("invalid value for --enabled: %q (must be true or false)", eventsListTriageRulesEnabled)
			}
			vars["enabled"] = eventsListTriageRulesEnabled == "true"
		}

		query := `
mutation EventGetTriageRules($cloud_account_id: String, $rule_type: String, $enabled: Boolean) {
  event_get_triage_rules(cloud_account_id: $cloud_account_id, rule_type: $rule_type, enabled: $enabled) {
    rules {
      id
      account_id
      name
      description
      rule_type
      action
      action_value
      priority
      enabled
      match_alertname
      match_namespace
      match_service
      match_source
      match_priority
      is_editable
      is_system_rule
      match_count
      last_matched_at
      created_at
      updated_at
    }
  }
}
`
		req := client.NewRequest(query)
		for k, v := range vars {
			req.Var(k, v)
		}

		var respData struct {
			EventGetTriageRules struct {
				Rules []struct {
					ID             string          `json:"id"`
					AccountID      string          `json:"account_id"`
					Name           string          `json:"name"`
					Description    string          `json:"description"`
					RuleType       string          `json:"rule_type"`
					Action         string          `json:"action"`
					ActionValue    string          `json:"action_value"`
					Priority       json.Number     `json:"priority"`
					Enabled        bool            `json:"enabled"`
					MatchAlertname string          `json:"match_alertname"`
					MatchNamespace string          `json:"match_namespace"`
					MatchService   string          `json:"match_service"`
					MatchSource    string          `json:"match_source"`
					MatchPriority  string          `json:"match_priority"`
					IsEditable     bool            `json:"is_editable"`
					IsSystemRule   bool            `json:"is_system_rule"`
					MatchCount     int             `json:"match_count"`
					LastMatchedAt  *string         `json:"last_matched_at"`
					CreatedAt      string      `json:"created_at"`
					UpdatedAt      string      `json:"updated_at"`
				} `json:"rules"`
			} `json:"event_get_triage_rules"`
		}

		c := client.NewClient()
		if err := c.Run(cmd.Context(), req, &respData); err != nil {
			return fmt.Errorf("failed to list triage rules: %w", err)
		}

		if format.GetFormat().Get() == "json" {
			format.GetFormat().Print(respData.EventGetTriageRules.Rules)
			return nil
		}

		type row struct {
			ID            string
			Name          string
			RuleType      string
			Action        string
			Priority      string
			Enabled       bool
			MatchCount    int
			LastMatchedAt string
			IsSystem      bool
		}
		var rows []row
		for _, r := range respData.EventGetTriageRules.Rules {
			last := "N/A"
			if r.LastMatchedAt != nil {
				last = *r.LastMatchedAt
			}
			rows = append(rows, row{
				ID:            r.ID,
				Name:          r.Name,
				RuleType:      r.RuleType,
				Action:        r.Action,
				Priority:      r.Priority.String(),
				Enabled:       r.Enabled,
				MatchCount:    r.MatchCount,
				LastMatchedAt: last,
				IsSystem:      r.IsSystemRule,
			})
		}

		format.GetFormat().Print(format.TabularData{
			Data: rows,
			Fields: []format.TableField{
				{Header: "ID", Field: "ID"},
				{Header: "Name", Field: "Name"},
				{Header: "Type", Field: "RuleType"},
				{Header: "Action", Field: "Action"},
				{Header: "Priority", Field: "Priority"},
				{Header: "Enabled", Field: "Enabled"},
				{Header: "Matches", Field: "MatchCount"},
				{Header: "Last Matched", Field: "LastMatchedAt"},
				{Header: "System", Field: "IsSystem"},
			},
		})

		return nil
	},
}

func init() {
	eventsCmd.AddCommand(eventsListTriageRulesCmd)
	eventsListTriageRulesCmd.Flags().String("account-id", "", "Account ID (overrides profile)")
	eventsListTriageRulesCmd.Flags().StringVar(&eventsListTriageRulesType, "rule-type", "", "Filter by rule type")
	eventsListTriageRulesCmd.Flags().StringVar(&eventsListTriageRulesEnabled, "enabled", "", "Filter by enabled status (true|false)")
}
