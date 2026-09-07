package cmd

import (
	"fmt"
	"strings"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var nubiMemoryCmd = &cobra.Command{
	Use:     "memory",
	Aliases: []string{"bcortex", "app-context"},
	Short:   "Manage AI-learned operational memory, architecture decisions, and application context",
}

type memoryItem struct {
	ID             string `json:"id"`
	AccountID      string `json:"account_id"`
	ConversationID string `json:"conversation_id"`
	MessageID      string `json:"message_id"`
	Content        string `json:"content"`
	MemoryType     string `json:"memory_type"`
	CreatedAt      string `json:"created_at"`
}

var nubiMemoryListCmd = &cobra.Command{
	Use:   "list [account-id]",
	Short: "List AI operational memory items, architecture decisions, and learned patterns",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		accountID, err := resolveAccountIDWithPositional(cmd, args)
		if err != nil {
			return err
		}

		memoryType, _ := cmd.Flags().GetString("type")
		queryFilter, _ := cmd.Flags().GetString("query")
		limit, _ := cmd.Flags().GetInt("limit")
		if limit <= 0 {
			return fmt.Errorf("limit must be greater than 0")
		}

		memoryType = strings.TrimSpace(memoryType)
		queryFilter = strings.TrimSpace(queryFilter)

		req := client.NewRequest(`
			query ListAIMemory($request: ListAIMemoryRequest!) {
				ai_list_memory(request: $request) {
					data {
						id
						account_id
						conversation_id
						message_id
						content
						memory_type
						created_at
					}
					errors {
						message
					}
				}
			}
		`)

		input := map[string]any{
			"account_id": accountID,
			"limit":      limit,
		}
		if memoryType != "" {
			input["memory_type"] = memoryType
		}
		if queryFilter != "" {
			input["query"] = queryFilter
		}
		req.Var("request", input)

		var respData struct {
			AiListMemory *struct {
				Data   []memoryItem       `json:"data"`
				Errors []graphqlErrorItem `json:"errors"`
			} `json:"ai_list_memory"`
		}

		if err := client.Run(cmd.Context(), req, &respData); err != nil {
			return err
		}

		if respData.AiListMemory == nil {
			return fmt.Errorf("empty response from backend")
		}

		if err := joinGraphQLErrors(respData.AiListMemory.Errors); err != nil {
			return err
		}

		table := format.TabularData{
			Data: respData.AiListMemory.Data,
			Fields: []format.TableField{
				{Header: "Memory ID", Field: "ID"},
				{Header: "Memory Type", Field: "MemoryType"},
				{Header: "Content", Field: "Content"},
				{Header: "Created At", Field: "CreatedAt"},
			},
		}
		format.GetFormat().Print(table)

		return nil
	},
}

func init() {
	nubiCmd.AddCommand(nubiMemoryCmd)
	nubiMemoryCmd.AddCommand(nubiMemoryListCmd)

	nubiMemoryListCmd.Flags().String("account-id", "", "Account ID (overrides profile)")
	nubiMemoryListCmd.Flags().String("type", "", "Filter memory by type (e.g. pattern, decision, architecture)")
	nubiMemoryListCmd.Flags().String("query", "", "Filter memory content by search term")
	nubiMemoryListCmd.Flags().Int("limit", 50, "Maximum number of memory items to return")
}
