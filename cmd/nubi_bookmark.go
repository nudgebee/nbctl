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

var nubiBookmarkCmd = &cobra.Command{
	Use:     "bookmark [add|remove|list] [conversation-id]",
	Aliases: []string{"bookmarks"},
	Short:   "Manage bookmarked conversations",
	Long:    `Add, remove, or list bookmarked conversations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		action := "list"
		if len(args) > 0 {
			action = args[0]
		}

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

		switch action {
		case "add":
			if len(args) < 2 {
				return fmt.Errorf("conversation-id is required for bookmark add")
			}
			convID := args[1]
			if err := nubiClient.AddBookmark(convID); err != nil {
				return fmt.Errorf("failed to add bookmark: %w", err)
			}
			if format.GetFormat().Get() == "json" {
				format.GetFormat().Print(map[string]string{"status": "success", "message": "Bookmark added", "conversation_id": convID})
				return nil
			}
			out := format.GetFormat().GetOutput()
			_, _ = fmt.Fprintf(out, "Added bookmark for conversation %s\n", convID)
			return nil

		case "remove":
			if len(args) < 2 {
				return fmt.Errorf("conversation-id is required for bookmark remove")
			}
			convID := args[1]
			if err := nubiClient.RemoveBookmark(convID); err != nil {
				return fmt.Errorf("failed to remove bookmark: %w", err)
			}
			if format.GetFormat().Get() == "json" {
				format.GetFormat().Print(map[string]string{"status": "success", "message": "Bookmark removed", "conversation_id": convID})
				return nil
			}
			out := format.GetFormat().GetOutput()
			_, _ = fmt.Fprintf(out, "Removed bookmark for conversation %s\n", convID)
			return nil

		case "list":
			bookmarks, err := nubiClient.ListBookmarks()
			if err != nil {
				return fmt.Errorf("failed to list bookmarks: %w", err)
			}
			if format.GetFormat().Get() == "json" {
				format.GetFormat().Print(bookmarks)
				return nil
			}
			format.GetFormat().Print(format.TabularData{
				Data: bookmarks,
				Fields: []format.TableField{
					{Header: "ID", Field: "ID"},
					{Header: "UPDATED AT", Field: "UpdatedAt"},
					{Header: "TITLE", Field: "Title"},
				},
			})
			return nil

		default:
			return fmt.Errorf("unknown bookmark action '%s'. Supported actions: add, remove, list", action)
		}
	},
}

func init() {
	nubiCmd.AddCommand(nubiBookmarkCmd)
}
