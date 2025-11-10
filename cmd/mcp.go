package cmd

import (
	"bytes"
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
	"github.com/spf13/pflag"
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

// GenericToolInput represents the input for a generic nbctl command tool.
type GenericToolInput struct {
	Flags map[string]interface{} `json:"flags"`
	Args  []string               `json:"args"`
}

// GenericToolOutput represents the output from a generic nbctl command tool.
type GenericToolOutput struct {
	Output string `json:"output"`
	Error  string `json:"error,omitempty"`
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

		registerCommands(rootCmd, server, logger)

		logger.Println("MCP server started, waiting for requests...")
		if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
			return fmt.Errorf("MCP server exited with error: %w", err)
		}
		return nil
	},
}

func registerCommands(cmd *cobra.Command, server *mcp.Server, logger *log.Logger) {
	for _, c := range cmd.Commands() {
		if c.Name() == "mcp" || c.Name() == "nubi" {
			continue
		}

		// Only register the command as a tool if it's runnable.
		if c.Runnable() {
			toolName := strings.ReplaceAll(c.CommandPath(), "nbctl ", "")
			toolName = strings.ReplaceAll(toolName, " ", "_")

			mcp.AddTool(server, &mcp.Tool{
				Name:        toolName,
				Description: c.Short,
			}, createHandler(c, logger))

			logger.Printf("Registered tool: %s", toolName)
		}

		// Always recurse into subcommands, even if the parent is not runnable.
		if c.HasSubCommands() {
			registerCommands(c, server, logger)
		}
	}
}

func createHandler(cmd *cobra.Command, logger *log.Logger) func(context.Context, *mcp.CallToolRequest, GenericToolInput) (*mcp.CallToolResult, GenericToolOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input GenericToolInput) (*mcp.CallToolResult, GenericToolOutput, error) {
		originalOut := cmd.OutOrStdout()
		originalErr := cmd.ErrOrStderr()
		originalFlags := make(map[string]string)

		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			originalFlags[f.Name] = f.Value.String()
		})

		defer func() {
			cmd.SetOut(originalOut)
			cmd.SetErr(originalErr)
			for name, value := range originalFlags {
				if err := cmd.Flags().Set(name, value); err != nil {
					logger.Printf("error restoring flag %s: %v", name, err)
				}
			}
		}()

		var outBuf, errBuf bytes.Buffer
		cmd.SetOut(&outBuf)
		cmd.SetErr(&errBuf)

		// Set flags
		for k, v := range input.Flags {
			if err := cmd.Flags().Set(k, fmt.Sprintf("%v", v)); err != nil {
				return nil, GenericToolOutput{Error: fmt.Sprintf("error setting flag %s: %v", k, err)}, nil
			}
		}

		// Execute the command
		err := cmd.RunE(cmd, input.Args)
		if err != nil {
			errBuf.WriteString(err.Error())
		}

		return nil, GenericToolOutput{
			Output: outBuf.String(),
			Error:  errBuf.String(),
		}, nil
	}
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
