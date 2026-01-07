package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var eventsGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get an event by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		graphqlClient := client.NewClient()

		req := client.NewRequest(`
			query ResolveEventRecord($id:uuid!) {
			  events(where: {id: {_eq:$id}}) {
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
				cloud_resource_id
				cloud_account_id
				status
				fingerprint
				source
				labels
			  }
			}
		`)

		req.Var("id", args[0])

		var respData struct {
			Events []struct {
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
				CloudResourceID  string          `json:"cloud_resource_id"`
				CloudAccountID   string          `json:"cloud_account_id"`
				Status           string          `json:"status"`
				Fingerprint      string          `json:"fingerprint"`
				Source           string          `json:"source"`
				Labels           json.RawMessage `json:"labels"`
			} `json:"events"`
		}

		if err := graphqlClient.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		if len(respData.Events) == 0 {
			fmt.Println("Event not found.")
			return nil
		}

		format.GetFormat().Print(respData.Events[0])

		return nil
	},
}

func init() {
	eventsCmd.AddCommand(eventsGetCmd)
}
