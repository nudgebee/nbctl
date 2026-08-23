package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/nudgebee/nbctl/pkg/nubi"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var nubiGetCmd = &cobra.Command{
	Use:   "get <conversation-id>",
	Short: "Get details of a specific conversation",
	Long:  `Retrieve all messages, sub-agent steps, tool calls, and details for a specific conversation ID.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		conversationID := args[0]
		accountID := viper.GetString("account-id")
		if accountID == "" {
			return fmt.Errorf("account-id is required, please set it in your config file or pass via flag")
		}

		username := viper.GetString("username")
		if username == "" {
			return fmt.Errorf("username is required, please set it in your config file")
		}

		endpoint := viper.GetString("endpoint")
		sessionID := uuid.New().String()
		nubiClient := nubi.New(client.NewClient(), accountID, username, sessionID, endpoint)
		nubiClient.ConversationID = conversationID

		ctx := cmd.Context()

		if format.GetFormat().Get() == "json" {
			details, err := nubiClient.GetConversationDetails(ctx)
			if err != nil {
				return fmt.Errorf("failed to get conversation details: %w", err)
			}
			stats, _ := nubiClient.GetConversationStats(ctx, conversationID)
			result := map[string]interface{}{
				"account_id":      accountID,
				"conversation_id": conversationID,
				"details":         details,
			}
			if stats != nil {
				result["stats"] = stats
			}
			format.GetFormat().Print(result)
			return nil
		}

		messages, err := nubiClient.SwitchToConversation(conversationID)
		if err != nil {
			return fmt.Errorf("failed to retrieve conversation: %w", err)
		}

		out := format.GetFormat().GetOutput()
		borderStyle := lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).Padding(0, 1)

		for _, msg := range messages {
			if msg.Role == "human" {
				_, _ = fmt.Fprintf(out, "\n>>> %s\n", msg.Message)
			} else {
				rendered, err := renderMarkdown(msg.Response)
				if err != nil {
					_, _ = fmt.Fprintln(out, msg.Response)
				} else {
					_, _ = fmt.Fprintln(out, borderStyle.Render(rendered))
				}
			}
		}

		details, err := nubiClient.GetConversationDetails(ctx)
		if err != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "\nWarning: failed to retrieve tool executions: %v\n", err)
		} else if details != nil && len(details.ToolCalls) > 0 {
			_, _ = fmt.Fprintf(out, "\nTool Executions (%d):\n", len(details.ToolCalls))
			type toolRow struct {
				ToolName  string
				Status    string
				StartedAt string
				EndedAt   string
				Duration  string
			}
			var rows []toolRow
			for _, t := range details.ToolCalls {
				toolName, _ := t["tool_name"].(string)
				status, _ := t["status"].(string)
				if status == "" {
					status = "SUCCESS"
				}
				startedAt, _ := t["created_at"].(string)
				if startedAt == "" {
					startedAt = "-"
				}
				endedAt, _ := t["updated_at"].(string)
				if endedAt == "" {
					endedAt = "-"
				}
				duration, _ := t["duration"].(string)
				if duration == "" {
					duration = "-"
				}
				rows = append(rows, toolRow{
					ToolName:  toolName,
					Status:    status,
					StartedAt: startedAt,
					EndedAt:   endedAt,
					Duration:  duration,
				})
			}
			format.GetFormat().Print(format.TabularData{
				Data: rows,
				Fields: []format.TableField{
					{Header: "Tool Name", Field: "ToolName"},
					{Header: "Status", Field: "Status"},
					{Header: "Started At", Field: "StartedAt"},
					{Header: "Ended At", Field: "EndedAt"},
					{Header: "Duration", Field: "Duration"},
				},
			})
		}

		return nil
	},
}

func init() {
	nubiCmd.AddCommand(nubiGetCmd)
}
