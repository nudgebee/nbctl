package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"nudgebee.com/nbctl/pkg/client"
	"nudgebee.com/nbctl/pkg/nubi"
)

// NubiToolInput represents the input for the Nubi tool.
type NubiToolInput struct {
	Query string `json:"query" jsonschema:"the query to send to the agent"`
}

// NubiToolOutput represents the output from the Nubi tool.
type NubiToolOutput struct {
	Response any `json:"response" jsonschema:"the response from the agent"`
}

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run the Model-Context-Protocol (MCP) server",
	Long: `Run the Model-Context-Protocol (MCP) server to interact with Nubi agents.
This command listens for JSON-RPC 2.0 messages on stdin and sends responses to stdout.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := log.New(os.Stderr, "[mcp-server] ", log.LstdFlags)

		oneShot, _ := cmd.Flags().GetBool("one-shot")

		accountID := viper.GetString("account-id")
		if accountID == "" {
			return fmt.Errorf("account-id is required, please provide it as an argument or set it in your config file")
		}

		username := viper.GetString("username")
		if username == "" {
			return fmt.Errorf("username is required, please set it in your config file")
		}

		nubiClient := nubi.New(
			client.NewClient(),
			accountID,
			username,
			"", // SessionID will be set per-request
			viper.GetString("endpoint"),
		)

		agents, err := nubiClient.ListAgents()
		if err != nil {
			return fmt.Errorf("failed to list nubi agents: %w", err)
		}

		server := mcp.NewServer(&mcp.Implementation{Name: "nubi-mcp-server", Version: "v1.0.0"}, nil)

		for _, agent := range agents {
			// Capture the agent for the closure
			agent := agent
			handler := func(ctx context.Context, req *mcp.CallToolRequest, input NubiToolInput) (
				*mcp.CallToolResult, NubiToolOutput, error,
			) {
				logger.Printf("Invoking agent %q with query: %s", agent.Name, input.Query)

				// Create a new session for each request
				sessionID := uuid.New().String()
				nubiClient.SessionID = sessionID

				// The query to Nubi should include the agent name
				fullQuery := fmt.Sprintf("@%s %s", agent.Name, input.Query)

				if err := nubiClient.TriggerInvestigation(ctx, fullQuery); err != nil {
					return nil, NubiToolOutput{}, fmt.Errorf("failed to trigger investigation: %w", err)
				}

				// Poll for the result
				for {
					select {
					case <-ctx.Done():
						return nil, NubiToolOutput{}, ctx.Err()
					case <-time.After(2 * time.Second):
						resp, status, _, _, _, _, err := nubiClient.GetConversation(ctx)
						if err != nil {
							return nil, NubiToolOutput{}, err
						}

						if status != "IN_PROGRESS" && status != "WAITING" {
							var result any
							if err := json.Unmarshal([]byte(resp), &result); err == nil {
								if oneShot {
									// Give the server a moment to write the response before exiting.
									time.AfterFunc(100*time.Millisecond, func() {
										os.Exit(0)
									})
								}
								return nil, NubiToolOutput{Response: result}, nil
							}

							if oneShot {
								time.AfterFunc(100*time.Millisecond, func() {
									os.Exit(0)
								})
							}
							return nil, NubiToolOutput{Response: resp}, nil
						}
					}
				}
			}

			// The tool name should not have the "@" prefix
			toolName := strings.TrimPrefix(agent.Name, "@")
			mcp.AddTool(server, &mcp.Tool{
				Name:        toolName,
				Description: agent.Description,
			}, handler)
			logger.Printf("Registered tool: %s", toolName)
		}

		logger.Println("MCP server started, waiting for requests...")
		if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
			return fmt.Errorf("MCP server exited with error: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
	mcpCmd.Flags().Bool("one-shot", false, "Terminate the server after the first successful response")
}
