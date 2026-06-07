package cmd

import (
	"encoding/json"
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTicketsListConfigurationsCmd(t *testing.T) {
	mockResponse := map[string]any{
		"integrations": map[string]any{
			"rows": []map[string]any{
				{
					"id":                        "int-1",
					"name":                      "Prod Jira",
					"type":                      "jira",
					"status":                    "enabled",
					"created_by_display_name":   "Alice",
					"updated_at":                "2026-01-01T00:00:00Z",
					"integration_config_values": "{}",
				},
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, ticketsCmd, []string{"tickets", "list-configurations"})
	require.NoError(t, err)

	assert.Contains(t, output, "int-1")
	assert.Contains(t, output, "Prod Jira")
	assert.Contains(t, output, "jira")
	assert.Contains(t, output, "enabled")
	assert.Contains(t, output, "Alice")
}

func TestTicketsListConfigurationsCmd_JSON(t *testing.T) {
	mockResponse := map[string]any{
		"integrations": map[string]any{
			"rows": []map[string]any{
				{
					"id":   "int-1",
					"name": "Prod Jira",
					"type": "jira",
				},
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, ticketsCmd, []string{"tickets", "list-configurations", "--format", "json"})
	require.NoError(t, err)

	var result []any
	require.NoError(t, json.Unmarshal([]byte(output), &result))
	require.Len(t, result, 1)
	first := result[0].(map[string]any)
	assert.Equal(t, "int-1", first["id"])
}
