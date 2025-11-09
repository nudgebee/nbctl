package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time" // Re-add time import

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"nudgebee.com/nbctl/pkg/client"
	"nudgebee.com/nbctl/pkg/nubi"
)

// NubiToolInput represents the input for the Nubi tool.
type NubiToolInput struct {
	Query string `json:"query" jsonschema:"The user's question or request in human-readable form."`
}

// NubiToolOutput represents the output from the Nubi tool.
type NubiToolOutput struct {
	Response any `json:"response" jsonschema:"The response from the Nubi agent, formatted as Markdown."`
}

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run the Model-Context-Protocol (MCP) server",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := log.New(os.Stderr, "[mcp-server] ", log.LstdFlags)
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
			uuid.New().String(), // Initialize SessionID once
			viper.GetString("endpoint"),
		)

		server := mcp.NewServer(&mcp.Implementation{Name: "nubi-mcp-server", Version: "v1.0.0"}, nil)

		handler := func(ctx context.Context, req *mcp.CallToolRequest, input NubiToolInput) (
			*mcp.CallToolResult, NubiToolOutput, error,
		) {
			logger.Printf("Invoking nubi with query: %s (SessionID: %s)", input.Query, nubiClient.SessionID)

			if err := nubiClient.TriggerInvestigation(ctx, input.Query); err != nil {
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
							return nil, NubiToolOutput{Response: result}, nil
						}
						return nil, NubiToolOutput{Response: resp}, nil
					}
				}
			}
		}

		mcp.AddTool(server, &mcp.Tool{
			Name:        "nubi",
			Description: "Interact with the Nudgebee to troubleshoot Cluster, get optimizations & Recent issues.",
		}, handler)
		logger.Printf("Registered tool: nubi")

		logger.Println("MCP server started, waiting for requests...")
		if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
			return fmt.Errorf("MCP server exited with error: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
