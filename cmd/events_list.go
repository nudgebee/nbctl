package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/machinebox/graphql"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"nudgebee.com/nbctl/pkg/client"
	"nudgebee.com/nbctl/pkg/format"
)

var eventsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List events",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := client.NewClient()

		accountId, _ := cmd.Flags().GetString("account-id")
		if accountId == "" {
			accountId = viper.GetString("account-id")
		}
		if accountId == "" {
			return fmt.Errorf("account-id is required")
		}

		startTimeStr, _ := cmd.Flags().GetString("start-time")
		endTimeStr, _ := cmd.Flags().GetString("end-time")
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")

		var startTime, endTime time.Time
		var err error

		if startTimeStr == "" {
			startTime = time.Now().Add(-24 * time.Hour)
		} else {
			startTime, err = time.Parse(time.RFC3339, startTimeStr)
			if err != nil {
				return err
			}
		}

		if endTimeStr == "" {
			endTime = time.Now()
		} else {
			endTime, err = time.Parse(time.RFC3339, endTimeStr)
			if err != nil {
				return err
			}
		}

		where := map[string]any{
			"account_id": map[string]any{"_eq": accountId},
			"_and": []map[string]any{
				{"created_at": map[string]any{"_gte": startTime.Format(time.RFC3339)}},
				{"created_at": map[string]any{"_lte": endTime.Format(time.RFC3339)}},
			},
		}

		if subject, _ := cmd.Flags().GetString("subject"); subject != "" {
			where["subject_name"] = map[string]any{"_ilike": "%" + subject + "%"}
		}
		if status, _ := cmd.Flags().GetString("status"); status != "" {
			where["status"] = map[string]any{"_eq": status}
		}
		if title, _ := cmd.Flags().GetString("title"); title != "" {
			where["title"] = map[string]any{"_ilike": "%" + title + "%"}
		}
		if priority, _ := cmd.Flags().GetString("priority"); priority != "" {
			where["priority"] = map[string]any{"_eq": priority}
		}

		req := graphql.NewRequest(`
			query list_k8_issues_data($where: EventsWhereRequest!, $limit: Int, $offset: Int) {
			  events: events_v2(where: $where, order_by: [{column: "starts_at", order: desc}], limit: $limit, offset: $offset) {
				rows{
				  account_id
				  subject_type
				  subject_name
				  subject_namespace
				  starts_at
				  title
				  finding_type
				  cluster
				  labels
				  status
				  aggregation_key
				  priority
				  id
				  resource_id
				  fingerprint
				}
			  }
			}
		`)

		req.Var("where", where)
		req.Var("limit", limit)
		req.Var("offset", offset)

		var respData struct {
			Events struct {
				Rows []struct {
					AccountID        string          `json:"account_id"`
					SubjectType      string          `json:"subject_type"`
					SubjectName      string          `json:"subject_name"`
					SubjectNamespace string          `json:"subject_namespace"`
					StartsAt         string          `json:"starts_at"`
					Title            string          `json:"title"`
					FindingType      string          `json:"finding_type"`
					Cluster          string          `json:"cluster"`
					Labels           json.RawMessage `json:"labels"`
					Status           string          `json:"status"`
					AggregationKey   string          `json:"aggregation_key"`
					Priority         string          `json:"priority"`
					ID               string          `json:"id"`
					ResourceID       string          `json:"resource_id"`
					Fingerprint      string          `json:"fingerprint"`
				} `json:"rows"`
			} `json:"events"`
		}

		if err := client.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		table := format.TabularData{
			Data: respData.Events.Rows,
			Fields: []format.TableField{
				{Header: "ID", Field: "ID"},
				{Header: "Title", Field: "Title"},
				{Header: "Priority", Field: "Priority"},
				{Header: "Status", Field: "Status"},
				{Header: "Subject", Field: "SubjectName"},
			},
		}
		format.GetFormat().Print(table)

		return nil
	},
}

func init() {
	eventsCmd.AddCommand(eventsListCmd)
}
