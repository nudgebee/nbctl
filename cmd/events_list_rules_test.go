package cmd

import (
	"encoding/json"
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventsListRulesCmd(t *testing.T) {
	mockResponse := map[string]any{
		"event_rules_v2": map[string]any{
			"rows": []map[string]any{
				{
					"id":         "rule-1",
					"alert":      "PostgreSQLCacheHitRatio",
					"severity":   "warning",
					"category":   "alert",
					"source":     "prometheus_alertmanager_webhook",
					"group":      "postgres",
					"enabled":    true,
					"updated_at": "2026-06-01T00:00:00Z",
				},
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, eventsCmd, []string{"events", "list-rules"})
	require.NoError(t, err)

	assert.Contains(t, output, "rule-1")
	assert.Contains(t, output, "PostgreSQLCacheHitRatio")
	assert.Contains(t, output, "warning")
	assert.Contains(t, output, "prometheus_alertmanager_webhook")
}

func TestEventsListRulesCmd_JSON(t *testing.T) {
	mockResponse := map[string]any{
		"event_rules_v2": map[string]any{
			"rows": []map[string]any{
				{"id": "rule-1", "alert": "X", "enabled": true},
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, eventsCmd, []string{"events", "list-rules", "--format", "json"})
	require.NoError(t, err)

	var result []any
	require.NoError(t, json.Unmarshal([]byte(output), &result))
	require.Len(t, result, 1)
	first := result[0].(map[string]any)
	assert.Equal(t, "rule-1", first["id"])
}
