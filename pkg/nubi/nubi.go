package nubi

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/machinebox/graphql"
	"github.com/prelude-org/nbctl/pkg/nubi/tools"
)

type NubiClient struct {
	Client         *graphql.Client
	AccountID      string
	Username       string
	SessionID      string
	ConversationID string
	Endpoint       string
	ToolRegistry   map[string]tools.Tool
}

func New(client *graphql.Client, accountID, username, sessionID, endpoint string) *NubiClient {
	return &NubiClient{
		Client:    client,
		AccountID: accountID,
		Username:  username,
		SessionID: sessionID,
		Endpoint:  endpoint,
		ToolRegistry: map[string]tools.Tool{
			"shell":         &tools.ShellTool{},
			"grep":          &tools.GrepTool{},
			"readfile":      &tools.ReadFileTool{},
			"readmanyfiles": &tools.ReadManyFilesTool{},
			"search":        &tools.SearchTool{},
		},
	}
}

func (c *NubiClient) ExecuteTool(ctx context.Context, name string, args any) (string, error) {
	tool, ok := c.ToolRegistry[name]
	if !ok {
		return "", fmt.Errorf("tool not found: %s", name)
	}
	return tool.Run(ctx, args)
}

type ConversationMessage struct {
	Role        string `json:"role"`
	Response    string `json:"response"`
	Message     string `json:"message"`
	MessageType string `json:"message_type"`
}

func (c *NubiClient) GetConversationMessages(conversationID string) ([]ConversationMessage, error) {
	req := graphql.NewRequest(`
		query GetLlmConversationMessages($conversationId: uuid!) {
		  llm_conversation_messages(where: {conversation_id: {_eq: $conversationId}, message_type: {_in: ["generation", "followup"]}}, order_by: {created_at: asc}) {
			role
			response
			message
			message_type
		  }
		}
	`)
	req.Var("conversationId", conversationID)

	var respData struct {
		LlmConversationMessages []ConversationMessage `json:"llm_conversation_messages"`
	}

	if err := c.Client.Run(context.Background(), req, &respData); err != nil {
		return nil, err
	}

	return respData.LlmConversationMessages, nil
}

func (c *NubiClient) SwitchToConversation(conversationID string) ([]ConversationMessage, error) {
	req := graphql.NewRequest(`
		query GetLlmConversationDetails($id: uuid!) {
		  llm_conversations_by_pk(id: $id) {
			id
			session_id
		  }
		}
	`)
	req.Var("id", conversationID)

	var respData struct {
		LlmConversationsByPk struct {
			ID        string `json:"id"`
			SessionID string `json:"session_id"`
		} `json:"llm_conversations_by_pk"`
	}

	if err := c.Client.Run(context.Background(), req, &respData); err != nil {
		return nil, err
	}

	if respData.LlmConversationsByPk.ID == "" {
		return nil, fmt.Errorf("conversation not found")
	}

	c.ConversationID = respData.LlmConversationsByPk.ID
	c.SessionID = respData.LlmConversationsByPk.SessionID

	return c.GetConversationMessages(conversationID)
}

type ConversationHistoryItem struct {
	ID        string
	Title     string
	UpdatedAt time.Time
}

func (c *NubiClient) ShowHistory(limit int) ([]ConversationHistoryItem, error) {
	req := graphql.NewRequest(`
		query LlmConversationHistory($limit: Int!, $accountId: uuid!, $username: citext!) {
		  llm_conversations(
			where: {
			  account_id: {_eq: $accountId},
			  source: {_eq: "UserInvestigation"},
			  user: {username: {_eq: $username}}
			},
			order_by: {updated_at: desc},
			limit: $limit
		  ) {
			id
			title
			updated_at
		  }
		}
	`)
	req.Var("limit", limit)
	req.Var("accountId", c.AccountID)
	req.Var("username", c.Username)

	var respData struct {
		LlmConversations []struct {
			ID        string `json:"id"`
			Title     string `json:"title"`
			UpdatedAt string `json:"updated_at"`
		} `json:"llm_conversations"`
	}

	if err := c.Client.Run(context.Background(), req, &respData); err != nil {
		return nil, err
	}

	var history []ConversationHistoryItem
	for _, conv := range respData.LlmConversations {
		updatedAt, err := time.Parse("2006-01-02T15:04:05.999999", conv.UpdatedAt)
		if err != nil {
			return nil, err
		}
		history = append(history, ConversationHistoryItem{
			ID:        conv.ID,
			Title:     conv.Title,
			UpdatedAt: updatedAt,
		})
	}

	return history, nil
}

func (c *NubiClient) TriggerInvestigation(ctx context.Context, query string) error {
	req := graphql.NewRequest(`
		mutation AiTriggerInvestigateResponse($accountId: String!, $query: String!, $sessionId: String!) {
		  ai_trigger_investigation(request: {account_id: $accountId, query: $query, user_id: "", session_id: $sessionId, async: true}) {
			data {
			  agent_step_response
			  response
			  query
			  chain_name
			}
		  }
		}
	`)

	req.Var("accountId", c.AccountID)
	req.Var("query", query)
	req.Var("sessionId", c.SessionID)

	var respData any
	return c.Client.Run(ctx, req, &respData)
}

func (c *NubiClient) GetConversation(ctx context.Context) (string, string, string, string, string, string, error) {
	req := graphql.NewRequest(`
		query GetLlmConversation($sessionId: String!) {
		  llm_conversations(where: {session_id: {_eq: $sessionId}}, order_by: {created_at: desc}, limit: 1) {
			id
			status
			llm_conversation_messages(where: {message_type: {_in: ["generation", "followup"]}}, order_by: {created_at: asc}) {
			  id
			  status
			  response
			  message_type
			  message_config
			  parent_agent_id
			  llm_conversation_agents(order_by: {created_at: asc}) {
			    id
				response
				agent_name
				status
				llm_conversation_tool_calls(order_by: {created_at: asc}) {
				  tool_name
				  parameters
				}
			  }
			}
		  }
		}
	`)

	req.Var("sessionId", c.SessionID)

	var respData struct {
		LlmConversations []struct {
			ID                      string `json:"id"`
			Status                  string `json:"status"`
			LlmConversationMessages []struct {
				ID                    string `json:"id"`
				Status                string `json:"status"`
				Response              string `json:"response"`
				MessageType           string `json:"message_type"`
				MessageConfig         string `json:"message_config"`
				ParentAgentID         string `json:"parent_agent_id"`
				LlmConversationAgents []struct {
					ID                       string `json:"id"`
					AgentName                string `json:"agent_name"`
					Status                   string `json:"status"`
					Response                 string `json:"response"`
					LlmConversationToolCalls []struct {
						ToolName   string `json:"tool_name"`
						Parameters string `json:"parameters"`
					} `json:"llm_conversation_tool_calls"`
				} `json:"llm_conversation_agents"`
			} `json:"llm_conversation_messages"`
		} `json:"llm_conversations"`
	}

	if err := c.Client.Run(ctx, req, &respData); err != nil {
		return "", "", "", "", "", "", err
	}

	if len(respData.LlmConversations) == 0 {
		return "", "IN_PROGRESS", "", "", "", "", nil // Not ready yet
	}

	c.ConversationID = respData.LlmConversations[0].ID

	if len(respData.LlmConversations[0].LlmConversationMessages) == 0 {
		return "", respData.LlmConversations[0].Status, "", "", "", "", nil
	}

	// Find the latest agent and tool call
	var latestAgentName, latestToolName, latestToolParams string
	var waitingMessageID, waitingAgentID string
	messages := respData.LlmConversations[0].LlmConversationMessages
	finalResponse := ""
	if len(messages) > 0 {
		// Find the latest "generation" message for finalResponse
		for i := len(messages) - 1; i >= 0; i-- {
			msg := messages[i]
			if msg.MessageType == "generation" {
				finalResponse = msg.Response
				if msg.Status == "WAITING" {
					waitingMessageID = msg.ID
					for _, a := range msg.LlmConversationAgents {
						if strings.EqualFold(a.Status, "waiting") {
							waitingAgentID = a.ID
						}
					}
				}
				break
			}
		}

		// If finalResponse is still empty, try to get it from the last agent's response
		// This ensures we don't miss a response if it's not explicitly a "generation" message
		// but is part of the agent's final output.
		// This part should only execute if no generation message was found.
		if finalResponse == "" {
			lastMessage := messages[len(messages)-1] // Get the actual last message for agent info
			agents := lastMessage.LlmConversationAgents
			if len(agents) > 0 {
				latestAgent := agents[len(agents)-1]
				latestAgentName = latestAgent.AgentName
				if finalResponse == "" { // Only set if still empty
					finalResponse = latestAgent.Response
				}
				toolCalls := latestAgent.LlmConversationToolCalls
				if len(toolCalls) > 0 {
					latestTool := toolCalls[len(toolCalls)-1]
					latestToolName = latestTool.ToolName
					latestToolParams = latestTool.Parameters
				}
			}
		}
	}

	var followupMessageConfig string
	if waitingAgentID != "" {
		for _, msg := range messages {
			if msg.MessageType == "followup" && msg.ParentAgentID == waitingAgentID {
				followupMessageConfig = msg.MessageConfig
				break
			}
		}
	}

	statusText := fmt.Sprintf("Agent: %s, Tool: %s, Action: %s", latestAgentName, latestToolName, latestToolParams)

	return finalResponse, respData.LlmConversations[0].Status, statusText, followupMessageConfig, waitingMessageID, waitingAgentID, nil
}

func (c *NubiClient) SendFollowupResponse(ctx context.Context, query, agentID, messageID string) error {
	req := graphql.NewRequest(`
		mutation AiFollowupResponse($accountId: String!, $query: String!, $userId: String!, $conversationId: String!, $agentId: String!, $messageId: String!) {
		  ai_followup_response(request: {account_id: $accountId, query: $query, user_id: $userId, conversation_id: $conversationId, agent_id: $agentId, message_id: $messageId, async: true}) {
			data {
			  response
			}
		  }
		}
	`)

	req.Var("accountId", c.AccountID)
	req.Var("query", query)
	req.Var("userId", "")
	req.Var("conversationId", c.ConversationID)
	req.Var("agentId", agentID)
	req.Var("messageId", messageID)

	var respData any
	return c.Client.Run(ctx, req, &respData)
}

func (c *NubiClient) StopConversation() {
	if c.ConversationID == "" {
		return
	}

	req := graphql.NewRequest(`
		mutation AIStopInvestigationRequest($accountId: String!, $conversationId: String!, $username: String!) {
		  ai_stop_investigation(request: {account_id: $accountId, conversation_id: $conversationId, user_id: $username}) {
			data
		  }
		}
	`)

	req.Var("accountId", c.AccountID)
	req.Var("conversationId", c.ConversationID)
	req.Var("username", c.Username)

	go func() {
		if err := c.Client.Run(context.Background(), req, nil); err != nil {
			fmt.Fprintf(os.Stderr, "Error stopping conversation: %v\n", err)
		}
		c.ConversationID = ""
		c.SessionID = ""
	}()
}

func (c *NubiClient) GetUsageMetrics(ctx context.Context) (string, error) {
	req := graphql.NewRequest(`
		mutation GetConversationUsageMetrics($accountId: String!, $conversationId: String!) {
		  ai_get_conversation_usage_metrics(request: {account_id: $accountId, conversation_id: $conversationId}) {
			data
		  }
		}
	`)

	req.Var("accountId", c.AccountID)
	req.Var("conversationId", c.SessionID)

	var respData struct {
		AiGetConversationUsageMetrics struct {
			Data struct {
				Conversation struct {
					TotalCost         float64 `json:"total_cost"`
					TotalInputTokens  int     `json:"total_input_tokens"`
					TotalOutputTokens int     `json:"total_output_tokens"`
				} `json:"conversation"`
			} `json:"data"`
		} `json:"ai_get_conversation_usage_metrics"`
	}

	if err := c.Client.Run(ctx, req, &respData); err != nil {
		return "", err
	}

	metrics := respData.AiGetConversationUsageMetrics.Data.Conversation
	return fmt.Sprintf("Cost: $%.6f, Input Tokens: %d, Output Tokens: %d", metrics.TotalCost, metrics.TotalInputTokens, metrics.TotalOutputTokens), nil
}

func (c *NubiClient) AddBookmark(conversationID string) error {
	req := graphql.NewRequest(`
		mutation SaveConversation($data: SaveLLMConversationRequest!) {
		  ai_save_conversation(request: $data) {
			data {
			  success
			}
		  }
		}
	`)

	type SaveLLMConversationRequest struct {
		ConversationID string `json:"conversation_id"`
	}

	req.Var("data", SaveLLMConversationRequest{
		ConversationID: conversationID,
	})

	return c.Client.Run(context.Background(), req, nil)
}

func (c *NubiClient) RemoveBookmark(conversationID string) error {
	req := graphql.NewRequest(`
		mutation RemoveBookmark($conversationId: uuid!, $username: String!) {
		  delete_llm_conversation_saveds_by_pk(conversation_id: $conversationId, user_id: $username) {
			conversation_id
		  }
		}
	`)
	req.Var("conversationId", conversationID)
	req.Var("username", c.Username)
	return c.Client.Run(context.Background(), req, nil)
}

type BookmarkItem struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

func (c *NubiClient) ListBookmarks() ([]BookmarkItem, error) {
	req := graphql.NewRequest(`
		query ListBookmarks($username: citext!, $accountId: uuid!) {
		  llm_conversations(
			where: {
			  account_id: {_eq: $accountId},
			  source: {_eq: "UserInvestigation"},
			  llm_conversation_messages: {message_type: {_eq: "generation"}, role: {_eq: "human"}},
			  llm_conversation_saveds: {user: {username: {_eq: $username}}}
			},
			order_by: {updated_at: desc}
		  ) {
			id
			title
		  }
		}
	`)
	req.Var("username", c.Username)
	req.Var("accountId", c.AccountID)

	var respData struct {
		LlmConversations []BookmarkItem `json:"llm_conversations"`
	}

	if err := c.Client.Run(context.Background(), req, &respData); err != nil {
		return nil, err
	}

	return respData.LlmConversations, nil
}

type AgentItem struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Tools       []string `json:"tools"`
}

func (c *NubiClient) ListAgents() ([]AgentItem, error) {
	req := graphql.NewRequest(`
		query ListAgents($accountId: String!) {
		  ai_list_agents(request: {account_id: $accountId}) {
			data
		  }
		}
	`)
	req.Var("accountId", c.AccountID)

	var respData struct {
		AiListAgents struct {
			Data json.RawMessage `json:"data"`
		} `json:"ai_list_agents"`
	}

	if err := c.Client.Run(context.Background(), req, &respData); err != nil {
		return nil, err
	}

	var agents []AgentItem
	if err := json.Unmarshal(respData.AiListAgents.Data, &agents); err != nil {
		return nil, err
	}

	return agents, nil
}

type ToolItem struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	NBToolType  string `json:"nb_tool_type"`
}

func (c *NubiClient) ListTools() ([]ToolItem, error) {
	req := graphql.NewRequest(`
		query ListTools($accountId: String!) {
		  ai_list_tools(request: {account_id: $accountId}) {
			data
		  }
		}
	`)
	req.Var("accountId", c.AccountID)

	var respData struct {
		AiListTools struct {
			Data json.RawMessage `json:"data"`
		} `json:"ai_list_tools"`
	}

	if err := c.Client.Run(context.Background(), req, &respData); err != nil {
		return nil, err
	}

	var tools []ToolItem
	if err := json.Unmarshal(respData.AiListTools.Data, &tools); err != nil {
		return nil, err
	}

	return tools, nil
}

type FunctionItem struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Variables   []string `json:"variables"`
}

func (c *NubiClient) ListFunctions() ([]FunctionItem, error) {
	req := graphql.NewRequest(`
		query GetFunctions($accountId: uuid!) {
		  llm_functions(where: {account_id: {_eq: $accountId}}) {
			id
			name
			description
			status
			variables
		  }
		}
	`)
	req.Var("accountId", c.AccountID)

	var respData struct {
		LlmFunctions []FunctionItem `json:"llm_functions"`
	}

	if err := c.Client.Run(context.Background(), req, &respData); err != nil {
		return nil, err
	}

	return respData.LlmFunctions, nil
}
