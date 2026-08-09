package cmd

import (
	"fmt"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var ticketsCommentsCmd = &cobra.Command{
	Use:   "comments",
	Short: "Inspect and add comments to tickets",
}

var ticketsCommentsListCmd = &cobra.Command{
	Use:   "list <ticket-id>",
	Short: "List comments for a ticket",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ticketID := args[0]
		graphqlClient := client.NewClient()

		req := client.NewRequest(`
			query GetTicketComments($ticketId: String!) {
				ticket_get_comments(ticket_id: $ticketId) {
					comments {
						id
						author
						body
						created_at
					}
				}
			}
		`)
		req.Var("ticketId", ticketID)

		var respData struct {
			TicketGetComments struct {
				Comments []struct {
					ID        string `json:"id"`
					Author    string `json:"author"`
					Body      string `json:"body"`
					CreatedAt string `json:"created_at"`
				} `json:"comments"`
			} `json:"ticket_get_comments"`
		}

		if err := graphqlClient.Run(cmd.Context(), req, &respData); err != nil {
			return err
		}

		table := format.TabularData{
			Data: respData.TicketGetComments.Comments,
			Fields: []format.TableField{
				{Header: "Comment ID", Field: "ID"},
				{Header: "Author", Field: "Author"},
				{Header: "Comment", Field: "Body"},
				{Header: "Created At", Field: "CreatedAt"},
			},
		}
		format.GetFormat().Print(table)

		return nil
	},
}

var ticketsCommentsAddCmd = &cobra.Command{
	Use:   "add <ticket-id> <comment-body>",
	Short: "Add a comment to a ticket",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ticketID := args[0]
		body := args[1]

		graphqlClient := client.NewClient()

		req := client.NewRequest(`
			mutation AddTicketComment($ticketId: String!, $body: String!) {
				ticket_add_comment(ticket_id: $ticketId, body: $body) {
					status
					comment_id
				}
			}
		`)
		req.Var("ticketId", ticketID)
		req.Var("body", body)

		var respData struct {
			TicketAddComment struct {
				Status    string `json:"status"`
				CommentID string `json:"comment_id"`
			} `json:"ticket_add_comment"`
		}

		if err := graphqlClient.Run(cmd.Context(), req, &respData); err != nil {
			return err
		}

		fmt.Printf("Comment added to ticket %s successfully (Comment ID: %s)\n", ticketID, respData.TicketAddComment.CommentID)
		return nil
	},
}

func init() {
	ticketsCmd.AddCommand(ticketsCommentsCmd)
	ticketsCommentsCmd.AddCommand(ticketsCommentsListCmd)
	ticketsCommentsCmd.AddCommand(ticketsCommentsAddCmd)
}
