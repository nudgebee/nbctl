package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/atotto/clipboard"
	"github.com/briandowns/spinner"
	"github.com/c-bata/go-prompt"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/nudgebee/nbctl/pkg/log"
	"github.com/nudgebee/nbctl/pkg/nubi"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var suggestions = []prompt.Suggest{
	{Text: "/help", Description: "Show this help message"},
	{Text: "/bookmarks", Description: "Manage your bookmarks"},
	{Text: "/conversation", Description: "Start a new conversation or switch to another one"},
	{Text: "/history", Description: "Show the last n conversations (default: 10)"},
	{Text: "/account", Description: "Switch to a different account"},
	{Text: "/agents", Description: "List available agents"},
	{Text: "/tools", Description: "List available tools"},
	{Text: "/functions", Description: "List available functions"},
	{Text: "/copy", Description: "Copy the last response to clipboard"},
	{Text: "/exit", Description: "Exit the Nubi shell"},
}

func completer(d prompt.Document) []prompt.Suggest {
	if strings.HasPrefix(d.Text, "/") {
		return prompt.FilterHasPrefix(suggestions, d.GetWordBeforeCursor(), true)
	}
	return nil
}

var nubiCmd = &cobra.Command{
	Use:   "nubi [account-id]",
	Short: "Nudgebee Interactive session",
	Long:  `Start an interactive session with Nudgebee AI to ask questions and get insights.`,
	RunE: func(cmd *cobra.Command, args []string) error {

		var accountID string
		if len(args) > 0 {
			accountID = args[0]
		} else {
			accountID = viper.GetString("account-id")
		}

		if accountID == "" {
			return fmt.Errorf("account-id is required, please provide it as an argument or set it in your config file")
		}

		username := viper.GetString("username")
		if username == "" {
			return fmt.Errorf("username is required, please set it in your config file")
		}

		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("finding home directory: %w", err)
		}
		configPath := filepath.Join(home, ".nudgebee")
		historyFile := filepath.Join(configPath, "history")
		history, err := readHistory(historyFile)
		if err != nil {
			// Log the error but don't fail
			log.Errorf("Error reading history file: %v\n", err)
		}

		s := &nubiShell{
			nubiClient: nubi.New(
				client.NewClient(),
				accountID,
				username,
				uuid.New().String(),
				viper.GetString("endpoint"),
			),
			spinner:     spinner.New(spinner.CharSets[9], 100*time.Millisecond),
			historyFile: historyFile,
			history:     history,
		}

		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGTERM)
		go func() {
			<-signals
			if s.spinner.Active() {
				s.spinner.Stop()
			}
			fmt.Println("\nGoodbye!")
			os.Exit(0)
		}()

		printNubiArt()

		// Welcome message styling
		style := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00B3FF"))

		fmt.Printf("Hello %s!\n", style.Render(username))
		fmt.Printf("Using account: %s\n\n", style.Render(accountID))

		printNubiHelp()

		p := prompt.New(
			s.executor,
			completer,
			prompt.OptionPrefix(">>> "),
			prompt.OptionPrefixTextColor(prompt.Yellow),
			prompt.OptionTitle("Nubi Shell"),
			prompt.OptionHistory(s.history),
			prompt.OptionAddKeyBind(prompt.KeyBind{
				Key: prompt.ControlC,
				Fn: func(_ *prompt.Buffer) {
					if s.cancel != nil {
						s.cancel()
						s.cancel = nil
					}
					fmt.Println("Goodbye!")
					os.Exit(0)
				},
			}),
			prompt.OptionAddKeyBind(prompt.KeyBind{
				Key: prompt.Escape,
				Fn: func(_ *prompt.Buffer) {
					if s.cancel != nil {
						s.cancel()
						s.cancel = nil
					}
					s.nubiClient.StopConversation()
					fmt.Println("Conversation stopped. You can type a new command.")
				},
			}),
		)
		p.Run()

		return nil
	},
}

type nubiShell struct {
	nubiClient       *nubi.NubiClient
	spinner          *spinner.Spinner
	cancel           context.CancelFunc
	historyFile      string
	history          []string
	waitingMessageID string
	waitingAgentID   string
	lastResponse     string
}

type MessageConfig struct {
	FollowupOptions []string `json:"followupOptions"`

	FollowupType string `json:"followupType"`

	Question string `json:"question"`
}

func (s *nubiShell) executor(in string) {

	if strings.TrimSpace(in) == "" {
		return
	}
	s.history = append(s.history, in)
	go func() {
		if err := saveHistory(s.historyFile, s.history); err != nil {
			log.Errorf("Error saving history: %v\n", err)
		}
	}()

	if strings.HasPrefix(in, "/") {
		s.handleSlashCommand(in)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	s.spinner.Start()
	startTime := time.Now()
	var response string
	var status string
	var err error

	// Check if a conversation is already in progress
	if s.nubiClient.ConversationID != "" {
		_, currentStatus, _, _, waitingMessageID, waitingAgentID, getErr := s.nubiClient.GetConversation(context.Background())
		if getErr != nil {
			fmt.Printf("Error checking conversation status: %v\n", getErr)
			// Even if there's an error checking status, we should probably reset to allow a new conversation
			s.nubiClient.ConversationID = ""
			s.nubiClient.SessionID = uuid.New().String() // Generate a new session ID
			fmt.Println("Could not check previous conversation status. Starting a new conversation.")
			response, status, err = s.triggerAndPoll(ctx, in)
		} else {
			s.waitingMessageID = waitingMessageID
			s.waitingAgentID = waitingAgentID
			switch currentStatus {
			case "IN_PROGRESS":
				fmt.Println("Previous conversation was still in progress. Starting a new conversation for your query.")
				s.nubiClient.ConversationID = ""
				s.nubiClient.SessionID = uuid.New().String() // Generate a new session ID
				response, status, err = s.triggerAndPoll(ctx, in)
			case "WAITING":
				err = s.nubiClient.SendFollowupResponse(context.Background(), in, s.waitingAgentID, s.waitingMessageID)
				if err == nil {
					response, status, err = s.poll(ctx)
				}
			default:
				// For other statuses like COMPLETED, FAILED, etc., continue the conversation
				response, status, err = s.triggerAndPoll(ctx, in)
			}
		}
	} else {
		response, status, err = s.triggerAndPoll(ctx, in)
	}
	duration := time.Since(startTime)
	s.spinner.Stop()

	if err != nil {
		if err == context.Canceled {
			fmt.Println("Request canceled.")
		} else {
			fmt.Printf("Error: %v\n", err)
		}
		return
	}

	s.lastResponse = response

	if status == "WAITING" {
		fmt.Println(response)
	} else {
		rendered, err := renderMarkdown(response)
		if err != nil {
			fmt.Printf("Error rendering markdown: %v\n", err)
			fmt.Println(response) // fallback to raw
		} else {
			borderStyle := lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).Padding(0, 1)
			fmt.Println(borderStyle.Render(rendered))
		}
	}

	metrics, err := s.nubiClient.GetUsageMetrics(context.Background()) // use a new context
	if err != nil {
		// Silently ignore metrics errors for now
	} else if metrics != "" {
		gray := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		fmt.Println(gray.Render(metrics))
	}

	gray := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	fmt.Println(gray.Render(fmt.Sprintf("Response time: %s", duration)))

	conversationURL := fmt.Sprintf("For more details: %s/ask-nudgebee?accountId=%s&conversation_id=%s", s.nubiClient.Endpoint, s.nubiClient.AccountID, s.nubiClient.ConversationID)
	fmt.Println(gray.Render(conversationURL))
}

func (s *nubiShell) handleSlashCommand(in string) {
	parts := strings.Fields(in)
	command := parts[0]

	switch command {
	case "/conversation":
		if len(parts) < 2 {
			s.nubiClient.ConversationID = ""
			s.nubiClient.SessionID = uuid.New().String()
			fmt.Println("Started a new conversation.")
			return
		}
		conversationID := parts[1]
		messages, err := s.nubiClient.SwitchToConversation(conversationID)
		if err != nil {
			fmt.Printf("Error switching conversation: %v\n", err)
		} else {
			fmt.Printf("Switched to conversation %s\n", conversationID)
			for _, msg := range messages {
				var markdown string
				if msg.Role == "human" {
					markdown = ">>> " + msg.Message
					fmt.Println(markdown)
					markdown = msg.Response
					rendered, err := renderMarkdown(markdown)
					if err != nil {
						fmt.Printf("Error rendering markdown: %v\n", err)
						fmt.Println(markdown) // fallback to raw
					} else {
						borderStyle := lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).Padding(0, 1)
						fmt.Println(borderStyle.Render(rendered))
					}
				}
			}
			// Now, check the status and show followup
			_, status, _, followupJSON, _, _, err := s.nubiClient.GetConversation(context.Background())
			if err == nil && status == "WAITING" && followupJSON != "" {
				var msgConfig MessageConfig
				if err := json.Unmarshal([]byte(followupJSON), &msgConfig); err == nil {
					var builder strings.Builder
					builder.WriteString(msgConfig.Question)
					builder.WriteString("\n\n")
					builder.WriteString("Options:\n")
					for _, opt := range msgConfig.FollowupOptions {
						fmt.Fprintf(&builder, "- %s\n", opt)
					}
					rendered, err := renderMarkdown(builder.String())
					if err != nil {
						fmt.Println(builder.String())
					} else {
						borderStyle := lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).Padding(0, 1)
						fmt.Println(borderStyle.Render(rendered))
					}
				}
			}
		}
	case "/history":
		limit := 10
		if len(parts) > 1 {
			var err error
			limit, err = strconv.Atoi(parts[1])
			if err != nil {
				fmt.Println("Usage: /history [n] where n is a number")
				return
			}
		}
		history, err := s.nubiClient.ShowHistory(limit)
		if err != nil {
			fmt.Printf("Error fetching history: %v\n", err)
			return
		}
		format.GetFormat().Print(format.TabularData{
			Data: history,
			Fields: []format.TableField{
				{Header: "ID", Field: "ID"},
				{Header: "UPDATED AT", Field: "UpdatedAt"},
				{Header: "TITLE", Field: "Title"},
			},
		})
	case "/account":
		if len(parts) < 2 {
			fmt.Println("Usage: /account <id>")
			return
		}
		accountID := parts[1]
		s.nubiClient.AccountID = accountID
		s.nubiClient.SessionID = uuid.New().String()
		s.nubiClient.ConversationID = ""
		fmt.Printf("Switched to account %s\n", accountID)
	case "/help":
		fmt.Println("Available commands:")
		fmt.Println("  /help: Show this help message")
		fmt.Println("  /bookmarks [add|remove|list]: Manage your bookmarks")
		fmt.Println("  /conversation [id]: Start a new conversation or switch to a different one")
		fmt.Println("  /history [n]: Show the last n conversations (default: 10)")
		fmt.Println("  /account <id>: Switch to a different account")
		fmt.Println("  /agents: List available agents")
		fmt.Println("  /tools: List available tools")
		fmt.Println("  /functions: List available functions")
		fmt.Println("  /exit: Exit the Nubi shell")
	case "/bookmarks":
		s.handleBookmarkCommand(parts)
	case "/agents":
		agents, err := s.nubiClient.ListAgents(context.Background())
		if err != nil {
			fmt.Printf("Error listing agents: %v\n", err)
			return
		}
		format.GetFormat().Print(format.TabularData{
			Data: agents,
			Fields: []format.TableField{
				{Header: "NAME", Field: "Name"},
				{Header: "DESCRIPTION", Field: "Description"},
				{Header: "STATUS", Field: "Status"},
				{Header: "TOOLS", Field: "Tools"},
			},
		})
	case "/tools":
		tools, err := s.nubiClient.ListTools(context.Background())
		if err != nil {
			fmt.Printf("Error listing tools: %v\n", err)
			return
		}
		format.GetFormat().Print(format.TabularData{
			Data: tools,
			Fields: []format.TableField{
				{Header: "NAME", Field: "Name"},
				{Header: "DESCRIPTION", Field: "Description"},
				{Header: "STATUS", Field: "Status"},
				{Header: "TYPE", Field: "NBToolType"},
			},
		})
	case "/functions":
		functions, err := s.nubiClient.ListFunctions()
		if err != nil {
			fmt.Printf("Error listing functions: %v\n", err)
			return
		}
		format.GetFormat().Print(format.TabularData{
			Data: functions,
			Fields: []format.TableField{
				{Header: "NAME", Field: "Name"},
				{Header: "DESCRIPTION", Field: "Description"},
				{Header: "STATUS", Field: "Status"},
				{Header: "VARIABLES", Field: "Variables"},
			},
		})
	case "/copy":
		if s.lastResponse == "" {
			fmt.Println("Nothing to copy.")
			return
		}
		if err := clipboard.WriteAll(s.lastResponse); err != nil {
			fmt.Printf("Error copying to clipboard: %v\n", err)
		} else {
			fmt.Println("Copied to clipboard!")
		}
	case "/exit":
		fmt.Println("Goodbye!")
		os.Exit(0)
	default:
		fmt.Printf("Unknown command: %s\n", command)
	}
}

func (s *nubiShell) handleBookmarkCommand(parts []string) {
	subcommand := "list" // Default to list
	if len(parts) > 1 {
		subcommand = parts[1]
	}

	var conversationID string
	if len(parts) > 2 {
		conversationID = parts[2]
	} else {
		conversationID = s.nubiClient.ConversationID
	}

	switch subcommand {
	case "add":
		if conversationID == "" {
			fmt.Println("No conversation to bookmark.")
			return
		}
		if err := s.nubiClient.AddBookmark(conversationID); err != nil {
			fmt.Printf("Error adding bookmark: %v\n", err)
		} else {
			fmt.Printf("Bookmarked conversation %s\n", conversationID)
		}
	case "remove":
		if conversationID == "" {
			fmt.Println("No conversation to remove bookmark from.")
			return
		}
		if err := s.nubiClient.RemoveBookmark(conversationID); err != nil {
			fmt.Printf("Error removing bookmark: %v\n", err)
		} else {
			fmt.Printf("Removed bookmark for conversation %s\n", conversationID)
		}
	case "list":
		bookmarks, err := s.nubiClient.ListBookmarks()
		if err != nil {
			fmt.Printf("Error listing bookmarks: %v\n", err)
			return
		}
		format.GetFormat().Print(format.TabularData{
			Data: bookmarks,
			Fields: []format.TableField{
				{Header: "ID", Field: "ID"},
				{Header: "TITLE", Field: "Title"},
			},
		})
	default:
		fmt.Println("Usage: /bookmarks <add|remove|list> [conversationId]")
	}
}

func (s *nubiShell) poll(ctx context.Context) (string, string, error) {
	for {
		select {
		case <-ctx.Done():
			return "", "", ctx.Err()
		case <-time.After(2 * time.Second):
			resp, status, statusText, followupJSON, waitingMessageID, waitingAgentID, err := s.nubiClient.GetConversation(ctx)
			if err != nil {
				return "", "", err
			}
			s.waitingMessageID = waitingMessageID
			s.waitingAgentID = waitingAgentID

			if s.spinner != nil {
				cyan := "\033[36m"
				reset := "\033[0m"
				s.spinner.Suffix = " " + cyan + statusText + reset
			}

			if status == "WAITING" && followupJSON != "" {
				var msgConfig MessageConfig
				if err := json.Unmarshal([]byte(followupJSON), &msgConfig); err == nil {
					var builder strings.Builder
					builder.WriteString(msgConfig.Question)
					builder.WriteString("\n\n")
					builder.WriteString("Options:\n")
					for _, opt := range msgConfig.FollowupOptions {
						fmt.Fprintf(&builder, "- %s\n", opt)
					}
					return builder.String(), status, nil
				}
			}

			if status != "IN_PROGRESS" && status != "WAITING" {
				return resp, status, nil
			}
		}
	}
}

func (s *nubiShell) triggerAndPoll(ctx context.Context, query string) (string, string, error) {
	if err := s.nubiClient.TriggerInvestigation(ctx, query); err != nil {
		return "", "", err
	}
	return s.poll(ctx)
}

func renderMarkdown(in string) (string, error) {
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(120),
	)
	if err != nil {
		return "", err
	}
	return r.Render(in)
}

func readHistory(file string) ([]string, error) {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return nil, nil // No history file yet
	}
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Errorf("Error closing history file for reading: %v\n", err)
		}
	}()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func saveHistory(file string, history []string) error {
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Errorf("Error closing history file for writing: %v\n", err)
		}
	}()

	// De-duplicate history before saving
	seen := make(map[string]struct{})
	var uniqueHistory []string
	for i := len(history) - 1; i >= 0; i-- {
		line := history[i]
		if _, ok := seen[line]; !ok {
			seen[line] = struct{}{}
			uniqueHistory = append([]string{line}, uniqueHistory...)
		}
	}

	for _, line := range uniqueHistory {
		if _, err := fmt.Fprintln(f, line); err != nil {
			return err
		}
	}
	return nil
}

func initNubiClient(args []string) (*nubi.NubiClient, error) {
	return initNubiClientWithAccountRequirement(args, true)
}

func initNubiClientOptionalAccount(args []string) (*nubi.NubiClient, error) {
	return initNubiClientWithAccountRequirement(args, false)
}

func initNubiClientWithAccountRequirement(args []string, requireAccount bool) (*nubi.NubiClient, error) {
	var accountID string
	if len(args) > 0 {
		accountID = args[0]
	} else {
		accountID = viper.GetString("account-id")
	}

	if requireAccount && accountID == "" {
		return nil, fmt.Errorf("account-id is required, please provide it as an argument or set it in your config file")
	}

	username := viper.GetString("username")
	if username == "" {
		return nil, fmt.Errorf("username is required, please set it in your config file")
	}

	endpoint := viper.GetString("endpoint")
	sessionID := uuid.New().String()
	return nubi.New(client.NewClient(), accountID, username, sessionID, endpoint), nil
}

func init() {
	rootCmd.AddCommand(nubiCmd)
}
