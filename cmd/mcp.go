package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"nudgebee.com/nbctl/pkg/client"
	"nudgebee.com/nbctl/pkg/format"
	"nudgebee.com/nbctl/pkg/nubi"
)

// NubiToolInput represents the input for the Nubi tool.
type NubiToolInput struct {
	Query string `json:"query" jsonschema:"The user's question or request in human-readable form."`
}

// NubiToolOutput represents the output from the Nubi tool.
type NubiToolOutput struct {
	Response json.RawMessage `json:"response" jsonschema:"The response from the Nubi agent, formatted as a JSON string."`
}

// GenericToolInput represents the input for a generic nbctl command tool.
type GenericToolInput struct {
	Flags map[string]any `json:"flags"`
	Args  []string       `json:"args"`
}

// GenericToolOutput represents the output from a generic nbctl command tool.
type GenericToolOutput struct {
	Output json.RawMessage `json:"output"`
	Error  string          `json:"error,omitempty"`
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
						trimmedResp := strings.TrimSpace(resp)
						var finalResponse json.RawMessage
						if trimmedResp == "" {
							finalResponse = json.RawMessage("null")
						} else if json.Valid([]byte(trimmedResp)) {
							finalResponse = json.RawMessage(trimmedResp)
						} else {
							// Not valid JSON, so marshal it as a JSON string.
							marshaledString, _ := json.Marshal(trimmedResp)
							finalResponse = json.RawMessage(marshaledString)
						}
						return nil, NubiToolOutput{Response: finalResponse}, nil
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
		if c.Name() == "mcp" || c.Name() == "nubi" || c.Name() == "completion" || c.Name() == "configure" || c.Name() == "help" || c.Name() == "version" {
			continue
		}

		// Only register the command as a tool if it's runnable.
		if c.Runnable() {
			toolName := strings.ReplaceAll(c.CommandPath(), "nbctl ", "")
			toolName = strings.ReplaceAll(toolName, " ", "_")
			toolName = strings.ReplaceAll(toolName, "-", "_")

			inputSchema, err := generateInputSchema(c)
			if err != nil {
				logger.Printf("error generating schema for command %s: %v", toolName, err)
				continue
			}

			mcp.AddTool(server, &mcp.Tool{
				Name:        toolName,
				Description: c.Short,
				InputSchema: inputSchema,
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
		// Backup original state
		originalOut := cmd.OutOrStdout()
		originalErr := cmd.ErrOrStderr()
		originalFlags := make(map[string]string)
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			originalFlags[f.Name] = f.Value.String()
		})

		formatter := format.GetFormat()
		originalFormat := formatter.Get()
		originalFormatterWriter := cmd.OutOrStdout() // Default is os.Stdout, which this should resolve to

		// Defer restoration of original state
		defer func() {
			cmd.SetOut(originalOut)
			cmd.SetErr(originalErr)
			for name, value := range originalFlags {
				if err := cmd.Flags().Set(name, value); err != nil {
					logger.Printf("error restoring flag %s: %v", name, err)
				}
			}
			formatter.Set(originalFormat)
			formatter.SetOutput(originalFormatterWriter)
		}()

		// Hijack output for this execution
		var outBuf, errBuf bytes.Buffer
		cmd.SetOut(&outBuf)
		cmd.SetErr(&errBuf)
		formatter.Set("json")
		formatter.SetOutput(&outBuf)

		// Set flags
		for k, v := range input.Flags {
			// Skip nil flags that might come from the model
			if v == nil {
				continue
			}
			if err := cmd.Flags().Set(k, fmt.Sprintf("%v", v)); err != nil {
				// Log the error but don't fail the whole command
				logger.Printf("error setting flag %s: %v", k, err)
			}
		}

		// Execute the command
		err := cmd.RunE(cmd, input.Args)
		if err != nil {
			errBuf.WriteString(err.Error())
		}

		outputStr := outBuf.String()
		var finalOutput json.RawMessage

		trimmedOutput := strings.TrimSpace(outputStr)
		if trimmedOutput == "" {
			finalOutput = json.RawMessage("null")
		} else if json.Valid([]byte(trimmedOutput)) {
			finalOutput = json.RawMessage(trimmedOutput)
		} else {
			// If it's not valid JSON, wrap it in a structured object.
			wrappedOutput, _ := json.Marshal(map[string]string{"Data": trimmedOutput})
			finalOutput = json.RawMessage(wrappedOutput)
		}

		return nil, GenericToolOutput{
			Output: finalOutput,
			Error:  strings.TrimSpace(errBuf.String()),
		}, nil
	}
}

// JSONSchema represents a basic JSON schema.
type JSONSchema struct {
	Type        string                `json:"type"`
	Properties  map[string]*JSONSchema `json:"properties,omitempty"`
	Items       *JSONSchema           `json:"items,omitempty"`
	Description string                `json:"description,omitempty"`
	Default     any                   `json:"default,omitempty"`
}

func generateInputSchema(cmd *cobra.Command) (json.RawMessage, error) {
	flagProperties := make(map[string]*JSONSchema)
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		var schemaType string
		switch f.Value.Type() {
		case "string", "stringSlice":
			schemaType = "string"
		case "int", "intSlice":
			schemaType = "integer"
		case "bool", "boolSlice":
			schemaType = "boolean"
		case "float64", "float64Slice":
			schemaType = "number"
		default:
			schemaType = "string" // Default to string for unknown types
		}

		// Correctly type the default value
		var defaultValue any
		var err error
		switch schemaType {
		case "integer":
			defaultValue, err = strconv.Atoi(f.DefValue)
			if err != nil {
				defaultValue = 0 // Or some other sensible default
			}
		case "boolean":
			defaultValue, err = strconv.ParseBool(f.DefValue)
			if err != nil {
				defaultValue = false
			}
		case "number":
			defaultValue, err = strconv.ParseFloat(f.DefValue, 64)
			if err != nil {
				defaultValue = 0.0
			}
		default:
			defaultValue = f.DefValue
		}

		flagProperties[f.Name] = &JSONSchema{
			Type:        schemaType,
			Description: f.Usage,
			Default:     defaultValue,
		}
	})

	schema := JSONSchema{
		Type: "object",
		Properties: map[string]*JSONSchema{
			"flags": {
				Type:       "object",
				Properties: flagProperties,
			},
			"args": {
				Type: "array",
				Items: &JSONSchema{
					Type: "string",
				},
				Description: "Positional arguments for the command.",
			},
		},
	}

	return json.Marshal(schema)
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}