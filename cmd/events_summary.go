package cmd

import (
	"context"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var eventsSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Show summary of notification rules and event triggers by severity",
	RunE: func(cmd *cobra.Command, args []string) error {
		graphqlClient := client.NewClient()

		req := client.NewRequest(`
			query AggregateEventRules {
				notifications_aggregate_rules {
					rows {
						severity
						source
						count
					}
				}
			}
		`)

		var respData struct {
			NotificationsAggregateRules struct {
				Rows []struct {
					Severity string `json:"severity"`
					Source   string `json:"source"`
					Count    int    `json:"count"`
				} `json:"rows"`
			} `json:"notifications_aggregate_rules"`
		}

		if err := graphqlClient.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		table := format.TabularData{
			Data: respData.NotificationsAggregateRules.Rows,
			Fields: []format.TableField{
				{Header: "Severity", Field: "Severity"},
				{Header: "Source", Field: "Source"},
				{Header: "Rule Count", Field: "Count"},
			},
		}
		format.GetFormat().Print(table)

		return nil
	},
}

func init() {
	eventsCmd.AddCommand(eventsSummaryCmd)
}
