package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
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
	Query string `json:"query" jsonschema:"the query to send to Nubi"`
}

// NubiToolOutput represents the output from the Nubi tool.
type NubiToolOutput struct {
	Response string `json:"response" jsonschema:"the response from Nubi"`
}

// NubiTool implements the MCP tool interface for Nubi agents.
type NubiTool struct {
	nubiClient *nubi.NubiClient
	logger     *log.Logger
}

// Invoke handles the invocation of the Nubi tool.
func (t *NubiTool) Invoke(ctx context.Context, req *mcp.CallToolRequest, input NubiToolInput) (
	*mcp.CallToolResult, NubiToolOutput, error,
) {
	t.logger.Printf("Invoking Nubi with query: %s", input.Query)

	// Create a new session for each request
	sessionID := uuid.New().String()
	t.nubiClient.SessionID = sessionID

	if err := t.nubiClient.TriggerInvestigation(ctx, input.Query); err != nil {
		return nil, NubiToolOutput{}, fmt.Errorf("failed to trigger investigation: %w", err)
	}

	// Poll for the result
	for {
		select {
		case <-ctx.Done():
			return nil, NubiToolOutput{}, ctx.Err()
		case <-time.After(2 * time.Second):
			resp, status, _, _, _, _, err := t.nubiClient.GetConversation(ctx)
			if err != nil {
				return nil, NubiToolOutput{}, err
			}

			if status != "IN_PROGRESS" && status != "WAITING" {
				// Try to unmarshal the response as JSON
				var result any
				if err := json.Unmarshal([]byte(resp), &result); err == nil {
					return nil, NubiToolOutput{Response: fmt.Sprintf("%v", result)}, nil
				}
				// If it's not JSON, return as a string
				return nil, NubiToolOutput{Response: resp}, nil
			}
		}
	}
}

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run the Model-Context-Protocol (MCP) server",
	Long: `Run the Model-Context-Protocol (MCP) server to interact with Nubi agents.
This command listens for JSON-RPC 2.0 messages on stdin and sends responses to stdout.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
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

		nubiToolInstance := &NubiTool{
			nubiClient: nubiClient,
			logger:     log.New(os.Stderr, "[mcp-server] ", log.LstdFlags),
		}

		// Create a server with a single tool.
		server := mcp.NewServer(&mcp.Implementation{Name: "nubi-agent", Version: "v1.0.0"}, nil)
		mcp.AddTool(server, &mcp.Tool{Name: "invoke", Description: "Invoke a Nubi agent"}, nubiToolInstance.Invoke)

		// Run the server over stdin/stdout, until the client disconnects.
		nubiToolInstance.logger.Println("MCP server started")
		if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
			return fmt.Errorf("MCP server exited with error: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
