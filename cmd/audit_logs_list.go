package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var auditLogsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List audit events",
	Long:  `List audit events for the current account, ordered by most recent first.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		accountID, err := resolveAccountID(cmd)
		if err != nil {
			return err
		}

		username, _ := cmd.Flags().GetString("username")
		category, _ := cmd.Flags().GetString("category")
		eventType, _ := cmd.Flags().GetString("type")
		action, _ := cmd.Flags().GetString("action")
		status, _ := cmd.Flags().GetString("status")
		startTime, _ := cmd.Flags().GetString("start-time")
		endTime, _ := cmd.Flags().GetString("end-time")
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")

		where := map[string]any{
			"account_id": map[string]any{"_eq": accountID},
		}
		if username != "" {
			where["username"] = map[string]any{"_eq": username}
		}
		if category != "" {
			where["event_category"] = map[string]any{"_eq": category}
		}
		if eventType != "" {
			where["event_type"] = map[string]any{"_eq": eventType}
		}
		if action != "" {
			where["event_action"] = map[string]any{"_eq": action}
		}
		if status != "" {
			where["event_status"] = map[string]any{"_eq": status}
		}
		if startTime != "" || endTime != "" {
			timeFilter := map[string]any{}
			if startTime != "" {
				timeFilter["_gt"] = startTime
			}
			if endTime != "" {
				timeFilter["_lt"] = endTime
			}
			where["event_time"] = map[string]any{"_between": timeFilter}
		}

		query := `
query ListAuditEvents($where: AuditEventWhereRequest!, $limit: Int, $offset: Int) {
  audits_v2(where: $where, limit: $limit, offset: $offset, order_by: [{column: "event_time", order: desc}]) {
    rows {
      user_id
      account_id
      event_time
      event_category
      event_type
      event_status
      event_target
      event_action
      transaction_id
      event_attr
    }
  }
}
`
		req := client.NewRequest(query)
		req.Var("where", where)
		req.Var("limit", limit)
		req.Var("offset", offset)

		var respData struct {
			AuditsV2 struct {
				Rows []struct {
					UserID        string          `json:"user_id"`
					AccountID     string          `json:"account_id"`
					EventTime     string          `json:"event_time"`
					EventCategory string          `json:"event_category"`
					EventType     string          `json:"event_type"`
					EventStatus   string          `json:"event_status"`
					EventTarget   string          `json:"event_target"`
					EventAction   string          `json:"event_action"`
					TransactionID string          `json:"transaction_id"`
					EventAttr     json.RawMessage `json:"event_attr"`
				} `json:"rows"`
			} `json:"audits_v2"`
		}

		c := client.NewClient()
		if err := c.Run(cmd.Context(), req, &respData); err != nil {
			return fmt.Errorf("failed to list audit events: %w", err)
		}

		if format.GetFormat().Get() == "json" {
			format.GetFormat().Print(respData.AuditsV2.Rows)
			return nil
		}

		type row struct {
			EventTime     string
			UserID        string
			EventCategory string
			EventType     string
			EventAction   string
			EventStatus   string
			EventTarget   string
		}
		var rows []row
		for _, r := range respData.AuditsV2.Rows {
			rows = append(rows, row{
				EventTime:     r.EventTime,
				UserID:        r.UserID,
				EventCategory: r.EventCategory,
				EventType:     r.EventType,
				EventAction:   r.EventAction,
				EventStatus:   r.EventStatus,
				EventTarget:   r.EventTarget,
			})
		}

		format.GetFormat().Print(format.TabularData{
			Data: rows,
			Fields: []format.TableField{
				{Header: "Event Time", Field: "EventTime"},
				{Header: "User ID", Field: "UserID"},
				{Header: "Category", Field: "EventCategory"},
				{Header: "Type", Field: "EventType"},
				{Header: "Action", Field: "EventAction"},
				{Header: "Status", Field: "EventStatus"},
				{Header: "Target", Field: "EventTarget"},
			},
		})

		return nil
	},
}

func init() {
	auditLogsCmd.AddCommand(auditLogsListCmd)
	auditLogsListCmd.Flags().String("account-id", "", "Account ID (overrides profile)")
	auditLogsListCmd.Flags().String("username", "", "Filter by username")
	auditLogsListCmd.Flags().String("category", "", "Filter by event category")
	auditLogsListCmd.Flags().String("type", "", "Filter by event type")
	auditLogsListCmd.Flags().String("action", "", "Filter by event action")
	auditLogsListCmd.Flags().String("status", "", "Filter by event status")
	auditLogsListCmd.Flags().String("start-time", "", "Filter events after this RFC3339 timestamp")
	auditLogsListCmd.Flags().String("end-time", "", "Filter events before this RFC3339 timestamp")
	auditLogsListCmd.Flags().Int("limit", 25, "Maximum number of events to return")
	auditLogsListCmd.Flags().Int("offset", 0, "Pagination offset")
}
