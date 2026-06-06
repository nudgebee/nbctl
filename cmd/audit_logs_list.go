package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	auditLogsListUsername      string
	auditLogsListEventCategory string
	auditLogsListEventType     string
	auditLogsListEventAction   string
	auditLogsListEventStatus   string
	auditLogsListStartTime     string
	auditLogsListEndTime       string
	auditLogsListLimit         int
	auditLogsListOffset        int
)

var auditLogsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List audit events",
	Long:  `List audit events for the current account, ordered by most recent first.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		accountID, _ := cmd.Flags().GetString("account-id")
		if accountID == "" {
			accountID = viper.GetString("account-id")
		}
		if accountID == "" {
			return fmt.Errorf("account-id is required; set it in your profile with `nbctl configure add` or pass --account-id")
		}

		where := map[string]any{
			"account_id": map[string]any{"_eq": accountID},
		}
		if auditLogsListUsername != "" {
			where["username"] = map[string]any{"_eq": auditLogsListUsername}
		}
		if auditLogsListEventCategory != "" {
			where["event_category"] = map[string]any{"_eq": auditLogsListEventCategory}
		}
		if auditLogsListEventType != "" {
			where["event_type"] = map[string]any{"_eq": auditLogsListEventType}
		}
		if auditLogsListEventAction != "" {
			where["event_action"] = map[string]any{"_eq": auditLogsListEventAction}
		}
		if auditLogsListEventStatus != "" {
			where["event_status"] = map[string]any{"_eq": auditLogsListEventStatus}
		}
		if auditLogsListStartTime != "" || auditLogsListEndTime != "" {
			timeFilter := map[string]any{}
			if auditLogsListStartTime != "" {
				timeFilter["_gt"] = auditLogsListStartTime
			}
			if auditLogsListEndTime != "" {
				timeFilter["_lt"] = auditLogsListEndTime
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
		req.Var("limit", auditLogsListLimit)
		req.Var("offset", auditLogsListOffset)

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
	auditLogsListCmd.Flags().StringVar(&auditLogsListUsername, "username", "", "Filter by username")
	auditLogsListCmd.Flags().StringVar(&auditLogsListEventCategory, "category", "", "Filter by event category")
	auditLogsListCmd.Flags().StringVar(&auditLogsListEventType, "type", "", "Filter by event type")
	auditLogsListCmd.Flags().StringVar(&auditLogsListEventAction, "action", "", "Filter by event action")
	auditLogsListCmd.Flags().StringVar(&auditLogsListEventStatus, "status", "", "Filter by event status")
	auditLogsListCmd.Flags().StringVar(&auditLogsListStartTime, "start-time", "", "Filter events after this RFC3339 timestamp")
	auditLogsListCmd.Flags().StringVar(&auditLogsListEndTime, "end-time", "", "Filter events before this RFC3339 timestamp")
	auditLogsListCmd.Flags().IntVar(&auditLogsListLimit, "limit", 25, "Maximum number of events to return")
	auditLogsListCmd.Flags().IntVar(&auditLogsListOffset, "offset", 0, "Pagination offset")
}
