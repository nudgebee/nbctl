package nubi

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nudgebee/nbctl/pkg/client"
)

type NubiClient struct {
	Client         *client.Client
	AccountID      string
	Username       string
	SessionID      string
	ConversationID string
	Endpoint       string
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
		query GetLlmConversationMessages($accountId: String!, $conversationId: String!) {
		  ai_get_conversation_v3(request: {account_id: $accountId, conversation_id: $conversationId}) {
			messages {
			  role
			  response
			  message
			  message_type
			}
		  }
		}
	`)
	req.Var("accountId", c.AccountID)
	req.Var("conversationId", conversationID)

	var respData struct {
		AiGetConversationV3 struct {
			Messages []ConversationMessage `json:"messages"`
		} `json:"ai_get_conversation_v3"`
	}

	if err := c.Client.Run(context.Background(), req, &respData); err != nil {
		return nil, err
	}

	return respData.AiGetConversationV3.Messages, nil
}

func (c *NubiClient) SwitchToConversation(conversationID string) ([]ConversationMessage, error) {
	req := client.NewRequest(`
		query GetLlmConversationDetails($accountId: String!, $id: String!) {
		  ai_get_conversation_v3(request: {account_id: $accountId, conversation_id: $id}) {
			conversation {
			  id
			  session_id
			}
			messages {
			  role
			  response
			  message
			  message_type
			}
		  }
		}
	`)
	req.Var("accountId", c.AccountID)
	req.Var("id", conversationID)

	var respData struct {
		AiGetConversationV3 struct {
			Conversation struct {
				ID        string `json:"id"`
				SessionID string `json:"session_id"`
			} `json:"conversation"`
			Messages []ConversationMessage `json:"messages"`
		} `json:"ai_get_conversation_v3"`
	}

	if err := c.Client.Run(context.Background(), req, &respData); err != nil {
		return nil, err
	}

	if respData.AiGetConversationV3.Conversation.ID == "" {
		return nil, fmt.Errorf("conversation not found")
	}

	c.ConversationID = respData.AiGetConversationV3.Conversation.ID
	c.SessionID = respData.AiGetConversationV3.Conversation.SessionID

	return respData.AiGetConversationV3.Messages, nil
}

type ConversationHistoryItem struct {
	ID        string
	Title     string
	UpdatedAt time.Time
}

func (c *NubiClient) ShowHistory(limit int) ([]ConversationHistoryItem, error) {
	req := client.NewRequest(`
		query LlmConversationHistory($limit: Int!, $where: LlmConversationListWhereRequest!) {
		  llm_conversations: ai_list_conversations(where: $where, limit: $limit, offset: 0, order_by: [{column: "updated_at", order: desc}]) {
			rows {
			  id
			  title
			  updated_at
			}
		  }
		}
	`)
	req.Var("limit", limit)
	req.Var("where", map[string]any{
		"account_id":    map[string]any{"_eq": c.AccountID},
		"source":        map[string]any{"_eq": "UserInvestigation"},
		"user_username": map[string]any{"_eq": c.Username},
	})

	var respData struct {
		LlmConversations struct {
			Rows []struct {
				ID        string `json:"id"`
				Title     string `json:"title"`
				UpdatedAt string `json:"updated_at"`
			} `json:"rows"`
		} `json:"llm_conversations"`
	}

	if err := c.Client.Run(context.Background(), req, &respData); err != nil {
		return nil, err
	}

	var history []ConversationHistoryItem
	for _, conv := range respData.LlmConversations.Rows {
		updatedAt, err := time.Parse(time.RFC3339, conv.UpdatedAt)
		if err != nil {
			updatedAt, _ = time.Parse("2006-01-02T15:04:05.999999", conv.UpdatedAt)
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
	req := client.NewRequest(`
		mutation AiTriggerInvestigateResponse($accountId: String!, $query: String!, $sessionId: String!) {
		  ai_execute_investigation(request: {account_id: $accountId, query: $query, session_id: $sessionId, async: true}) {
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
	// ai_get_conversation_v3 returns a "conversation shell + flat messages/agents/tool_calls deltas".
	// We re-correlate the flat lists by ID to preserve the previous nested-traversal behavior.
	req := client.NewRequest(`
		query GetLlmConversation($accountId: String!, $conversationId: String, $sessionId: String) {
		  ai_get_conversation_v3(request: {account_id: $accountId, conversation_id: $conversationId, session_id: $sessionId}) {
			conversation {
			  id
			  status
			}
			messages {
			  id
			  status
			  response
			  message_type
			  message_config
			  parent_agent_id
			}
			agents {
			  id
			  message_id
			  agent_name
			  status
			  response
			}
			tool_calls {
			  agent_id
			  tool_name
			  parameters
			}
		  }
		}
	`)

	req.Var("accountId", c.AccountID)
	idToUse := c.ConversationID
	if idToUse == "" {
		idToUse = c.SessionID
	}
	if idToUse != "" {
		req.Var("conversationId", idToUse)
		req.Var("sessionId", idToUse)
	}

	var respData struct {
		AiGetConversationV3 struct {
			Conversation struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"conversation"`
			Messages []struct {
				ID            string `json:"id"`
				Status        string `json:"status"`
				Response      string `json:"response"`
				MessageType   string `json:"message_type"`
				MessageConfig string `json:"message_config"`
				ParentAgentID string `json:"parent_agent_id"`
			} `json:"messages"`
			Agents []struct {
				ID        string `json:"id"`
				MessageID string `json:"message_id"`
				AgentName string `json:"agent_name"`
				Status    string `json:"status"`
				Response  string `json:"response"`
			} `json:"agents"`
			ToolCalls []struct {
				AgentID    string `json:"agent_id"`
				ToolName   string `json:"tool_name"`
				Parameters string `json:"parameters"`
			} `json:"tool_calls"`
		} `json:"ai_get_conversation_v3"`
	}

	if err := c.Client.Run(ctx, req, &respData); err != nil {
		return "", "", "", "", "", "", err
	}

	conv := respData.AiGetConversationV3
	if conv.Conversation.ID == "" {
		return "", "IN_PROGRESS", "", "", "", "", nil // Not ready yet
	}

	c.ConversationID = conv.Conversation.ID

	// Filter messages to generation/followup, matching the prior query.
	type msgT = struct {
		ID            string
		Status        string
		Response      string
		MessageType   string
		MessageConfig string
		ParentAgentID string
	}
	var messages []msgT
	for _, m := range conv.Messages {
		if m.MessageType == "generation" || m.MessageType == "followup" {
			messages = append(messages, msgT{m.ID, m.Status, m.Response, m.MessageType, m.MessageConfig, m.ParentAgentID})
		}
	}
	if len(messages) == 0 {
		return "", conv.Conversation.Status, "", "", "", "", nil
	}

	// Index agents by message_id and tool_calls by agent_id.
	agentsByMsg := map[string][]struct {
		ID        string
		AgentName string
		Status    string
		Response  string
	}{}
	for _, a := range conv.Agents {
		agentsByMsg[a.MessageID] = append(agentsByMsg[a.MessageID], struct {
			ID        string
			AgentName string
			Status    string
			Response  string
		}{a.ID, a.AgentName, a.Status, a.Response})
	}
	toolsByAgent := map[string][]struct {
		ToolName   string
		Parameters string
	}{}
	for _, t := range conv.ToolCalls {
		toolsByAgent[t.AgentID] = append(toolsByAgent[t.AgentID], struct {
			ToolName   string
			Parameters string
		}{t.ToolName, t.Parameters})
	}

	var latestAgentName, latestToolName, latestToolParams string
	var waitingMessageID, waitingAgentID string
	finalResponse := ""

	// Find the latest "generation" message for finalResponse
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.MessageType == "generation" {
			finalResponse = msg.Response
			if msg.Status == "WAITING" {
				waitingMessageID = msg.ID
				for _, a := range agentsByMsg[msg.ID] {
					if strings.EqualFold(a.Status, "waiting") {
						waitingAgentID = a.ID
					}
				}
			}
			break
		}
	}

	// Fallback: if no generation response, use the last agent of the last message.
	if finalResponse == "" {
		lastMessage := messages[len(messages)-1]
		agents := agentsByMsg[lastMessage.ID]
		if len(agents) > 0 {
			latestAgent := agents[len(agents)-1]
			latestAgentName = latestAgent.AgentName
			finalResponse = latestAgent.Response
			toolCalls := toolsByAgent[latestAgent.ID]
			if len(toolCalls) > 0 {
				latestTool := toolCalls[len(toolCalls)-1]
				latestToolName = latestTool.ToolName
				latestToolParams = latestTool.Parameters
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

	return finalResponse, conv.Conversation.Status, statusText, followupMessageConfig, waitingMessageID, waitingAgentID, nil
}

type ConversationDetails struct {
	Conversation struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	} `json:"conversation"`
	Messages  []map[string]any `json:"messages,omitempty"`
	Agents    []map[string]any `json:"agents,omitempty"`
	ToolCalls []map[string]any `json:"tool_calls,omitempty"`
}

func (c *NubiClient) GetConversationDetails(ctx context.Context) (*ConversationDetails, error) {
	idToUse := c.ConversationID
	if idToUse == "" {
		idToUse = c.SessionID
	}

	details, err := c.fetchDetails(ctx, idToUse, "")
	if (err == nil && details != nil && details.Conversation.ID != "") || idToUse == "" {
		return details, err
	}

	// Fallback: if querying by conversation_id returned empty, try session_id
	return c.fetchDetails(ctx, "", idToUse)
}

func (c *NubiClient) fetchDetails(ctx context.Context, conversationID, sessionID string) (*ConversationDetails, error) {
	req := client.NewRequest(`
		query GetLlmConversationDetails($accountId: String!, $conversationId: String, $sessionId: String) {
		  ai_get_conversation_v3(request: {account_id: $accountId, conversation_id: $conversationId, session_id: $sessionId}) {
			conversation {
			  id
			  status
			}
			messages {
			  id
			  status
			  response
			  message_type
			  message_config
			  parent_agent_id
			}
			agents {
			  id
			  message_id
			  agent_name
			  status
			  response
			}
			tool_calls {
			  agent_id
			  tool_name
			  parameters
			}
		  }
		}
	`)

	req.Var("accountId", c.AccountID)
	if conversationID != "" {
		req.Var("conversationId", conversationID)
	}
	if sessionID != "" {
		req.Var("sessionId", sessionID)
	}

	var respData struct {
		AiGetConversationV3 struct {
			Conversation struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"conversation"`
			Messages []struct {
				ID            string `json:"id"`
				Status        string `json:"status"`
				Response      string `json:"response"`
				MessageType   string `json:"message_type"`
				MessageConfig string `json:"message_config"`
				ParentAgentID string `json:"parent_agent_id"`
			} `json:"messages"`
			Agents []struct {
				ID        string `json:"id"`
				MessageID string `json:"message_id"`
				AgentName string `json:"agent_name"`
				Status    string `json:"status"`
				Response  string `json:"response"`
			} `json:"agents"`
			ToolCalls []struct {
				AgentID    string `json:"agent_id"`
				ToolName   string `json:"tool_name"`
				Parameters string `json:"parameters"`
			} `json:"tool_calls"`
		} `json:"ai_get_conversation_v3"`
	}

	if err := c.Client.Run(ctx, req, &respData); err != nil {
		return nil, err
	}

	conv := respData.AiGetConversationV3
	details := &ConversationDetails{
		Conversation: conv.Conversation,
	}

	for _, m := range conv.Messages {
		msgMap := map[string]any{
			"id":              m.ID,
			"status":          m.Status,
			"message_type":   m.MessageType,
			"parent_agent_id": m.ParentAgentID,
		}
		if m.Response != "" {
			var respAny any
			if err := json.Unmarshal([]byte(m.Response), &respAny); err == nil {
				msgMap["response"] = respAny
			} else {
				msgMap["response"] = m.Response
			}
		}
		if m.MessageConfig != "" {
			var cfgAny any
			if err := json.Unmarshal([]byte(m.MessageConfig), &cfgAny); err == nil {
				msgMap["message_config"] = cfgAny
			} else {
				msgMap["message_config"] = m.MessageConfig
			}
		}
		details.Messages = append(details.Messages, msgMap)
	}

	for _, a := range conv.Agents {
		agentMap := map[string]any{
			"id":         a.ID,
			"message_id": a.MessageID,
			"agent_name": a.AgentName,
			"status":     a.Status,
		}
		if a.Response != "" {
			var respAny any
			if err := json.Unmarshal([]byte(a.Response), &respAny); err == nil {
				agentMap["response"] = respAny
			} else {
				agentMap["response"] = a.Response
			}
		}
		details.Agents = append(details.Agents, agentMap)
	}

	for _, t := range conv.ToolCalls {
		toolMap := map[string]any{
			"agent_id":  t.AgentID,
			"tool_name": t.ToolName,
		}
		if t.Parameters != "" {
			var paramsAny any
			if err := json.Unmarshal([]byte(t.Parameters), &paramsAny); err == nil {
				toolMap["parameters"] = paramsAny
			} else {
				toolMap["parameters"] = t.Parameters
			}
		}
		details.ToolCalls = append(details.ToolCalls, toolMap)
	}

	return details, nil
}

func (c *NubiClient) SendFollowupResponse(ctx context.Context, query, agentID, messageID string) error {
	req := client.NewRequest(`
		mutation AiFollowupResponse($accountId: String!, $query: String!, $conversationId: String!, $agentId: String!, $messageId: String!) {
		  ai_get_followup_response(request: {account_id: $accountId, query: $query, conversation_id: $conversationId, agent_id: $agentId, message_id: $messageId, async: true}) {
			data {
			  response
			}
		  }
		}
	`)

	req.Var("accountId", c.AccountID)
	req.Var("query", query)
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
		mutation AiStopInvestigation($accountId: String!, $conversationId: String!) {
		  ai_cancel_investigation(request: {account_id: $accountId, conversation_id: $conversationId}) {
			data
		  }
		}
	`)

	req.Var("accountId", c.AccountID)
	req.Var("conversationId", c.ConversationID)

	go func() {
		if err := c.Client.Run(context.Background(), req, nil); err != nil {
			fmt.Fprintf(os.Stderr, "Error stopping conversation: %v\n", err)
		}
		c.ConversationID = ""
		c.SessionID = ""
	}()
}

type ConversationStats map[string]any

func (c *NubiClient) GetConversationStats(ctx context.Context, conversationID string) (ConversationStats, error) {
	req := client.NewRequest(`
		mutation GetConversationUsageMetrics($accountId: String!, $conversationId: String!) {
		  ai_get_conversation_usage_metrics(request: {account_id: $accountId, conversation_id: $conversationId}) {
			data
		  }
		}
	`)

	if conversationID == "" {
		conversationID = c.ConversationID
		if conversationID == "" {
			conversationID = c.SessionID
		}
	}
	req.Var("accountId", c.AccountID)
	req.Var("conversationId", conversationID)

	var respData struct {
		AiGetConversationUsageMetrics struct {
			Data struct {
				Conversation map[string]any `json:"conversation"`
			} `json:"data"`
		} `json:"ai_get_conversation_usage_metrics"`
	}

	if err := c.Client.Run(ctx, req, &respData); err != nil {
		return nil, err
	}

	return respData.AiGetConversationUsageMetrics.Data.Conversation, nil
}

func getFloat64FromMap(m map[string]any, keys ...string) float64 {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			switch val := v.(type) {
			case float64:
				return val
			case int:
				return float64(val)
			case int64:
				return float64(val)
			}
		}
	}
	return 0
}

func getIntFromMap(m map[string]any, keys ...string) int {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			switch val := v.(type) {
			case float64:
				return int(val)
			case int:
				return val
			case int64:
				return int(val)
			}
		}
	}
	return 0
}

func (c *NubiClient) GetUsageMetrics(ctx context.Context) (string, error) {
	stats, err := c.GetConversationStats(ctx, "")
	if err != nil {
		return "", err
	}
	cost := getFloat64FromMap(stats, "total_cost_usd", "total_cost")
	inputTokens := getIntFromMap(stats, "total_input_tokens")
	outputTokens := getIntFromMap(stats, "total_output_tokens")
	cachedTokens := getIntFromMap(stats, "total_cached_input_tokens")
	hitRate := getFloat64FromMap(stats, "total_cache_hit_rate_percentage")

	if cachedTokens > 0 {
		return fmt.Sprintf("Cost: $%.6f, Input Tokens: %d (Cached: %d, %.1f%%), Output Tokens: %d", cost, inputTokens, cachedTokens, hitRate, outputTokens), nil
	}
	return fmt.Sprintf("Cost: $%.6f, Input Tokens: %d, Output Tokens: %d", cost, inputTokens, outputTokens), nil
}

func (c *NubiClient) AddBookmark(conversationID string) error {
	req := client.NewRequest(`
		mutation SaveConversation($data: SaveLLMConversationRequest!) {
		  ai_create_saved_conversation(request: $data) {
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
		mutation DeleteSavedConversation($data: DeleteLLMConversationRequest!) {
		  ai_delete_saved_conversation(request: $data) {
			data {
			  success
			}
		  }
		}
	`)

	type DeleteLLMConversationRequest struct {
		ConversationID string `json:"conversation_id"`
	}

	req.Var("data", DeleteLLMConversationRequest{
		ConversationID: conversationID,
	})
	return c.Client.Run(context.Background(), req, nil)
}

type BookmarkItem struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

func (c *NubiClient) ListBookmarks() ([]BookmarkItem, error) {
	req := client.NewRequest(`
		query ListBookmarks($where: LlmConversationListWhereRequest!) {
		  llm_conversations: ai_list_conversations(where: $where, order_by: [{column: "updated_at", order: desc}]) {
			rows {
			  id
			  title
			}
		  }
		}
	`)
	req.Var("where", map[string]any{
		"account_id":    map[string]any{"_eq": c.AccountID},
		"source":        map[string]any{"_eq": "UserInvestigation"},
		"user_username": map[string]any{"_eq": c.Username},
		"is_saved":      map[string]any{"_eq": true},
	})

	var respData struct {
		LlmConversations struct {
			Rows []BookmarkItem `json:"rows"`
		} `json:"llm_conversations"`
	}

	if err := c.Client.Run(context.Background(), req, &respData); err != nil {
		return nil, err
	}

	return respData.LlmConversations.Rows, nil
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
			Data []AgentItem `json:"data"`
		} `json:"ai_list_agents"`
	}

	if err := c.Client.Run(context.Background(), req, &respData); err != nil {
		return nil, err
	}

	return respData.AiListAgents.Data, nil
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
			Data []ToolItem `json:"data"`
		} `json:"ai_list_tools"`
	}

	if err := c.Client.Run(context.Background(), req, &respData); err != nil {
		return nil, err
	}

	return respData.AiListTools.Data, nil
}

type FunctionItem struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Variables   []string `json:"variables"`
}

func (f *FunctionItem) UnmarshalJSON(data []byte) error {
	type Alias FunctionItem
	aux := &struct {
		*Alias
		RawVars any `json:"variables"`
	}{
		Alias: (*Alias)(f),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	switch v := aux.RawVars.(type) {
	case string:
		var slice []string
		if err := json.Unmarshal([]byte(v), &slice); err == nil {
			f.Variables = slice
		} else if v != "" {
			f.Variables = []string{v}
		}
	case []any:
		for _, item := range v {
			f.Variables = append(f.Variables, fmt.Sprintf("%v", item))
		}
	}
	return nil
}

func (c *NubiClient) ListFunctions() ([]FunctionItem, error) {
	req := client.NewRequest(`
		query GetFunctions($where: LlmFunctionsWhereRequest!) {
		  llm_functions: ai_list_functions(where: $where) {
			rows {
			  id
			  name
			  description
			  status
			  variables
			}
		  }
		}
	`)
	req.Var("where", map[string]any{
		"account_id": map[string]any{"_eq": c.AccountID},
	})

	var respData struct {
		LlmFunctions struct {
			Rows []FunctionItem `json:"rows"`
		} `json:"llm_functions"`
	}

	if err := c.Client.Run(context.Background(), req, &respData); err != nil {
		return nil, err
	}

	return respData.LlmFunctions.Rows, nil
}

func (c *NubiClient) DeleteConversation(ctx context.Context, conversationID string) error {
	req := client.NewRequest(`
		mutation DeleteLlmConversation($conversationId: String!) {
		  ai_delete_llm_conversation_by_id(request: {conversation_id: $conversationId}) {
			data
		  }
		}
	`)

	req.Var("conversationId", conversationID)

	var respData struct {
		AiDeleteLlmConversationById struct {
			Data any `json:"data"`
		} `json:"ai_delete_llm_conversation_by_id"`
	}

	return c.Client.Run(ctx, req, &respData)
}

type PlaybookItem struct {
	ID        string `json:"id"`
	AlertName string `json:"alert_name"`
	Source    string `json:"source"`
	Processor string `json:"processor"`
	UpdatedAt string `json:"updated_at"`
}

func (c *NubiClient) ListPlaybooks(ctx context.Context) ([]PlaybookItem, error) {
	req := client.NewRequest(`
		query ListPlaybooks($where: AgentPlaybookWhereRequest!) {
		  agents_list_playbooks(where: $where) {
			rows {
			  id
			  alert_name
			  source
			  processor
			  updated_at
			}
		  }
		}
	`)
	req.Var("where", map[string]any{
		"cloud_account_id": map[string]any{"_eq": c.AccountID},
	})

	var respData struct {
		AgentsListPlaybooks struct {
			Rows []PlaybookItem `json:"rows"`
		} `json:"agents_list_playbooks"`
	}

	if err := c.Client.Run(ctx, req, &respData); err != nil {
		return nil, err
	}

	return respData.AgentsListPlaybooks.Rows, nil
}
