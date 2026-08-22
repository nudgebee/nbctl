package cmd

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/nudgebee/nbctl/pkg/nubi"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var nubiDeleteCmd = &cobra.Command{
	Use:     "delete <conversation-id>",
	Aliases: []string{"remove", "rm"},
	Short:   "Delete a conversation by ID",
	Long:    `Delete a conversation and all its associated messages from backend storage.`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		conversationID := args[0]
		accountID := viper.GetString("account-id")
		if accountID == "" {
			return fmt.Errorf("account-id is required, please set it in your config file")
		}

		username := viper.GetString("username")
		if username == "" {
			return fmt.Errorf("username is required, please set it in your config file")
		}

		endpoint := viper.GetString("endpoint")
		sessionID := uuid.New().String()
		nubiClient := nubi.New(client.NewClient(), accountID, username, sessionID, endpoint)

		if err := nubiClient.DeleteConversation(cmd.Context(), conversationID); err != nil {
			return fmt.Errorf("failed to delete conversation: %w", err)
		}

		if format.GetFormat().Get() == "json" {
			format.GetFormat().Print(map[string]string{
				"status":          "deleted",
				"conversation_id": conversationID,
			})
			return nil
		}

		out := format.GetFormat().GetOutput()
		_, _ = fmt.Fprintf(out, "Deleted conversation %s\n", conversationID)
		return nil
	},
}

func init() {
	nubiCmd.AddCommand(nubiDeleteCmd)
}
