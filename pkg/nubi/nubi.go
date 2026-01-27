package nubi

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/nubi/tools"
)

type NubiClient struct {
	Client           *client.Client
	AccountID        string
	Username         string
	SessionID        string
	ConversationID   string
	Endpoint         string
	EnableLocalTools bool
}

func New(client *client.Client, accountID, username, sessionID, endpoint string) *NubiClient {
	return &NubiClient{
		Client:    client,
		AccountID: accountID,
		Username:  username,
		SessionID: sessionID,
		Endpoint:  endpoint,
	}
}

type ConversationMessage struct {
	Role        string `json:"role"`
	Response    string `json:"response"`
	Message     string `json:"message"`
	MessageType string `json:"message_type"`
}

func (c *NubiClient) GetConversationMessages(conversationID string) ([]ConversationMessage, error) {
	req := client.NewRequest(`
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
	req := client.NewRequest(`
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
	req := client.NewRequest(`
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

func (c *NubiClient) TriggerInvestigation(ctx context.Context, query string, disabledTools []string) error {
	var toolsList []map[string]any
	if c.EnableLocalTools {
		var err error
		toolsList, err = tools.GetLocalToolsJSON()
		if err != nil {
			return fmt.Errorf("failed to get local tools: %w", err)
		}
	}

	req := client.NewRequest(`
		mutation AiTriggerInvestigateResponse($accountId: String!, $query: String!, $sessionId: String!, $clientTools: [AIConfigClientToolInput], $userId: String!, $capabilities: AIConfigCapabilitiesInput) {
		  ai_trigger_investigation(request: {
			account_id: $accountId,
			query: $query,
			user_id: $userId,
			session_id: $sessionId,
			async: true,
			client_tools: $clientTools,
			capabilities: $capabilities
		  }) {
			data {
			  agent_step_response
			  response
			  query
			  chain_name
			  session_id
			  conversation_id
			}
		  }
		}
	`)

	req.Var("accountId", c.AccountID)
	req.Var("query", query)
	req.Var("sessionId", c.SessionID)
	req.Var("clientTools", toolsList)
	req.Var("userId", "")
	req.Var("capabilities", map[string]any{
		"disabled_tools": disabledTools,
	})

	var respData struct {
		AiTriggerInvestigation struct {
			Data struct {
				SessionID      string `json:"session_id"`
				ConversationID string `json:"conversation_id"`
			} `json:"data"`
		} `json:"ai_trigger_investigation"`
	}

	if err := c.Client.Run(ctx, req, &respData); err != nil {
		return err
	}

	if respData.AiTriggerInvestigation.Data.SessionID != "" {
		c.SessionID = respData.AiTriggerInvestigation.Data.SessionID
	}
	if respData.AiTriggerInvestigation.Data.ConversationID != "" {
		c.ConversationID = respData.AiTriggerInvestigation.Data.ConversationID
	}

	return nil
}

type ToolCallDetail struct {
	ID string

	UUID string

	Name string

	Args string

	AgentID string

	MessageID string
}

type ConversationDetails struct {
	FinalResponse string

	Status string

	StatusText string

	FollowupMessageConfig string

	WaitingMessageID string

	WaitingAgentID string

	WaitingToolCallID string

	WaitingToolUUID string

	WaitingToolName string

	WaitingToolArgs string

	PendingToolCalls []ToolCallDetail
}

func (c *NubiClient) GetConversation(ctx context.Context) (*ConversationDetails, error) {
	if c.ConversationID != "" {
		req := client.NewRequest(`
			query GetLlmConversationDetails($id: uuid!) {
			  llm_conversations_by_pk(id: $id) {
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
					agent_step_response
					agent_name
					status
					llm_conversation_tool_calls(order_by: {created_at: asc}) {
					  id
					  tool_id
					  tool_name
					  parameters
					  status
					}
				  }
				}
			  }
			}
		`)
		req.Var("id", c.ConversationID)

		var respData struct {
			LlmConversationsByPk struct {
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
						AgentStepResponse        string `json:"agent_step_response"`
						LlmConversationToolCalls []struct {
							ID         string `json:"id"`
							ToolID     string `json:"tool_id"`
							ToolName   string `json:"tool_name"`
							Parameters string `json:"parameters"`
						} `json:"llm_conversation_tool_calls"`
					} `json:"llm_conversation_agents"`
				} `json:"llm_conversation_messages"`
			} `json:"llm_conversations_by_pk"`
		}

		if err := c.Client.Run(ctx, req, &respData); err != nil {
			return nil, err
		}

		if respData.LlmConversationsByPk.ID != "" {
			return c.processConversationData(respData.LlmConversationsByPk.Status, respData.LlmConversationsByPk.LlmConversationMessages)
		}
		// If ID lookup failed, fall back to SessionID
	}

	req := client.NewRequest(`
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
				agent_step_response
				agent_name
				status
				llm_conversation_tool_calls(order_by: {created_at: asc}) {
				  id
				  tool_id
				  tool_name
				  parameters
				  status
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
					AgentStepResponse        string `json:"agent_step_response"`
					LlmConversationToolCalls []struct {
						ID         string `json:"id"`
						ToolID     string `json:"tool_id"`
						ToolName   string `json:"tool_name"`
						Parameters string `json:"parameters"`
					} `json:"llm_conversation_tool_calls"`
				} `json:"llm_conversation_agents"`
			} `json:"llm_conversation_messages"`
		} `json:"llm_conversations"`
	}

	if err := c.Client.Run(ctx, req, &respData); err != nil {
		return nil, err
	}

	if len(respData.LlmConversations) == 0 {
		return &ConversationDetails{Status: "IN_PROGRESS"}, nil // Not ready yet
	}

	conv := respData.LlmConversations[0]
	c.ConversationID = conv.ID

	return c.processConversationData(conv.Status, conv.LlmConversationMessages)
}

func (c *NubiClient) processConversationData(status string, messages []struct {
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
		AgentStepResponse        string `json:"agent_step_response"`
		LlmConversationToolCalls []struct {
			ID         string `json:"id"`
			ToolID     string `json:"tool_id"`
			ToolName   string `json:"tool_name"`
			Parameters string `json:"parameters"`
		} `json:"llm_conversation_tool_calls"`
	} `json:"llm_conversation_agents"`
}) (*ConversationDetails, error) {
	details := &ConversationDetails{
		Status: status,
	}

	// Prefer agent_step_response_data for tool info if status is correct
	if status == "WAITING_FOR_CLIENT_TOOL" {
		type toolRequest struct {
			ToolID    string `json:"tool_id"`
			ToolName  string `json:"tool_name"`
			ToolInput any    `json:"tool_input"`
		}
		var agentToolRequests []toolRequest
		var agentID, messageID string
		var agentToolCalls []struct {
			ID         string `json:"id"`
			ToolID     string `json:"tool_id"`
			ToolName   string `json:"tool_name"`
			Parameters string `json:"parameters"`
		}

		// get recent agent tool request
		for i := len(messages) - 1; i >= 0; i-- {
			msg := messages[i]
			if msg.MessageType == "generation" {
				for _, agent := range msg.LlmConversationAgents {
					if agent.Status == "waiting_for_client_tool" {
						if agent.AgentStepResponse != "" {
							_ = json.Unmarshal([]byte(agent.AgentStepResponse), &agentToolRequests)
							agentID = agent.ID
							messageID = msg.ID
							agentToolCalls = agent.LlmConversationToolCalls
						}
						break
					}
				}
			}
			if len(agentToolRequests) > 0 {
				break
			}
		}

		for i, toolData := range agentToolRequests {
			var argsStr string
			switch v := toolData.ToolInput.(type) {
			case string:
				var args map[string]any
				if primaryArg := tools.GetPrimaryArgument(toolData.ToolName); primaryArg != "" {
					args = map[string]any{primaryArg: v}
					b, _ := json.Marshal(args)
					argsStr = string(b)
				} else {
					argsStr = v
				}
			case map[string]any:
				b, _ := json.Marshal(v)
				argsStr = string(b)
			}

			// Use ShortID for backend, but UUID for tracking if available
			toolUUID := toolData.ToolID
			if len(agentToolCalls) == len(agentToolRequests) {
				toolUUID = agentToolCalls[i].ID
			}

			details.PendingToolCalls = append(details.PendingToolCalls, ToolCallDetail{
				ID:        toolData.ToolID, // "E1"
				UUID:      toolUUID,        // UUID
				Name:      toolData.ToolName,
				Args:      argsStr,
				AgentID:   agentID,
				MessageID: messageID,
			})
		}

		if len(details.PendingToolCalls) > 0 {
			lastTool := details.PendingToolCalls[len(details.PendingToolCalls)-1]
			details.WaitingToolCallID = lastTool.ID
			details.WaitingToolUUID = lastTool.UUID
			details.WaitingToolName = lastTool.Name
			details.WaitingToolArgs = lastTool.Args
		}
	}

	if len(messages) == 0 {
		return details, nil
	}

	var latestAgentName, latestToolName, latestToolParams string
	var latestAgentStepResponse string
	if len(messages) > 0 {
		for i := len(messages) - 1; i >= 0; i-- {
			msg := messages[i]
			if msg.MessageType == "generation" {
				details.FinalResponse = msg.Response
				if msg.Status == "WAITING" || msg.Status == "WAITING_FOR_CLIENT_TOOL" {
					details.WaitingMessageID = msg.ID
					for _, a := range msg.LlmConversationAgents {
						if strings.EqualFold(a.Status, "waiting") || strings.EqualFold(a.Status, "waiting_for_client_tool") {
							details.WaitingAgentID = a.ID
						}
						// Fallback if AgentStepResponseData wasn't used or was missing
						if details.WaitingToolCallID == "" && (strings.EqualFold(a.Status, "waiting_for_client_tool") || strings.EqualFold(a.Status, "waiting")) {
							if len(a.LlmConversationToolCalls) > 0 {
								lastTool := a.LlmConversationToolCalls[len(a.LlmConversationToolCalls)-1]
								details.WaitingToolCallID = lastTool.ID
								details.WaitingToolName = lastTool.ToolName
								details.WaitingToolArgs = lastTool.Parameters
							}
						}
					}
				}
				break
			}
		}

		if details.FinalResponse == "" && len(messages) > 0 {
			lastMessage := messages[len(messages)-1]
			agents := lastMessage.LlmConversationAgents
			if len(agents) > 0 {
				latestAgent := agents[len(agents)-1]
				latestAgentName = latestAgent.AgentName
				var steps []any
				if latestAgent.AgentStepResponse != "" {
					_ = json.Unmarshal([]byte(latestAgent.AgentStepResponse), &steps)
				}
				if len(steps) > 0 {
					lastStep := steps[len(steps)-1]
					if s, ok := lastStep.(string); ok {
						latestAgentStepResponse = s
					} else {
						b, _ := json.Marshal(lastStep)
						latestAgentStepResponse = string(b)
					}
				}
				if details.FinalResponse == "" {
					details.FinalResponse = latestAgent.Response
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

	if details.WaitingAgentID != "" {
		for _, msg := range messages {
			if msg.MessageType == "followup" && msg.ParentAgentID == details.WaitingAgentID {
				details.FollowupMessageConfig = msg.MessageConfig
				break
			}
		}
	}

	if latestAgentStepResponse != "" {
		details.StatusText = latestAgentStepResponse
	} else if latestAgentName != "" {
		details.StatusText = fmt.Sprintf("Agent: %s, Tool: %s, Action: %s", latestAgentName, latestToolName, latestToolParams)
	} else if status == "WAITING_FOR_CLIENT_TOOL" && details.WaitingToolName != "" {
		details.StatusText = fmt.Sprintf("Waiting for local tool: %s", details.WaitingToolName)
	} else if status == "IN_PROGRESS" {
		details.StatusText = "Nubi is thinking..."
	} else {
		// Format status: IN_PROGRESS -> In progress
		s := strings.ReplaceAll(status, "_", " ")
		if len(s) > 0 {
			details.StatusText = strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
		} else {
			details.StatusText = s
		}
	}

	return details, nil
}

func (c *NubiClient) SubmitClientToolResult(ctx context.Context, toolID, agentID, messageID, result, status string) error {
	req := client.NewRequest(`
		mutation SubmitClientToolCallResponse($request: AISubmitClientToolCallRequest!) {
		  ai_submit_client_tool_call_response(request: $request) {
			data {
			  status
			}
		  }
		}
	`)

	req.Var("request", map[string]any{
		"account_id":      c.AccountID,
		"conversation_id": c.ConversationID,
		"agent_id":        agentID,
		"message_id":      messageID,
		"async":           true,
		"results": []map[string]any{
			{
				"tool_id": toolID,
				"result":  result,
				"status":  status,
			},
		},
	})

	var respData any
	return c.Client.Run(ctx, req, &respData)
}

func (c *NubiClient) SendFollowupResponse(ctx context.Context, query, agentID, messageID string) error {
	req := client.NewRequest(`
		mutation AiFollowupResponse($accountId: String!, $query: String!, $userId: String!, $conversationId: String!, $agentId: String!, $messageId: String!) {
		  ai_followup_response(request: {account_id: $accountId, query: $query, user_id: $userId, conversation_id: $conversationId, agent_id: $agentId, message_id: $messageId, async: true}) {
			data {
			  status
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

	req := client.NewRequest(`
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
	req := client.NewRequest(`
		mutation GetConversationUsageMetrics($accountId: String!, $conversationId: String!) {
		  ai_get_conversation_usage_metrics(request: {account_id: $accountId, conversation_id: $conversationId}) {
			data
		  }
		}
	`)

	req.Var("accountId", c.AccountID)
	req.Var("conversationId", c.ConversationID)

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
	req := client.NewRequest(`
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
	req := client.NewRequest(`
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
	req := client.NewRequest(`
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
	req := client.NewRequest(`
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
	req := client.NewRequest(`
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
	req := client.NewRequest(`
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
