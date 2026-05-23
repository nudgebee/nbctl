package nubi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestNubiClient(handler http.HandlerFunc) (*NubiClient, func()) {
	// Wrapper handler to handle token endpoint
	wrappedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth/token" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"token": "fake-token", "expiry": 3600})
			return
		}
		// Delegate to the test-specific handler for other requests (presumably graphql)
		handler(w, r)
	})

	srv := httptest.NewServer(wrappedHandler)
	c := client.NewClient(client.WithEndpoint(srv.URL))
	nubiClient := New(c, "test-account", "test-user", "test-session", srv.URL)
	return nubiClient, srv.Close
}

func TestNubiClient_GetConversationMessages(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"ai_get_conversation_v3": map[string]any{
					"messages": []map[string]any{
						{"role": "user", "message": "hello"},
						{"role": "assistant", "response": "hi there"},
					},
				},
			},
		}
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}

	c, teardown := newTestNubiClient(handler)
	defer teardown()

	messages, err := c.GetConversationMessages("test-conversation")
	assert.NoError(t, err)
	assert.Len(t, messages, 2)
	assert.Equal(t, "user", messages[0].Role)
	assert.Equal(t, "hello", messages[0].Message)
	assert.Equal(t, "assistant", messages[1].Role)
	assert.Equal(t, "hi there", messages[1].Response)
}

func TestNubiClient_ShowHistory(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"llm_conversations": map[string]any{
					"rows": []map[string]any{
						{"id": "1", "title": "conv 1", "updated_at": "2023-01-01T12:00:00.000000"},
						{"id": "2", "title": "conv 2", "updated_at": "2023-01-02T12:00:00.000000"},
					},
				},
			},
		}
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}

	c, teardown := newTestNubiClient(handler)
	defer teardown()

	history, err := c.ShowHistory(2)
	assert.NoError(t, err)
	assert.Len(t, history, 2)
	assert.Equal(t, "1", history[0].ID)
	assert.Equal(t, "conv 1", history[0].Title)
	expectedTime, _ := time.Parse("2006-01-02T15:04:05.999999", "2023-01-01T12:00:00.000000")
	assert.Equal(t, expectedTime, history[0].UpdatedAt)
}

func TestNubiClient_TriggerInvestigation(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"ai_trigger_investigation": map[string]any{
					"data": "{}",
				},
			},
		}
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}

	c, teardown := newTestNubiClient(handler)
	defer teardown()

	err := c.TriggerInvestigation(context.Background(), "test query")
	assert.NoError(t, err)
}

func TestNubiClient_SwitchToConversation(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"ai_get_conversation_v3": map[string]any{
					"conversation": map[string]any{
						"id":         "test-conv",
						"session_id": "test-sess",
					},
					"messages": []map[string]any{},
				},
			},
		}
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}

	c, teardown := newTestNubiClient(handler)
	defer teardown()

	messages, err := c.SwitchToConversation("test-conv")
	assert.NoError(t, err)
	assert.Empty(t, messages)
	assert.Equal(t, "test-conv", c.ConversationID)
	assert.Equal(t, "test-sess", c.SessionID)
}

func TestNubiClient_GetConversation(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"ai_get_conversation_v3": map[string]any{
					"conversation": map[string]any{
						"id":     "test-conv",
						"status": "COMPLETED",
					},
					"messages": []map[string]any{
						{
							"id":           "msg-1",
							"status":       "COMPLETED",
							"response":     "Final Response",
							"message_type": "generation",
						},
					},
					"agents":     []map[string]any{},
					"tool_calls": []map[string]any{},
				},
			},
		}
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}

	c, teardown := newTestNubiClient(handler)
	defer teardown()

	finalResponse, status, _, _, _, _, err := c.GetConversation(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "Final Response", finalResponse)
	assert.Equal(t, "COMPLETED", status)
}

func TestNubiClient_SendFollowupResponse(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"ai_followup_response": map[string]any{
					"data": "{}",
				},
			},
		}
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}

	c, teardown := newTestNubiClient(handler)
	defer teardown()
	c.ConversationID = "test-conv"

	err := c.SendFollowupResponse(context.Background(), "test query", "agent-1", "msg-1")
	assert.NoError(t, err)
}

func TestNubiClient_StopConversation(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"ai_stop_investigation": map[string]any{
					"data": "{}",
				},
			},
		}
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}

	c, teardown := newTestNubiClient(handler)
	defer teardown()
	c.ConversationID = "test-conv"

	c.StopConversation()
	// No error to assert, just ensuring it doesn't panic
}

func TestNubiClient_GetUsageMetrics(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"ai_get_conversation_usage_metrics": map[string]any{
					"data": map[string]any{
						"conversation": map[string]any{
							"total_cost":          0.00123,
							"total_input_tokens":  100,
							"total_output_tokens": 200,
						},
					},
				},
			},
		}
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}

	c, teardown := newTestNubiClient(handler)
	defer teardown()

	metrics, err := c.GetUsageMetrics(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "Cost: $0.001230, Input Tokens: 100, Output Tokens: 200", metrics)
}

func TestNubiClient_AddBookmark(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"ai_save_conversation": map[string]any{
					"data": map[string]any{
						"success": true,
					},
				},
			},
		}
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}

	c, teardown := newTestNubiClient(handler)
	defer teardown()

	err := c.AddBookmark("test-conv")
	assert.NoError(t, err)
}

func TestNubiClient_RemoveBookmark(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"ai_delete_saved_conversation": map[string]any{
					"data": "{}",
				},
			},
		}
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}

	c, teardown := newTestNubiClient(handler)
	defer teardown()

	err := c.RemoveBookmark("test-conv")
	assert.NoError(t, err)
}

func TestNubiClient_ListBookmarks(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"llm_conversations": map[string]any{
					"rows": []map[string]any{
						{"id": "1", "title": "bookmark 1"},
						{"id": "2", "title": "bookmark 2"},
					},
				},
			},
		}
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}

	c, teardown := newTestNubiClient(handler)
	defer teardown()

	bookmarks, err := c.ListBookmarks()
	assert.NoError(t, err)
	assert.Len(t, bookmarks, 2)
	assert.Equal(t, "1", bookmarks[0].ID)
	assert.Equal(t, "bookmark 1", bookmarks[0].Title)
}

func TestNubiClient_ListAgents(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"ai_list_agents": map[string]any{
					"data": json.RawMessage(`[{"name":"agent1","description":"desc1"}]`),
				},
			},
		}
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}

	c, teardown := newTestNubiClient(handler)
	defer teardown()

	agents, err := c.ListAgents()
	assert.NoError(t, err)
	assert.Len(t, agents, 1)
	assert.Equal(t, "agent1", agents[0].Name)
	assert.Equal(t, "desc1", agents[0].Description)
}

func TestNubiClient_ListTools(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"ai_list_tools": map[string]any{
					"data": json.RawMessage(`[{"name":"tool1","description":"desc1"}]`),
				},
			},
		}
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}

	c, teardown := newTestNubiClient(handler)
	defer teardown()

	tools, err := c.ListTools()
	assert.NoError(t, err)
	assert.Len(t, tools, 1)
	assert.Equal(t, "tool1", tools[0].Name)
	assert.Equal(t, "desc1", tools[0].Description)
}

func TestNubiClient_ListFunctions(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"llm_functions": map[string]any{
					"rows": []map[string]any{
						{"name": "func1", "description": "desc1"},
					},
				},
			},
		}
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}

	c, teardown := newTestNubiClient(handler)
	defer teardown()

	functions, err := c.ListFunctions()
	assert.NoError(t, err)
	assert.Len(t, functions, 1)
	assert.Equal(t, "func1", functions[0].Name)
	assert.Equal(t, "desc1", functions[0].Description)
}
