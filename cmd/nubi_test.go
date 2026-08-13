package cmd

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNubiCmd_AsyncQuery(t *testing.T) {
	viper.Set("username", "test-user")
	t.Cleanup(func() {
		_ = nubiCmd.Flags().Set("query", "")
		_ = nubiCmd.Flags().Set("async", "false")
		viper.Set("username", "")
	})

	mockResponse := map[string]interface{}{
		"ai_execute_investigation": map[string]interface{}{
			"data": map[string]interface{}{
				"response": "ok",
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, nubiCmd, []string{"nubi", "test-account-id", "-q", "hello", "--async"})
	require.NoError(t, err)

	assert.Contains(t, output, "Investigation triggered asynchronously.")
	assert.Contains(t, output, "Session ID:")
}

func TestNubiCmd_AsyncWithoutQuery(t *testing.T) {
	viper.Set("username", "test-user")
	t.Cleanup(func() {
		_ = nubiCmd.Flags().Set("query", "")
		_ = nubiCmd.Flags().Set("async", "false")
		viper.Set("username", "")
	})

	_, err := testutil.RunWithSimpleGraphQL(nil, nubiCmd, []string{"nubi", "test-account-id", "--async"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--async requires --query / -q")
}

func TestNubiCmd_SyncQuery(t *testing.T) {
	viper.Set("username", "test-user")
	t.Cleanup(func() {
		_ = nubiCmd.Flags().Set("query", "")
		_ = nubiCmd.Flags().Set("async", "false")
		viper.Set("username", "")
	})

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
	output, err := testutil.RunWithMockServer(handler, defaults, nubiCmd, []string{"nubi", "test-account-id", "-q", "system status"})
	require.NoError(t, err)

	assert.Contains(t, output, "System status")
	assert.Contains(t, output, "healthy")
	assert.Contains(t, output, "Cost: $0.001000")
	assert.Contains(t, output, "Response time:")
}
