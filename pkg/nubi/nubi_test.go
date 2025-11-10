package nubi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/machinebox/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestNubiClient(handler http.HandlerFunc) (*NubiClient, func()) {
	srv := httptest.NewServer(handler)
	client := graphql.NewClient(srv.URL)
	nubiClient := New(client, "test-account", "test-user", "test-session", srv.URL)
	return nubiClient, srv.Close
}

func TestNubiClient_GetConversationMessages(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"llm_conversation_messages": []map[string]any{
					{"role": "user", "message": "hello"},
					{"role": "assistant", "response": "hi there"},
				},
			},
		}
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}

	client, teardown := newTestNubiClient(handler)
	defer teardown()

	messages, err := client.GetConversationMessages("test-conversation")
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
				"llm_conversations": []map[string]any{
					{"id": "1", "title": "conv 1", "updated_at": "2023-01-01T12:00:00.000000"},
					{"id": "2", "title": "conv 2", "updated_at": "2023-01-02T12:00:00.000000"},
				},
			},
		}
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}

	client, teardown := newTestNubiClient(handler)
	defer teardown()

	history, err := client.ShowHistory(2)
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

	client, teardown := newTestNubiClient(handler)
	defer teardown()

	err := client.TriggerInvestigation(context.Background(), "test query")
	assert.NoError(t, err)
}
