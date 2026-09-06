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

var (
	nubiQueryAsync   bool
	nubiQueryTimeout time.Duration
)

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

		accountID, err := resolveAccountID(cmd)
		if err != nil {
			return err
		}

		username := viper.GetString("username")
		if username == "" {
			return fmt.Errorf("username is required, please set it in your config file")
		}

		endpoint := viper.GetString("endpoint")
		sessionID := uuid.New().String()
		nubiClient := nubi.New(client.NewClient(), accountID, username, sessionID, endpoint)

		var ctx context.Context
		var cancel context.CancelFunc
		if nubiQueryTimeout > 0 {
			ctx, cancel = context.WithTimeout(cmd.Context(), nubiQueryTimeout)
		} else {
			ctx, cancel = context.WithCancel(cmd.Context())
		}
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
				if format.GetFormat().Get() == "json" {
					jsonResp := map[string]interface{}{
						"error":      fmt.Sprintf("failed to trigger investigation: %v", err),
						"status":     "ERROR",
						"account_id": nubiClient.AccountID,
						"query":      query,
					}
					if hint := triggerErrorHint(err, nubiClient.AccountID); hint != "" {
						jsonResp["hint"] = hint
					}
					format.GetFormat().Print(jsonResp)
					return nil
				}
				if hint := triggerErrorHint(err, nubiClient.AccountID); hint != "" {
					return fmt.Errorf("failed to trigger investigation: %w\nHint: %s", err, hint)
				}
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
			isTimeout := errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded)
			isCanceled := errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled)

			if isTimeout || isCanceled {
				statusStr := "TIMED_OUT"
				durStr := duration.Round(time.Second).String()
				if duration < time.Second {
					durStr = duration.Round(100 * time.Millisecond).String()
				}
				errMsg := fmt.Sprintf("Query timed out after %s.", durStr)
				if isCanceled {
					statusStr = "CANCELED"
					errMsg = "Request canceled."
				}

				endpointURL := strings.TrimSuffix(nubiClient.Endpoint, "/")
				refID := nubiClient.ConversationID
				if refID == "" {
					refID = nubiClient.SessionID
				}
				conversationURL := fmt.Sprintf("%s/ask-nudgebee?accountId=%s&conversation_id=%s", endpointURL, nubiClient.AccountID, refID)

				if format.GetFormat().Get() == "json" {
					jsonResp := map[string]interface{}{
						"error":      errMsg,
						"status":     statusStr,
						"account_id": nubiClient.AccountID,
						"session_id": nubiClient.SessionID,
						"query":      query,
						"url":        conversationURL,
						"hint":       "The investigation was triggered server-side. Retrieve results using 'nbctl nubi get' or increase timeout using '--timeout'.",
					}
					if nubiClient.ConversationID != "" {
						jsonResp["conversation_id"] = nubiClient.ConversationID
					}
					format.GetFormat().Print(jsonResp)
					return nil
				}

				grayStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
				boldStyle := lipgloss.NewStyle().Bold(true)
				yellowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))

				_, _ = fmt.Fprintln(out, yellowStyle.Render(errMsg))
				_, _ = fmt.Fprintln(out, grayStyle.Render("\nThe investigation was triggered and may still be running or completed server-side."))
				if nubiClient.ConversationID != "" {
					_, _ = fmt.Fprintf(out, "  %s %s\n", boldStyle.Render("Conversation ID:"), nubiClient.ConversationID)
				}
				_, _ = fmt.Fprintf(out, "  %s %s\n", boldStyle.Render("Session ID:"), nubiClient.SessionID)

				_, _ = fmt.Fprintln(out, grayStyle.Render("\nTo retrieve the response once completed:"))
				if nubiClient.ConversationID != "" {
					_, _ = fmt.Fprintf(out, "  nbctl nubi get %s\n", nubiClient.ConversationID)
				} else {
					_, _ = fmt.Fprintf(out, "  nbctl nubi get --session-id %s\n", nubiClient.SessionID)
				}

				_, _ = fmt.Fprintln(out, grayStyle.Render("\nTo view in browser:"))
				_, _ = fmt.Fprintf(out, "  %s\n", conversationURL)

				_, _ = fmt.Fprintln(out, grayStyle.Render("\nOptions to increase timeout or run in background:"))
				_, _ = fmt.Fprintf(out, "  nbctl nubi query %q --timeout 5m\n", query)
				_, _ = fmt.Fprintf(out, "  nbctl nubi query %q --async\n", query)

				return nil
			}

			// If polling failed after triggering investigation, provide recovery information
			if nubiClient.SessionID != "" && !strings.Contains(err.Error(), "triggering investigation") {
				endpointURL := strings.TrimSuffix(nubiClient.Endpoint, "/")
				refID := nubiClient.ConversationID
				if refID == "" {
					refID = nubiClient.SessionID
				}
				conversationURL := fmt.Sprintf("%s/ask-nudgebee?accountId=%s&conversation_id=%s", endpointURL, nubiClient.AccountID, refID)

				if format.GetFormat().Get() == "json" {
					jsonResp := map[string]interface{}{
						"error":      fmt.Sprintf("error executing query: %v", err),
						"status":     "ERROR",
						"account_id": nubiClient.AccountID,
						"session_id": nubiClient.SessionID,
						"query":      query,
						"url":        conversationURL,
						"hint":       "The investigation was triggered server-side. Retrieve results using 'nbctl nubi get'.",
					}
					if nubiClient.ConversationID != "" {
						jsonResp["conversation_id"] = nubiClient.ConversationID
					}
					format.GetFormat().Print(jsonResp)
					return nil
				}

				grayStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
				boldStyle := lipgloss.NewStyle().Bold(true)
				redStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

				_, _ = fmt.Fprintln(out, redStyle.Render(fmt.Sprintf("Error executing query: %v", err)))
				_, _ = fmt.Fprintln(out, grayStyle.Render("\nThe investigation was triggered and may still be running or completed server-side."))
				if nubiClient.ConversationID != "" {
					_, _ = fmt.Fprintf(out, "  %s %s\n", boldStyle.Render("Conversation ID:"), nubiClient.ConversationID)
				}
				_, _ = fmt.Fprintf(out, "  %s %s\n", boldStyle.Render("Session ID:"), nubiClient.SessionID)

				_, _ = fmt.Fprintln(out, grayStyle.Render("\nTo retrieve the response once completed:"))
				if nubiClient.ConversationID != "" {
					_, _ = fmt.Fprintf(out, "  nbctl nubi get %s\n", nubiClient.ConversationID)
				} else {
					_, _ = fmt.Fprintf(out, "  nbctl nubi get --session-id %s\n", nubiClient.SessionID)
				}
				return nil
			}

			if format.GetFormat().Get() == "json" {
				jsonResp := map[string]interface{}{
					"error":      fmt.Sprintf("error executing query: %v", err),
					"status":     "ERROR",
					"account_id": nubiClient.AccountID,
					"query":      query,
				}
				if hint := triggerErrorHint(err, nubiClient.AccountID); hint != "" {
					jsonResp["hint"] = hint
				}
				format.GetFormat().Print(jsonResp)
				return nil
			}

			if hint := triggerErrorHint(err, nubiClient.AccountID); hint != "" {
				return fmt.Errorf("error executing query: %w\nHint: %s", err, hint)
			}

			return fmt.Errorf("error executing query: %w", err)
		}

		endpointURL := strings.TrimSuffix(nubiClient.Endpoint, "/")
		conversationURL := fmt.Sprintf("%s/ask-nudgebee?accountId=%s&conversation_id=%s", endpointURL, nubiClient.AccountID, nubiClient.ConversationID)

		if format.GetFormat().Get() == "json" {
			metrics, _ := nubiClient.GetUsageMetrics(ctx)
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
			if metrics != "" {
				result["metrics"] = metrics
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
	consecutiveErrors := 0
	const maxConsecutiveErrors = 5

	check := func() (string, string, bool, error) {
		resp, status, statusText, _, _, _, err := s.nubiClient.GetConversation(ctx)
		if err != nil {
			if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return "", "", false, ctx.Err()
			}
			consecutiveErrors++
			if consecutiveErrors >= maxConsecutiveErrors {
				return "", "", false, fmt.Errorf("getting conversation: %w", err)
			}
			return "", "", false, nil
		}
		consecutiveErrors = 0

		if s.spinner != nil && statusText != "" {
			s.spinner.Suffix = " " + statusText
		}

		if status != "IN_PROGRESS" {
			return resp, status, true, nil
		}
		return "", "", false, nil
	}

	// Immediate check
	if resp, status, done, err := check(); done || err != nil {
		return resp, status, err
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", "", ctx.Err()
		case <-ticker.C:
			if resp, status, done, err := check(); done || err != nil {
				return resp, status, err
			}
		}
	}
}

func init() {
	nubiQueryCmd.Flags().BoolVar(&nubiQueryAsync, "async", false, "Trigger query asynchronously without waiting for response")
	nubiQueryCmd.Flags().DurationVarP(&nubiQueryTimeout, "timeout", "t", 0, "Maximum time to wait for query completion (e.g. 2m, 5m). Default is 0 (no timeout)")
	nubiQueryCmd.Flags().String("account-id", "", "Account ID to execute the query against")
	nubiCmd.AddCommand(nubiQueryCmd)
}

func triggerErrorHint(err error, accountID string) string {
	if err == nil {
		return ""
	}
	if strings.Contains(strings.ToLower(err.Error()), "user does not have access") {
		return fmt.Sprintf("User does not have access to account %s. Verify the account ID or assign an account role via 'nbctl auth assign-role'.", accountID)
	}
	return ""
}
