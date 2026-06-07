package cmd

import (
	"fmt"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var eventsListDuplicatesCmd = &cobra.Command{
	Use:   "list-duplicates [event-id]",
	Short: "List duplicate occurrences for an event",
	Long:  `Show the deduplication chain for an event — every occurrence the system has linked together via fingerprint matching.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		eventID := args[0]

		query := `
mutation EventGetDuplicates($event_id: String!) {
  event_get_duplicates(event_id: $event_id) {
    duplicates {
      event_id
      fingerprint
      occurrence_number
      first_event_id
      starts_at
    }
  }
}
`
		req := client.NewRequest(query)
		req.Var("event_id", eventID)

		var respData struct {
			EventGetDuplicates struct {
				Duplicates []struct {
					EventID          string `json:"event_id"`
					Fingerprint      string `json:"fingerprint"`
					OccurrenceNumber int    `json:"occurrence_number"`
					FirstEventID     string `json:"first_event_id"`
					StartsAt         string `json:"starts_at"`
				} `json:"duplicates"`
			} `json:"event_get_duplicates"`
		}

		c := client.NewClient()
		if err := c.Run(cmd.Context(), req, &respData); err != nil {
			return fmt.Errorf("failed to list event duplicates: %w", err)
		}

		if format.GetFormat().Get() == "json" {
			format.GetFormat().Print(respData.EventGetDuplicates.Duplicates)
			return nil
		}

		if len(respData.EventGetDuplicates.Duplicates) == 0 {
			_, _ = fmt.Fprintf(format.GetFormat().GetOutput(), "No duplicates found for event %s.\n", eventID)
			return nil
		}

		format.GetFormat().Print(format.TabularData{
			Data: respData.EventGetDuplicates.Duplicates,
			Fields: []format.TableField{
				{Header: "Occurrence #", Field: "OccurrenceNumber"},
				{Header: "Event ID", Field: "EventID"},
				{Header: "Starts At", Field: "StartsAt"},
				{Header: "First Event ID", Field: "FirstEventID"},
				{Header: "Fingerprint", Field: "Fingerprint"},
			},
		})

		return nil
	},
}

func init() {
	eventsCmd.AddCommand(eventsListDuplicatesCmd)
}
