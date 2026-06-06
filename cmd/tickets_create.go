package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var (
	ticketsCreateIntegrationID    string
	ticketsCreateTitle            string
	ticketsCreateProjectKey       string
	ticketsCreateDescription      string
	ticketsCreateTicketType       string
	ticketsCreateSeverity         string
	ticketsCreateAssignee         string
	ticketsCreateReferenceID      string
	ticketsCreateSource           string
	ticketsCreateAdditionalFields string
)

var ticketsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a ticket in an external system",
	Long: `Create a ticket via a configured ticket integration (Jira, ServiceNow, PagerDuty, etc.).
Use ` + "`nbctl tickets list-configurations`" + ` to find the --integration-id and --project-key values.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		accountID, err := resolveAccountID(cmd)
		if err != nil {
			return err
		}
		if ticketsCreateIntegrationID == "" {
			return fmt.Errorf("--integration-id is required")
		}
		if ticketsCreateTitle == "" {
			return fmt.Errorf("--title is required")
		}
		if ticketsCreateProjectKey == "" {
			return fmt.Errorf("--project-key is required")
		}

		vars := map[string]any{
			"account_id":     accountID,
			"integration_id": ticketsCreateIntegrationID,
			"title":          ticketsCreateTitle,
			"project_key":    ticketsCreateProjectKey,
		}
		if ticketsCreateDescription != "" {
			vars["description"] = ticketsCreateDescription
		}
		if ticketsCreateTicketType != "" {
			vars["ticket_type"] = ticketsCreateTicketType
		}
		if ticketsCreateSeverity != "" {
			vars["severity"] = ticketsCreateSeverity
		}
		if ticketsCreateAssignee != "" {
			vars["assignee"] = ticketsCreateAssignee
		}
		if ticketsCreateReferenceID != "" {
			vars["reference_id"] = ticketsCreateReferenceID
		}
		if ticketsCreateSource != "" {
			vars["source"] = ticketsCreateSource
		}
		if ticketsCreateAdditionalFields != "" {
			var parsed any
			if err := json.Unmarshal([]byte(ticketsCreateAdditionalFields), &parsed); err != nil {
				return fmt.Errorf("--additional-fields must be valid JSON: %w", err)
			}
			vars["additional_fields"] = parsed
		}

		query := `
mutation CreateTicket($assignee: String, $integration_id: String, $reference_id: String, $ticket_type: String, $project_key: String, $description: String, $title: String, $source: String, $severity: String, $account_id: String, $additional_fields: jsonb) {
  tickets_create(object: {assignee: $assignee, integration_id: $integration_id, reference_id: $reference_id, ticket_type: $ticket_type, project_key: $project_key, description: $description, title: $title, source: $source, severity: $severity, account_id: $account_id, additional_fields: $additional_fields}) {
    data {
      insert_tickets_one {
        id
        ticket_id
        url
        action
        message
        error
      }
    }
  }
}
`
		req := client.NewRequest(query)
		for k, v := range vars {
			req.Var(k, v)
		}

		var respData struct {
			TicketsCreate struct {
				Data struct {
					InsertTicketsOne *struct {
						ID       string `json:"id"`
						TicketID string `json:"ticket_id"`
						URL      string `json:"url"`
						Action   string `json:"action"`
						Message  string `json:"message"`
						Error    string `json:"error"`
					} `json:"insert_tickets_one"`
				} `json:"data"`
			} `json:"tickets_create"`
		}

		c := client.NewClient()
		if err := c.Run(cmd.Context(), req, &respData); err != nil {
			return fmt.Errorf("failed to create ticket: %w", err)
		}

		result := respData.TicketsCreate.Data.InsertTicketsOne
		if result == nil {
			return fmt.Errorf("ticket creation failed: server returned no ticket data")
		}
		if result.Error != "" {
			return fmt.Errorf("ticket creation failed: %s", result.Error)
		}

		if format.GetFormat().Get() == "json" {
			format.GetFormat().Print(result)
			return nil
		}

		w := format.GetFormat().GetOutput()
		_, _ = fmt.Fprintf(w, "Ticket created.\n")
		if result.TicketID != "" {
			_, _ = fmt.Fprintf(w, "Ticket ID: %s\n", result.TicketID)
		}
		if result.URL != "" {
			_, _ = fmt.Fprintf(w, "URL:       %s\n", result.URL)
		}
		if result.Message != "" {
			_, _ = fmt.Fprintf(w, "Message:   %s\n", result.Message)
		}

		return nil
	},
}

func init() {
	ticketsCmd.AddCommand(ticketsCreateCmd)
	ticketsCreateCmd.Flags().String("account-id", "", "Account ID (overrides profile)")
	ticketsCreateCmd.Flags().StringVar(&ticketsCreateIntegrationID, "integration-id", "", "Ticket integration configuration ID (required)")
	ticketsCreateCmd.Flags().StringVar(&ticketsCreateTitle, "title", "", "Ticket title (required)")
	ticketsCreateCmd.Flags().StringVar(&ticketsCreateProjectKey, "project-key", "", "Project/board key in the external system (required)")
	ticketsCreateCmd.Flags().StringVar(&ticketsCreateDescription, "description", "", "Ticket description / body")
	ticketsCreateCmd.Flags().StringVar(&ticketsCreateTicketType, "ticket-type", "", "Issue type (e.g. Bug, Task)")
	ticketsCreateCmd.Flags().StringVar(&ticketsCreateSeverity, "severity", "", "Severity / priority")
	ticketsCreateCmd.Flags().StringVar(&ticketsCreateAssignee, "assignee", "", "Assignee identifier")
	ticketsCreateCmd.Flags().StringVar(&ticketsCreateReferenceID, "reference-id", "", "Reference ID linking this ticket to a Nudgebee event")
	ticketsCreateCmd.Flags().StringVar(&ticketsCreateSource, "source", "", "Source system identifier")
	ticketsCreateCmd.Flags().StringVar(&ticketsCreateAdditionalFields, "additional-fields", "", "Additional integration-specific fields as a JSON object")
}
