package cmd

import (
	"bufio"
	"context"
	"crypto/md5"
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
	"github.com/nudgebee/nbctl/pkg/nubi/tools"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
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
			// If the first argument is "true" or "false", it's likely a misplaced boolean flag value
			if args[0] != "true" && args[0] != "false" {
				accountID = args[0]
			}
		}

		if accountID == "" {
			accountID = viper.GetString("account-id")
		}

		if accountID == "" || accountID == "true" || accountID == "false" {
			return fmt.Errorf("invalid account-id: %q. Please provide a valid account ID as an argument or set it in your config file", accountID)
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

		// Handle disabled tools merging
		disabledMap := make(map[string]bool)

		// If either flag was changed, merge them. Otherwise use the default from one of them.
		if cmd.Flags().Changed("disabled-tools") || cmd.Flags().Changed("disabled-server-tools") {
			for _, t := range viper.GetStringSlice("disabled-tools") {
				disabledMap[t] = true
			}
			for _, t := range viper.GetStringSlice("disabled-server-tools") {
				disabledMap[t] = true
			}
		} else {
			// Use default from the primary flag
			for _, t := range viper.GetStringSlice("disabled-tools") {
				disabledMap[t] = true
			}
		}

		var finalDisabledTools []string
		for t := range disabledMap {
			if t != "" {
				finalDisabledTools = append(finalDisabledTools, t)
			}
		}

		s := &nubiShell{
			nubiClient: nubi.New(
				client.NewClient(),
				accountID,
				username,
				uuid.New().String(),
				viper.GetString("endpoint"),
			),
			spinner:              spinner.New(spinner.CharSets[9], 100*time.Millisecond),
			historyFile:          historyFile,
			history:              history,
			processedToolCallIDs: make(map[string]bool),
			disabledTools:        finalDisabledTools,
			toolResults:          make(map[string]toolResult),
		}
		s.nubiClient.EnableLocalTools = viper.GetBool("enable-local-tools")

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
		infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

		fmt.Printf("Hello %s!\n", style.Render(username))
		fmt.Printf("Using account: %s\n", style.Render(accountID))

		if s.nubiClient.EnableLocalTools {
			localTools := tools.GetLocalToolNames()
			fmt.Printf("%s\n", infoStyle.Render(fmt.Sprintf("Local tools enabled: %s", strings.Join(localTools, ", "))))
		}
		if len(s.disabledTools) > 0 {
			fmt.Printf("%s\n", infoStyle.Render(fmt.Sprintf("Disabled server tools: %s", strings.Join(s.disabledTools, ", "))))
		}
		fmt.Println()

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
	nubiClient           *nubi.NubiClient
	spinner              *spinner.Spinner
	cancel               context.CancelFunc
	historyFile          string
	history              []string
	waitingMessageID     string
	waitingAgentID       string
	lastResponse         string
	processedToolCallIDs map[string]bool
	disabledTools        []string
	toolResults          map[string]toolResult // Cache for re-submission
}

type toolResult struct {
	result    string
	status    string
	timestamp time.Time
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
		details, getErr := s.nubiClient.GetConversation(context.Background())
		if getErr != nil {
			fmt.Printf("Error checking conversation status: %v\n", getErr)
			// Even if there's an error checking status, we should probably reset to allow a new conversation
			s.nubiClient.ConversationID = ""
			s.nubiClient.SessionID = uuid.New().String() // Generate a new session ID
			fmt.Println("Could not check previous conversation status. Starting a new conversation.")
			response, status, err = s.triggerAndPoll(ctx, in)
		} else {
			s.waitingMessageID = details.WaitingMessageID
			s.waitingAgentID = details.WaitingAgentID
			switch details.Status {
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
			case "WAITING_FOR_CLIENT_TOOL":
				response, status, err = s.poll(ctx)
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
			s.processedToolCallIDs = make(map[string]bool)
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
			details, err := s.nubiClient.GetConversation(context.Background())
			if err == nil && details.Status == "WAITING" && details.FollowupMessageConfig != "" {
				var msgConfig MessageConfig
				if err := json.Unmarshal([]byte(details.FollowupMessageConfig), &msgConfig); err == nil {
					var builder strings.Builder
					builder.WriteString(msgConfig.Question)
					builder.WriteString("\n\n")
					builder.WriteString("Options:\n")
					for _, opt := range msgConfig.FollowupOptions {
						builder.WriteString(fmt.Sprintf("- %s\n", opt))
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
		s.processedToolCallIDs = make(map[string]bool)
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
		agents, err := s.nubiClient.ListAgents()
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
		tools, err := s.nubiClient.ListTools()
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
			details, err := s.nubiClient.GetConversation(ctx)
			if err != nil {
				return "", "", err
			}
			s.waitingMessageID = details.WaitingMessageID
			s.waitingAgentID = details.WaitingAgentID

			if s.spinner != nil {
				cyan := "\033[36m"
				reset := "\033[0m"
				s.spinner.Suffix = " " + cyan + details.StatusText + reset
			}

			// If we have pending tool calls, pick the first one that hasn't been processed
			if len(details.PendingToolCalls) > 0 {
				for _, toolCall := range details.PendingToolCalls {
					// Use MessageID + UUID + Args hash to uniquely identify this tool call attempt
					// This ensures that if the server updates the parameters but keeps the same ID, we re-process it.
					argHash := fmt.Sprintf("%x", md5.Sum([]byte(toolCall.Args)))
					toolKey := fmt.Sprintf("%s:%s:%s", toolCall.MessageID, toolCall.UUID, argHash)

					if !s.processedToolCallIDs[toolKey] {
						details.WaitingToolCallID = toolCall.ID
						details.WaitingToolUUID = toolCall.UUID
						details.WaitingToolName = toolCall.Name
						details.WaitingToolArgs = toolCall.Args
						// Update waiting IDs to match the specific tool call
						s.waitingAgentID = toolCall.AgentID
						s.waitingMessageID = toolCall.MessageID
						break
					}
				}
			}

			// Handle client tool execution
			if details.WaitingToolCallID != "" && details.WaitingToolName != "" {
				argHash := fmt.Sprintf("%x", md5.Sum([]byte(details.WaitingToolArgs)))
				toolKey := fmt.Sprintf("%s:%s:%s", s.waitingMessageID, details.WaitingToolUUID, argHash)

				if s.processedToolCallIDs[toolKey] {
					// If already processed, check if we should re-submit (every 30s) if the server is still waiting
					cached := s.toolResults[toolKey]
					if !cached.timestamp.IsZero() && time.Since(cached.timestamp) > 30*time.Second {
						if s.spinner != nil {
							s.spinner.Suffix = fmt.Sprintf(" Still waiting for server to acknowledge %s, re-submitting...", details.WaitingToolName)
						}
						_ = s.nubiClient.SubmitClientToolResult(ctx, details.WaitingToolCallID, s.waitingAgentID, s.waitingMessageID, cached.result, cached.status)
						cached.timestamp = time.Now()
						s.toolResults[toolKey] = cached
					} else if s.spinner != nil {
						s.spinner.Suffix = fmt.Sprintf(" Already submitted %s, waiting for server...", details.WaitingToolName)
					}
					continue
				}

				args, err := lenientUnmarshal(details.WaitingToolArgs, details.WaitingToolName)
				result := ""
				resultStatus := "SUCCESS"

				if err != nil {
					// If all parsing fails, report error back to server
					result = fmt.Sprintf("Invalid argument format. Please provide arguments as a JSON or YAML object. Error: %v", err)
					resultStatus = "ERROR"
				} else {
					s.spinner.Suffix = fmt.Sprintf(" Executing local tool: %s", details.WaitingToolName)
					result, err = tools.ExecuteTool(ctx, details.WaitingToolName, args)
					if err != nil {
						result = err.Error()
						resultStatus = "ERROR"
						if s.spinner != nil {
							s.spinner.Stop()
							fmt.Printf("❌ Error executing tool %s: %v\n", details.WaitingToolName, err)
							s.spinner.Start()
						}
					}
				}

				// Ensure result is not empty to satisfy backend validation
				if result == "" {
					result = "[no output]"
				}

				if err := s.nubiClient.SubmitClientToolResult(ctx, details.WaitingToolCallID, s.waitingAgentID, s.waitingMessageID, result, resultStatus); err != nil {
					if strings.Contains(err.Error(), "conversation is currently in progress") {
						// If server is already processing, mark as handled locally to stop looping
						s.processedToolCallIDs[toolKey] = true
						s.toolResults[toolKey] = toolResult{result: result, status: resultStatus, timestamp: time.Now()}
					} else {
						log.Errorf("Error submitting client tool result: %v\n", err)
					}
				} else {
					s.processedToolCallIDs[toolKey] = true
					s.toolResults[toolKey] = toolResult{result: result, status: resultStatus, timestamp: time.Now()}
				}
				continue
			}

			if details.Status == "WAITING" && details.FollowupMessageConfig != "" {
				var msgConfig MessageConfig
				if err := json.Unmarshal([]byte(details.FollowupMessageConfig), &msgConfig); err == nil {
					var builder strings.Builder
					builder.WriteString(msgConfig.Question)
					builder.WriteString("\n\n")
					builder.WriteString("Options:\n")
					for _, opt := range msgConfig.FollowupOptions {
						builder.WriteString(fmt.Sprintf("- %s\n", opt))
					}
					return builder.String(), details.Status, nil
				}
			}

			if details.Status != "IN_PROGRESS" && details.Status != "WAITING" && details.Status != "WAITING_FOR_CLIENT_TOOL" {
				return details.FinalResponse, details.Status, nil
			}
		}
	}
}

func (s *nubiShell) triggerAndPoll(ctx context.Context, query string) (string, string, error) {
	err := s.nubiClient.TriggerInvestigation(ctx, query, s.disabledTools)
	if err != nil {
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

func init() {
	rootCmd.AddCommand(nubiCmd)
	defaultDisabled := []string{"server", "aws", "azure", "gcp", "kubectl", "github"}
	nubiCmd.PersistentFlags().StringSlice("disabled-tools", defaultDisabled, "List of server-side tools to disable for this session")
	nubiCmd.PersistentFlags().StringSlice("disabled-server-tools", defaultDisabled, "Alias for --disabled-tools")
	nubiCmd.PersistentFlags().Bool("enable-local-tools", false, "Enable local tool execution (default: false)")
	_ = viper.BindPFlag("disabled-tools", nubiCmd.PersistentFlags().Lookup("disabled-tools"))
	_ = viper.BindPFlag("disabled-server-tools", nubiCmd.PersistentFlags().Lookup("disabled-server-tools"))
	_ = viper.BindPFlag("enable-local-tools", nubiCmd.PersistentFlags().Lookup("enable-local-tools"))
}

func lenientUnmarshal(input string, toolName string) (map[string]any, error) {
	var args map[string]any
	// 1. Try JSON
	if err := json.Unmarshal([]byte(input), &args); err == nil {
		return args, nil
	}

	// 2. Try YAML
	if err := yaml.Unmarshal([]byte(input), &args); err == nil {
		return args, nil
	}

	// 3. Lenient fallback for local_write_file with non-indented content
	if toolName == "local_write_file" {
		lines := strings.Split(input, "\n")
		res := make(map[string]any)
		contentFound := false

		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}

			// If we haven't found content yet, look for path/filename/mode/search
			if !contentFound {
				if strings.HasPrefix(trimmed, "path:") || strings.HasPrefix(trimmed, "filename:") {
					parts := strings.SplitN(trimmed, ":", 2)
					res["path"] = strings.TrimSpace(parts[1])
					continue
				}
				if strings.HasPrefix(trimmed, "mode:") {
					parts := strings.SplitN(trimmed, ":", 2)
					res["mode"] = strings.TrimSpace(parts[1])
					continue
				}
				if strings.HasPrefix(trimmed, "search:") {
					parts := strings.SplitN(trimmed, ":", 2)
					res["search"] = strings.TrimSpace(parts[1])
					continue
				}
				if strings.HasPrefix(trimmed, "content:") {
					// Check if there's anything on the same line
					parts := strings.SplitN(line, ":", 2)
					firstPart := strings.TrimSpace(parts[1])
					// If it's just "content:" or "content: |" or "content: >", we ignore firstPart if it's just the indicator
					if firstPart == "|" || firstPart == ">" {
						firstPart = ""
					}

					remaining := strings.Join(lines[i+1:], "\n")
					if firstPart != "" {
						if remaining != "" {
							res["content"] = firstPart + "\n" + remaining
						} else {
							res["content"] = firstPart
						}
					} else {
						res["content"] = remaining
					}
					contentFound = true
					break
				}
			}
		}

		if res["path"] != "" && contentFound {
			return res, nil
		}
	}

	return nil, fmt.Errorf("failed to parse arguments as JSON or YAML")
}
