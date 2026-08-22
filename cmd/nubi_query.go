package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/briandowns/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/nudgebee/nbctl/pkg/nubi"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var nubiQueryAsync bool

var nubiQueryCmd = &cobra.Command{
	Use:     "query <prompt>",
	Aliases: []string{"ask", "exec", "run"},
	Short:   "Execute a single query non-interactively and exit",
	Long:    `Execute a single prompt against Nubi AI non-interactively and display the result.`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rawQuery := args[0]
		query := strings.TrimSpace(rawQuery)
		if query == "" {
			return fmt.Errorf("query cannot be empty")
		}

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

		ctx, cancel := context.WithCancel(cmd.Context())
		defer cancel()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		go func() {
			select {
			case <-sigChan:
				cancel()
			case <-ctx.Done():
			}
		}()
		defer signal.Stop(sigChan)

		async, _ := cmd.Flags().GetBool("async")
		if async {
			if err := nubiClient.TriggerInvestigation(ctx, query); err != nil {
				return fmt.Errorf("failed to trigger investigation: %w", err)
			}
			if format.GetFormat().Get() == "json" {
				format.GetFormat().Print(map[string]interface{}{
					"message":    "Investigation triggered asynchronously.",
					"session_id": nubiClient.SessionID,
					"account_id": nubiClient.AccountID,
					"query":      query,
				})
				return nil
			}
			out := format.GetFormat().GetOutput()
			_, _ = fmt.Fprintln(out, "Investigation triggered asynchronously.")
			_, _ = fmt.Fprintf(out, "Session ID: %s\n", nubiClient.SessionID)
			return nil
		}

		s := &nubiQueryShell{
			nubiClient: nubiClient,
			spinner:    spinner.New(spinner.CharSets[9], 100*time.Millisecond),
		}

		if format.GetFormat().Get() != "json" {
			s.spinner.Start()
		}
		startTime := time.Now()
		response, status, err := s.triggerAndPoll(ctx, query)
		duration := time.Since(startTime)
		if s.spinner.Active() {
			s.spinner.Stop()
		}

		out := format.GetFormat().GetOutput()

		if err != nil {
			if errors.Is(err, context.Canceled) {
				if format.GetFormat().Get() == "json" {
					format.GetFormat().Print(map[string]interface{}{
						"error":  "Request canceled.",
						"status": "CANCELED",
					})
					return nil
				}
				_, _ = fmt.Fprintln(out, "Request canceled.")
				return nil
			}
			return fmt.Errorf("error executing query: %w", err)
		}

		endpointURL := strings.TrimSuffix(nubiClient.Endpoint, "/")
		conversationURL := fmt.Sprintf("%s/ask-nudgebee?accountId=%s&conversation_id=%s", endpointURL, nubiClient.AccountID, nubiClient.ConversationID)

		if format.GetFormat().Get() == "json" {
			stats, _ := nubiClient.GetConversationStats(ctx, nubiClient.ConversationID)
			details, _ := nubiClient.GetConversationDetails(ctx)

			var respObj any
			trimmedResp := strings.TrimSpace(response)
			if err := json.Unmarshal([]byte(trimmedResp), &respObj); err != nil {
				respObj = response
			}

			result := map[string]interface{}{
				"account_id":      nubiClient.AccountID,
				"conversation_id": nubiClient.ConversationID,
				"session_id":      nubiClient.SessionID,
				"query":           query,
				"response":        respObj,
				"status":          status,
				"duration":        duration.String(),
				"duration_ms":     duration.Milliseconds(),
				"url":             conversationURL,
			}
			if details != nil {
				result["details"] = details
			}
			if stats != nil {
				result["stats"] = stats
			}
			format.GetFormat().Print(result)
			return nil
		}

		grayStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

		if status == "WAITING" {
			_, _ = fmt.Fprintln(out, response)
			_, _ = fmt.Fprintln(out, grayStyle.Render(fmt.Sprintf("\nNote: Nubi is waiting for a followup response. To continue interactively, run 'nbctl nubi' and switch to this conversation using:\n  /conversation %s\nOr visit the URL below.", nubiClient.ConversationID)))
		} else {
			rendered, err := renderMarkdown(response)
			if err != nil {
				_, _ = fmt.Fprintf(out, "Error rendering markdown: %v\n", err)
				_, _ = fmt.Fprintln(out, response)
			} else {
				borderStyle := lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).Padding(0, 1)
				_, _ = fmt.Fprintln(out, borderStyle.Render(rendered))
			}
		}

		metrics, err := nubiClient.GetUsageMetrics(ctx)
		if err == nil && metrics != "" {
			_, _ = fmt.Fprintln(out, grayStyle.Render(metrics))
		}

		_, _ = fmt.Fprintln(out, grayStyle.Render(fmt.Sprintf("Response time: %s", duration)))

		conversationURLText := fmt.Sprintf("For more details: %s", conversationURL)
		_, _ = fmt.Fprintln(out, grayStyle.Render(conversationURLText))

		return nil
	},
}

type nubiQueryShell struct {
	nubiClient *nubi.NubiClient
	spinner    *spinner.Spinner
}

func (s *nubiQueryShell) triggerAndPoll(ctx context.Context, query string) (string, string, error) {
	if err := s.nubiClient.TriggerInvestigation(ctx, query); err != nil {
		return "", "", fmt.Errorf("triggering investigation: %w", err)
	}
	return s.poll(ctx)
}

func (s *nubiQueryShell) poll(ctx context.Context) (string, string, error) {
	for {
		select {
		case <-ctx.Done():
			return "", "", ctx.Err()
		case <-time.After(2 * time.Second):
			resp, status, _, _, _, _, err := s.nubiClient.GetConversation(ctx)
			if err != nil {
				return "", "", fmt.Errorf("getting conversation: %w", err)
			}

			if status != "IN_PROGRESS" {
				return resp, status, nil
			}
		}
	}
}

func init() {
	nubiQueryCmd.Flags().BoolVar(&nubiQueryAsync, "async", false, "Trigger query asynchronously without waiting for response")
	nubiCmd.AddCommand(nubiQueryCmd)
}
