package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
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

		integrationID, _ := cmd.Flags().GetString("integration-id")
		title, _ := cmd.Flags().GetString("title")
		projectKey, _ := cmd.Flags().GetString("project-key")
		description, _ := cmd.Flags().GetString("description")
		ticketType, _ := cmd.Flags().GetString("ticket-type")
		severity, _ := cmd.Flags().GetString("severity")
		assignee, _ := cmd.Flags().GetString("assignee")
		referenceID, _ := cmd.Flags().GetString("reference-id")
		source, _ := cmd.Flags().GetString("source")
		additionalFields, _ := cmd.Flags().GetString("additional-fields")

		if integrationID == "" {
			return fmt.Errorf("--integration-id is required")
		}
		if title == "" {
			return fmt.Errorf("--title is required")
		}
		if projectKey == "" {
			return fmt.Errorf("--project-key is required")
		}

		vars := map[string]any{
			"account_id":     accountID,
			"integration_id": integrationID,
			"title":          title,
			"project_key":    projectKey,
		}
		if description != "" {
			vars["description"] = description
		}
		if ticketType != "" {
			vars["ticket_type"] = ticketType
		}
		if severity != "" {
			vars["severity"] = severity
		}
		if assignee != "" {
			vars["assignee"] = assignee
		}
		if referenceID != "" {
			vars["reference_id"] = referenceID
		}
		if source != "" {
			vars["source"] = source
		}
		if additionalFields != "" {
			var parsed any
			if err := json.Unmarshal([]byte(additionalFields), &parsed); err != nil {
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
	ticketsCreateCmd.Flags().String("integration-id", "", "Ticket integration configuration ID (required)")
	ticketsCreateCmd.Flags().String("title", "", "Ticket title (required)")
	ticketsCreateCmd.Flags().String("project-key", "", "Project/board key in the external system (required)")
	ticketsCreateCmd.Flags().String("description", "", "Ticket description / body")
	ticketsCreateCmd.Flags().String("ticket-type", "", "Issue type (e.g. Bug, Task)")
	ticketsCreateCmd.Flags().String("severity", "", "Severity / priority")
	ticketsCreateCmd.Flags().String("assignee", "", "Assignee identifier")
	ticketsCreateCmd.Flags().String("reference-id", "", "Reference ID linking this ticket to a Nudgebee event")
	ticketsCreateCmd.Flags().String("source", "", "Source system identifier")
	ticketsCreateCmd.Flags().String("additional-fields", "", "Additional integration-specific fields as a JSON object")
}
