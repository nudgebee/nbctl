package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

// peelJSONString returns the inner JSON payload when the server has
// double-encoded a jsonb field as a JSON-quoted string ("{...}"); otherwise
// it returns the input unchanged.
func peelJSONString(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 || raw[0] != '"' {
		return raw
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return raw
	}
	return json.RawMessage(s)
}

var eventsGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get an event by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		graphqlClient := client.NewClient()

		req := client.NewRequest(`
			query ResolveEventRecord($where: EventsWhereRequest!) {
			  events: events_v2(where: $where, limit: 1, offset: 0) {
			    rows {
				evidences
				id
				subject_name
				starts_at
				subject_namespace
				cluster
				subject_node
				subject_owner
				subject_type
				title
				priority
				description
				aggregation_key
				service_key
				resource_id
				account_id
				status
				fingerprint
				source
				labels
			    }
			  }
			}
		`)

		req.Var("where", map[string]any{
			"id": map[string]any{"_eq": args[0]},
		})

		var respData struct {
			Events struct {
				Rows []struct {
					Evidences        json.RawMessage `json:"evidences"`
					ID               string          `json:"id"`
					SubjectName      string          `json:"subject_name"`
					StartsAt         string          `json:"starts_at"`
					SubjectNamespace string          `json:"subject_namespace"`
					Cluster          string          `json:"cluster"`
					SubjectNode      string          `json:"subject_node"`
					SubjectOwner     string          `json:"subject_owner"`
					SubjectType      string          `json:"subject_type"`
					Title            string          `json:"title"`
					Priority         string          `json:"priority"`
					Description      string          `json:"description"`
					AggregationKey   string          `json:"aggregation_key"`
					ServiceKey       string          `json:"service_key"`
					ResourceID       string          `json:"resource_id"`
					AccountID        string          `json:"account_id"`
					Status           string          `json:"status"`
					Fingerprint      string          `json:"fingerprint"`
					Source           string          `json:"source"`
					Labels           json.RawMessage `json:"labels"`
				} `json:"rows"`
			} `json:"events"`
		}

		if err := graphqlClient.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		if len(respData.Events.Rows) == 0 {
			fmt.Println("Event not found.")
			return nil
		}

		row := respData.Events.Rows[0]
		row.Evidences = peelJSONString(row.Evidences)
		row.Labels = peelJSONString(row.Labels)
		format.GetFormat().Print(row)

		return nil
	},
}

func init() {
	eventsCmd.AddCommand(eventsGetCmd)
}
