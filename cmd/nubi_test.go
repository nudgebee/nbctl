package cmd

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/nudgebee/nbctl/pkg/testutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetNubiFlags() {
	if f := nubiQueryCmd.Flags().Lookup("async"); f != nil {
		_ = f.Value.Set("false")
		f.Changed = false
	}
	if f := nubiQueryCmd.Flags().Lookup("timeout"); f != nil {
		_ = f.Value.Set("0s")
		f.Changed = false
	}
	if f := nubiQueryCmd.Flags().Lookup("account-id"); f != nil {
		_ = f.Value.Set("")
		f.Changed = false
	}
	nubiQueryTimeout = 0
	if f := nubiGetCmd.Flags().Lookup("session-id"); f != nil {
		_ = f.Value.Set("")
		f.Changed = false
	}
	if f := nubiGetCmd.Flags().Lookup("account-id"); f != nil {
		_ = f.Value.Set("")
		f.Changed = false
	}
	if f := rootCmd.PersistentFlags().Lookup("format"); f != nil {
		_ = f.Value.Set("text")
		f.Changed = false
	}
	if f := rootCmd.PersistentFlags().Lookup("output"); f != nil {
		_ = f.Value.Set("")
		f.Changed = false
	}
	format.GetFormat().Set("text")
	viper.Set("username", "")
	viper.Set("account-id", "")
}

func TestNubiCmd_AsyncQuery(t *testing.T) {
	resetNubiFlags()
	viper.Set("username", "test-user")
	viper.Set("account-id", "test-account-id")
	t.Cleanup(resetNubiFlags)

	mockResponse := map[string]interface{}{
		"ai_execute_investigation": map[string]interface{}{
			"data": map[string]interface{}{
				"response": "ok",
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, nubiCmd, []string{"nubi", "query", "hello", "--async"})
	require.NoError(t, err)

	assert.Contains(t, output, "Investigation triggered asynchronously.")
	assert.Contains(t, output, "Session ID:")
}

func TestNubiCmd_AsyncQuery_TriggerError_JSON(t *testing.T) {
	resetNubiFlags()
	viper.Set("username", "test-user")
	t.Cleanup(resetNubiFlags)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/auth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{"token": "fake-token", "expiry": 3600})
		case "/api/graphql":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"errors": []map[string]any{
					{"message": "api: user does not have access"},
				},
			})
		default:
			http.NotFound(w, r)
		}
	})

	defaults := map[string]any{
		"api-key":    "dummy",
		"username":   "dummy-user",
		"account-id": "dummy-account",
	}
	output, err := testutil.RunWithMockServer(handler, defaults, nubiCmd, []string{"nubi", "query", "hello", "--async", "-o", "json"})
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err)

	assert.Equal(t, "ERROR", result["status"])
	assert.Contains(t, result["error"], "user does not have access")
	assert.Equal(t, "dummy-account", result["account_id"])
	assert.Contains(t, result["hint"], "User does not have access to account dummy-account")
}

func TestNubiCmd_Query_AccessDenied_Text(t *testing.T) {
	resetNubiFlags()
	viper.Set("username", "test-user")
	t.Cleanup(resetNubiFlags)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/auth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{"token": "fake-token", "expiry": 3600})
		case "/api/graphql":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"errors": []map[string]any{
					{"message": "api: user does not have access"},
				},
			})
		default:
			http.NotFound(w, r)
		}
	})

	defaults := map[string]any{
		"api-key":    "dummy",
		"username":   "dummy-user",
		"account-id": "dummy-account",
	}
	output, err := testutil.RunWithMockServer(handler, defaults, nubiCmd, []string{"nubi", "query", "hello", "--async"})
	require.Error(t, err)

	assert.Contains(t, err.Error(), "user does not have access")
	assert.Contains(t, err.Error(), "Hint: User does not have access to account dummy-account")
	assert.Contains(t, output, "Hint: User does not have access to account dummy-account")
}

func TestNubiCmd_EmptyQuery(t *testing.T) {
	resetNubiFlags()
	viper.Set("username", "test-user")
	viper.Set("account-id", "test-account-id")
	t.Cleanup(resetNubiFlags)

	_, err := testutil.RunWithSimpleGraphQL(nil, nubiCmd, []string{"nubi", "query", "   "})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "query cannot be empty")
}

func TestNubiCmd_SyncQuery(t *testing.T) {
	resetNubiFlags()
	viper.Set("username", "test-user")
	t.Cleanup(resetNubiFlags)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/auth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{"token": "fake-token", "expiry": 3600})
		case "/api/graphql":
			resp := map[string]interface{}{
				"data": map[string]interface{}{
					"ai_execute_investigation": map[string]interface{}{
						"data": map[string]interface{}{
							"response": "started",
						},
					},
					"ai_get_conversation_v3": map[string]interface{}{
						"conversation": map[string]interface{}{
							"id":     "conv-123",
							"status": "COMPLETED",
						},
						"messages": []map[string]interface{}{
							{
								"id":           "msg-1",
								"status":       "COMPLETED",
								"response":     "System status is healthy",
								"message_type": "generation",
							},
						},
					},
					"ai_get_conversation_usage_metrics": map[string]interface{}{
						"data": map[string]interface{}{
							"conversation": map[string]interface{}{
								"total_cost":         0.001,
								"total_input_tokens": 50,
								"total_output_tokens": 100,
							},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		default:
			http.NotFound(w, r)
		}
	})

	defaults := map[string]any{
		"api-key":    "dummy",
		"username":   "dummy-user",
		"account-id": "dummy-account",
	}
	output, err := testutil.RunWithMockServer(handler, defaults, nubiCmd, []string{"nubi", "query", "system status"})
	require.NoError(t, err)

	assert.Contains(t, output, "System status")
	assert.Contains(t, output, "healthy")
	assert.Contains(t, output, "Cost: $0.001000")
	assert.Contains(t, output, "Response time:")
}

func TestNubiCmd_SyncQuery_JSON(t *testing.T) {
	resetNubiFlags()
	viper.Set("username", "test-user")
	t.Cleanup(resetNubiFlags)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/auth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{"token": "fake-token", "expiry": 3600})
		case "/api/graphql":
			resp := map[string]interface{}{
				"data": map[string]interface{}{
					"ai_execute_investigation": map[string]interface{}{
						"data": map[string]interface{}{
							"response": "started",
						},
					},
					"ai_get_conversation_v3": map[string]interface{}{
						"conversation": map[string]interface{}{
							"id":     "conv-123",
							"status": "COMPLETED",
						},
						"messages": []map[string]interface{}{
							{
								"id":           "msg-1",
								"status":       "COMPLETED",
								"response":     "System status is healthy",
								"message_type": "generation",
							},
						},
					},
					"ai_get_conversation_usage_metrics": map[string]interface{}{
						"data": map[string]interface{}{
							"conversation": map[string]interface{}{
								"total_cost":         0.001,
								"total_input_tokens": 50,
								"total_output_tokens": 100,
							},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		default:
			http.NotFound(w, r)
		}
	})

	defaults := map[string]any{
		"api-key":    "dummy",
		"username":   "dummy-user",
		"account-id": "dummy-account",
	}
	output, err := testutil.RunWithMockServer(handler, defaults, nubiCmd, []string{"nubi", "query", "system status", "--output", "json"})
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err)

	assert.Equal(t, "dummy-account", result["account_id"])
	assert.Equal(t, "conv-123", result["conversation_id"])
	assert.Equal(t, "system status", result["query"])
	assert.Equal(t, "System status is healthy", result["response"])
	assert.Equal(t, "COMPLETED", result["status"])
	assert.NotEmpty(t, result["url"])
}

func TestNubiCmd_Query_Timeout(t *testing.T) {
	resetNubiFlags()
	viper.Set("username", "test-user")
	t.Cleanup(resetNubiFlags)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/auth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{"token": "fake-token", "expiry": 3600})
		case "/api/graphql":
			resp := map[string]interface{}{
				"data": map[string]interface{}{
					"ai_execute_investigation": map[string]interface{}{
						"data": map[string]interface{}{
							"response": "started",
						},
					},
					"ai_get_conversation_v3": map[string]interface{}{
						"conversation": map[string]interface{}{
							"id":     "conv-timeout-1",
							"status": "IN_PROGRESS",
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		default:
			http.NotFound(w, r)
		}
	})

	defaults := map[string]any{
		"api-key":    "dummy",
		"username":   "dummy-user",
		"account-id": "dummy-account",
	}
	output, err := testutil.RunWithMockServer(handler, defaults, nubiCmd, []string{"nubi", "query", "check performance", "--timeout", "100ms"})
	require.NoError(t, err)

	assert.Contains(t, output, "Query timed out after")
	assert.Contains(t, output, "The investigation was triggered and may still be running or completed server-side.")
	assert.Contains(t, output, "Conversation ID: conv-timeout-1")
	assert.Contains(t, output, "Session ID:")
	assert.Contains(t, output, "nbctl nubi get conv-timeout-1 --account-id dummy-account")
	assert.Contains(t, output, "--timeout 5m")
	assert.Contains(t, output, "--async")
}

func TestNubiCmd_Query_Timeout_JSON(t *testing.T) {
	resetNubiFlags()
	viper.Set("username", "test-user")
	t.Cleanup(resetNubiFlags)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/auth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{"token": "fake-token", "expiry": 3600})
		case "/api/graphql":
			resp := map[string]interface{}{
				"data": map[string]interface{}{
					"ai_execute_investigation": map[string]interface{}{
						"data": map[string]interface{}{
							"response": "started",
						},
					},
					"ai_get_conversation_v3": map[string]interface{}{
						"conversation": map[string]interface{}{
							"id":     "conv-timeout-2",
							"status": "IN_PROGRESS",
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		default:
			http.NotFound(w, r)
		}
	})

	defaults := map[string]any{
		"api-key":    "dummy",
		"username":   "dummy-user",
		"account-id": "dummy-account",
	}
	output, err := testutil.RunWithMockServer(handler, defaults, nubiCmd, []string{"nubi", "query", "check performance", "--timeout", "100ms", "--output", "json"})
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err)

	assert.Equal(t, "TIMED_OUT", result["status"])
	assert.Equal(t, "dummy-account", result["account_id"])
	assert.Equal(t, "conv-timeout-2", result["conversation_id"])
	assert.NotEmpty(t, result["session_id"])
	assert.Contains(t, result["url"], "conv-timeout-2")
	assert.Contains(t, result["hint"], "nbctl nubi get conv-timeout-2 --account-id dummy-account")
}

func TestNubiCmd_Query_TransientRetry(t *testing.T) {
	resetNubiFlags()
	viper.Set("username", "test-user")
	t.Cleanup(resetNubiFlags)

	pollCount := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/auth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{"token": "fake-token", "expiry": 3600})
		case "/api/graphql":
			pollCount++
			if pollCount == 2 {
				// Simulate transient 500 error on first poll
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]any{"errors": []map[string]any{{"message": "transient error"}}})
				return
			}
			resp := map[string]interface{}{
				"data": map[string]interface{}{
					"ai_execute_investigation": map[string]interface{}{
						"data": map[string]interface{}{
							"response": "started",
						},
					},
					"ai_get_conversation_v3": map[string]interface{}{
						"conversation": map[string]interface{}{
							"id":     "conv-retry-1",
							"status": "COMPLETED",
						},
						"messages": []map[string]interface{}{
							{
								"id":           "msg-1",
								"status":       "COMPLETED",
								"response":     "Recovered successfully",
								"message_type": "generation",
							},
						},
					},
					"ai_get_conversation_usage_metrics": map[string]interface{}{
						"data": map[string]interface{}{
							"conversation": map[string]interface{}{
								"total_cost": 0.001,
							},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		default:
			http.NotFound(w, r)
		}
	})

	defaults := map[string]any{
		"api-key":    "dummy",
		"username":   "dummy-user",
		"account-id": "dummy-account",
	}
	output, err := testutil.RunWithMockServer(handler, defaults, nubiCmd, []string{"nubi", "query", "status check"})
	require.NoError(t, err)
	assert.Contains(t, output, "Recovered")
	assert.Contains(t, output, "successfully")
}

func TestNubiCmd_List(t *testing.T) {
	resetNubiFlags()
	viper.Set("username", "test-user")
	viper.Set("account-id", "test-account-id")
	t.Cleanup(resetNubiFlags)

	mockResponse := map[string]interface{}{
		"llm_conversations": map[string]interface{}{
			"rows": []map[string]interface{}{
				{"id": "conv-1", "title": "First conversation", "updated_at": "2026-08-20T12:00:00Z"},
				{"id": "conv-2", "title": "Second conversation", "updated_at": "2026-08-21T12:00:00Z"},
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, nubiCmd, []string{"nubi", "list"})
	require.NoError(t, err)

	assert.Contains(t, output, "conv-1")
	assert.Contains(t, output, "First conversation")
	assert.Contains(t, output, "conv-2")
}

func TestNubiCmd_Get(t *testing.T) {
	resetNubiFlags()
	viper.Set("username", "test-user")
	viper.Set("account-id", "test-account-id")
	t.Cleanup(resetNubiFlags)

	mockResponse := map[string]interface{}{
		"ai_get_conversation_v3": map[string]interface{}{
			"conversation": map[string]interface{}{
				"id":     "conv-123",
				"status": "COMPLETED",
			},
			"messages": []map[string]interface{}{
				{
					"id":           "msg-1",
					"status":       "COMPLETED",
					"response":     "Details response",
					"message_type": "generation",
				},
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, nubiCmd, []string{"nubi", "get", "conv-123"})
	require.NoError(t, err)

	assert.Contains(t, output, "Details")
	assert.Contains(t, output, "response")
}

func TestNubiCmd_Get_JSON(t *testing.T) {
	resetNubiFlags()
	viper.Set("username", "test-user")
	viper.Set("account-id", "test-account-id")
	t.Cleanup(resetNubiFlags)

	mockResponse := map[string]interface{}{
		"ai_get_conversation_v3": map[string]interface{}{
			"conversation": map[string]interface{}{
				"id":     "conv-123",
				"status": "COMPLETED",
			},
			"messages": []map[string]interface{}{
				{
					"id":           "msg-1",
					"status":       "COMPLETED",
					"response":     "Details response",
					"message_type": "generation",
				},
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, nubiCmd, []string{"nubi", "get", "conv-123", "--output", "json"})
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err)

	assert.Equal(t, "conv-123", result["conversation_id"])
}

func TestNubiCmd_Bookmark(t *testing.T) {
	resetNubiFlags()
	viper.Set("username", "test-user")
	viper.Set("account-id", "test-account-id")
	t.Cleanup(resetNubiFlags)

	mockResponse := map[string]interface{}{
		"ai_bookmark_conversation": map[string]interface{}{
			"data": "ok",
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, nubiCmd, []string{"nubi", "bookmark", "add", "conv-123"})
	require.NoError(t, err)

	assert.Contains(t, output, "Added bookmark for conversation conv-123")
}

func TestNubiCmd_Delete(t *testing.T) {
	resetNubiFlags()
	viper.Set("username", "test-user")
	viper.Set("account-id", "test-account-id")
	t.Cleanup(resetNubiFlags)

	mockResponse := map[string]interface{}{
		"ai_delete_llm_conversation_by_id": map[string]interface{}{
			"data": "ok",
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, nubiCmd, []string{"nubi", "delete", "conv-123"})
	require.NoError(t, err)

	assert.Contains(t, output, "Deleted conversation conv-123")
}

func TestNubiCmd_Agents(t *testing.T) {
	resetNubiFlags()
	viper.Set("username", "test-user")
	viper.Set("account-id", "test-account-id")
	t.Cleanup(resetNubiFlags)

	mockResponse := map[string]interface{}{
		"ai_list_agents": map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"name":        "k8s_agent",
					"description": "Kubernetes management agent",
					"status":      "active",
					"tools":       []string{"kubectl_get", "kubectl_logs"},
				},
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, nubiCmd, []string{"nubi", "agents"})
	require.NoError(t, err)

	assert.Contains(t, output, "k8s_agent")
	assert.Contains(t, output, "Kubernetes management agent")
}

func TestNubiCmd_Tools(t *testing.T) {
	resetNubiFlags()
	viper.Set("username", "test-user")
	viper.Set("account-id", "test-account-id")
	t.Cleanup(resetNubiFlags)

	mockResponse := map[string]interface{}{
		"ai_list_tools": map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"name":         "kubectl_get",
					"description":  "Get Kubernetes resources",
					"status":       "enabled",
					"nb_tool_type": "k8s",
				},
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, nubiCmd, []string{"nubi", "tools"})
	require.NoError(t, err)

	assert.Contains(t, output, "kubectl_get")
	assert.Contains(t, output, "Get Kubernetes resources")
}

func TestNubiCmd_Functions(t *testing.T) {
	resetNubiFlags()
	viper.Set("username", "test-user")
	viper.Set("account-id", "test-account-id")
	t.Cleanup(resetNubiFlags)

	mockResponse := map[string]interface{}{
		"llm_functions": map[string]interface{}{
			"rows": []map[string]interface{}{
				{
					"name":        "analyze_logs",
					"description": "Analyze pod log streams",
					"status":      "active",
					"variables":   []string{"namespace", "pod_name"},
				},
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, nubiCmd, []string{"nubi", "functions"})
	require.NoError(t, err)

	assert.Contains(t, output, "analyze_logs")
	assert.Contains(t, output, "Analyze pod log streams")
}

func TestNubiCmd_Playbooks(t *testing.T) {
	resetNubiFlags()
	viper.Set("username", "test-user")
	viper.Set("account-id", "test-account-id")
	t.Cleanup(resetNubiFlags)

	mockResponse := map[string]interface{}{
		"agents_list_playbooks": map[string]interface{}{
			"rows": []map[string]interface{}{
				{
					"id":         "pb-101",
					"alert_name": "HighMemoryUsage",
					"source":     "Prometheus",
					"processor":  "auto_triage",
					"updated_at": "2026-08-20T12:00:00Z",
				},
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, nubiCmd, []string{"nubi", "playbooks"})
	require.NoError(t, err)

	assert.Contains(t, output, "pb-101")
	assert.Contains(t, output, "HighMemoryUsage")
}

func TestNubiCmd_Stats(t *testing.T) {
	resetNubiFlags()
	viper.Set("username", "test-user")
	viper.Set("account-id", "test-account-id")
	t.Cleanup(resetNubiFlags)

	mockResponse := map[string]interface{}{
		"ai_get_conversation_usage_metrics": map[string]interface{}{
			"data": map[string]interface{}{
				"conversation": map[string]interface{}{
					"total_cost":         0.005,
					"total_input_tokens": 1200,
					"total_output_tokens": 350,
				},
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, nubiCmd, []string{"nubi", "stats", "conv-123"})
	require.NoError(t, err)

	assert.Contains(t, output, "$0.005000")
	assert.Contains(t, output, "1200")
	assert.Contains(t, output, "350")
}

func TestNubiCmd_Stats_JSON(t *testing.T) {
	resetNubiFlags()
	viper.Set("username", "test-user")
	viper.Set("account-id", "test-account-id")
	t.Cleanup(resetNubiFlags)

	mockResponse := map[string]interface{}{
		"ai_get_conversation_usage_metrics": map[string]interface{}{
			"data": map[string]interface{}{
				"conversation": map[string]interface{}{
					"total_cost_usd":                  0.0275597,
					"total_input_tokens":             57270,
					"total_output_tokens":            996,
					"total_cached_input_tokens":       39857,
					"total_cache_hit_rate_percentage": 69.59,
					"model_usage": []map[string]interface{}{
						{
							"model_name": "gemini-3.5-flash-lite",
							"requests":   8,
						},
					},
				},
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, nubiCmd, []string{"nubi", "stats", "conv-123", "--output", "json"})
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err)

	assert.Equal(t, 0.0275597, result["total_cost_usd"])
	assert.Equal(t, float64(57270), result["total_input_tokens"])
	assert.NotEmpty(t, result["model_usage"])
}

func TestNubiCmd_Get_WithSessionIDFlag(t *testing.T) {
	resetNubiFlags()
	viper.Set("username", "test-user")
	viper.Set("account-id", "test-account-id")
	t.Cleanup(resetNubiFlags)

	mockResponse := map[string]interface{}{
		"ai_get_conversation_v3": map[string]interface{}{
			"conversation": map[string]interface{}{"id": "conv-real-id", "status": "COMPLETED"},
			"messages": []map[string]interface{}{
				{"id": "msg-1", "message_type": "generation", "response": "get-output", "status": "COMPLETED"},
			},
		},
	}

	t.Run("session-id flag with no positional argument", func(t *testing.T) {
		output, err := testutil.RunWithSimpleGraphQL(mockResponse, nubiCmd, []string{"nubi", "get", "--session-id", "sess-999", "-o", "json"})
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal([]byte(output), &result)
		require.NoError(t, err)

		assert.Equal(t, "conv-real-id", result["conversation_id"])
		assert.Equal(t, "sess-999", result["session_id"])
		assert.NotNil(t, result["details"])
	})

	t.Run("session-id flag with positional argument fallback", func(t *testing.T) {
		output, err := testutil.RunWithSimpleGraphQL(mockResponse, nubiCmd, []string{"nubi", "get", "conv-123", "--session-id", "sess-999", "-o", "json"})
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal([]byte(output), &result)
		require.NoError(t, err)

		assert.Equal(t, "conv-real-id", result["conversation_id"])
		assert.Equal(t, "sess-999", result["session_id"])
		assert.NotNil(t, result["details"])
	})

	t.Run("session-id flag in text mode resolves conversation and renders", func(t *testing.T) {
		output, err := testutil.RunWithSimpleGraphQL(mockResponse, nubiCmd, []string{"nubi", "get", "--session-id", "sess-999"})
		require.NoError(t, err)
		assert.Contains(t, output, "get-output")
	})

	t.Run("session-id flag in text mode returns error when conversation not found", func(t *testing.T) {
		emptyResponse := map[string]interface{}{
			"ai_get_conversation_v3": map[string]interface{}{
				"conversation": map[string]interface{}{"id": "", "status": ""},
			},
		}
		_, err := testutil.RunWithSimpleGraphQL(emptyResponse, nubiCmd, []string{"nubi", "get", "--session-id", "sess-unknown"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "conversation not found")
	})
}

func TestNubiCmd_Get_WithAccountId(t *testing.T) {
	resetNubiFlags()
	viper.Set("username", "test-user")
	viper.Set("account-id", "default-account")
	t.Cleanup(resetNubiFlags)

	var receivedAccountID string
	var receivedConvID string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/auth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{"token": "fake-token", "expiry": 3600})
		case "/api/graphql":
			var payload struct {
				Query     string                 `json:"query"`
				Variables map[string]interface{} `json:"variables"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err == nil {
				if acc, ok := payload.Variables["accountId"].(string); ok {
					receivedAccountID = acc
				}
				if conv, ok := payload.Variables["conversationId"].(string); ok {
					receivedConvID = conv
				}
			}
			resp := map[string]interface{}{
				"data": map[string]interface{}{
					"ai_get_conversation_v3": map[string]interface{}{
						"conversation": map[string]interface{}{
							"id":     "conv-explicit-1",
							"status": "COMPLETED",
						},
						"messages": []map[string]interface{}{
							{
								"id":           "msg-1",
								"status":       "COMPLETED",
								"response":     "Scoped to explicit account",
								"message_type": "generation",
							},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		default:
			http.NotFound(w, r)
		}
	})

	defaults := map[string]any{
		"api-key":    "dummy",
		"username":   "dummy-user",
		"account-id": "default-account",
	}

	output, err := testutil.RunWithMockServer(handler, defaults, nubiCmd, []string{
		"nubi", "get", "conv-explicit-1", "--account-id", "explicit-account-id",
	})
	require.NoError(t, err)
	assert.Equal(t, "explicit-account-id", receivedAccountID)
	assert.Equal(t, "conv-explicit-1", receivedConvID)
	assert.Contains(t, output, "Scoped")
	assert.Contains(t, output, "explicit")
}

func TestNubiCmd_Get_SessionId_WithAccountId(t *testing.T) {
	resetNubiFlags()
	viper.Set("username", "test-user")
	viper.Set("account-id", "default-account")
	t.Cleanup(resetNubiFlags)

	var receivedAccountID string
	var receivedSessionID string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/auth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{"token": "fake-token", "expiry": 3600})
		case "/api/graphql":
			var payload struct {
				Query     string                 `json:"query"`
				Variables map[string]interface{} `json:"variables"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err == nil {
				if acc, ok := payload.Variables["accountId"].(string); ok {
					receivedAccountID = acc
				}
				if sess, ok := payload.Variables["sessionId"].(string); ok {
					receivedSessionID = sess
				}
			}
			resp := map[string]interface{}{
				"data": map[string]interface{}{
					"ai_get_conversation_v3": map[string]interface{}{
						"conversation": map[string]interface{}{
							"id":     "conv-session-1",
							"status": "COMPLETED",
						},
						"messages": []map[string]interface{}{
							{
								"id":           "msg-1",
								"status":       "COMPLETED",
								"response":     "Found by session ID",
								"message_type": "generation",
							},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		default:
			http.NotFound(w, r)
		}
	})

	defaults := map[string]any{
		"api-key":    "dummy",
		"username":   "dummy-user",
		"account-id": "default-account",
	}

	output, err := testutil.RunWithMockServer(handler, defaults, nubiCmd, []string{
		"nubi", "get", "--session-id", "sess-explicit-1", "--account-id", "explicit-account-id",
	})
	require.NoError(t, err)
	assert.Equal(t, "explicit-account-id", receivedAccountID)
	assert.Equal(t, "sess-explicit-1", receivedSessionID)
	assert.Contains(t, output, "Found")
	assert.Contains(t, output, "session")
}

func TestNubiCmd_Get_NotFound_JSON(t *testing.T) {
	resetNubiFlags()
	viper.Set("username", "test-user")
	t.Cleanup(resetNubiFlags)

	emptyResponse := map[string]interface{}{
		"ai_get_conversation_v3": map[string]interface{}{
			"conversation": map[string]interface{}{"id": "", "status": ""},
		},
	}
	_, err := testutil.RunWithSimpleGraphQL(emptyResponse, nubiCmd, []string{"nubi", "get", "conv-missing", "-o", "json"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "conversation not found for account dummy-account")
}

func TestNubiCmd_SyncQuery_SessionIdDiffersFromConversationId(t *testing.T) {
	resetNubiFlags()
	viper.Set("username", "test-user")
	t.Cleanup(resetNubiFlags)

	var recordedVars []map[string]interface{}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/auth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{"token": "fake-token", "expiry": 3600})
		case "/api/graphql":
			var payload struct {
				Query     string                 `json:"query"`
				Variables map[string]interface{} `json:"variables"`
			}
			_ = json.NewDecoder(r.Body).Decode(&payload)
			recordedVars = append(recordedVars, payload.Variables)

			resp := map[string]interface{}{
				"data": map[string]interface{}{
					"ai_execute_investigation": map[string]interface{}{
						"data": map[string]interface{}{
							"response": "started",
						},
					},
					"ai_get_conversation_v3": map[string]interface{}{
						"conversation": map[string]interface{}{
							"id":     "conv-different-uuid",
							"status": "COMPLETED",
						},
						"messages": []map[string]interface{}{
							{
								"id":           "msg-1",
								"status":       "COMPLETED",
								"response":     "Polled successfully with distinct session and conversation IDs",
								"message_type": "generation",
							},
						},
					},
					"ai_get_conversation_usage_metrics": map[string]interface{}{
						"data": map[string]interface{}{
							"conversation": map[string]interface{}{
								"total_cost": 0.001,
							},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		default:
			http.NotFound(w, r)
		}
	})

	defaults := map[string]any{
		"api-key":    "dummy",
		"username":   "dummy-user",
		"account-id": "dummy-account",
	}

	output, err := testutil.RunWithMockServer(handler, defaults, nubiCmd, []string{"nubi", "query", "check distinct ids"})
	require.NoError(t, err)
	assert.Contains(t, output, "Polled")
	assert.Contains(t, output, "successfully")

	require.True(t, len(recordedVars) >= 2)
	pollVars := recordedVars[1]
	assert.NotEmpty(t, pollVars["sessionId"])
	assert.Nil(t, pollVars["conversationId"])
}

