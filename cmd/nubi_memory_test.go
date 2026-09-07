package cmd

import (
	"encoding/json"
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNubiMemoryCmd_List(t *testing.T) {
	resetNubiFlags()
	viper.Set("account-id", "test-account-id")
	t.Cleanup(resetNubiFlags)

	mockResponse := map[string]any{
		"ai_list_memory": map[string]any{
			"data": []map[string]any{
				{
					"id":          "mem-201",
					"memory_type": "investigation_result",
					"content":     "RabbitMQ queue backlog pattern",
					"created_at":  "2026-09-07T10:00:00Z",
				},
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, nubiCmd, []string{"nubi", "memory", "list"})
	require.NoError(t, err)

	assert.Contains(t, output, "mem-201")
	assert.Contains(t, output, "investigation_result")
	assert.Contains(t, output, "RabbitMQ queue backlog pattern")
}

func TestNubiMemoryCmd_List_JSON(t *testing.T) {
	resetNubiFlags()
	viper.Set("account-id", "test-account-id")
	t.Cleanup(resetNubiFlags)

	mockResponse := map[string]any{
		"ai_list_memory": map[string]any{
			"data": []map[string]any{
				{
					"id":          "mem-202",
					"memory_type": "architecture_decision",
					"content":     "Use PostgreSQL for transaction log persistence",
					"created_at":  "2026-09-07T11:00:00Z",
				},
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, nubiCmd, []string{"nubi", "memory", "list", "-o", "json", "--type", "architecture_decision"})
	require.NoError(t, err)

	var result []map[string]any
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err)

	require.Len(t, result, 1)
	assert.Equal(t, "mem-202", result[0]["id"])
	assert.Equal(t, "architecture_decision", result[0]["memory_type"])
}
