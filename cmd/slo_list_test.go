package cmd

import (
	"encoding/json"
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSLOListCmd(t *testing.T) {
	mockResponse := map[string]any{
		"slo_config_v2": map[string]any{
			"rows": []map[string]any{
				{
					"id":                 "slo-1",
					"name":               "latency",
					"workload_name":      "api-server",
					"workload_namespace": "prod",
					"cloud_account_id":   "acc-1",
					"goal":               "0.99",
					"threshold":          "5",
					"window":             "3600",
					"enabled":            true,
					"created_at":         "2026-01-01T00:00:00Z",
					"updated_at":         "2026-01-02T00:00:00Z",
				},
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, sloCmd, []string{"slo", "list"})
	require.NoError(t, err)

	assert.Contains(t, output, "slo-1")
	assert.Contains(t, output, "latency")
	assert.Contains(t, output, "api-server")
	assert.Contains(t, output, "0.99")
}

func TestSLOListCmd_JSON(t *testing.T) {
	mockResponse := map[string]any{
		"slo_config_v2": map[string]any{
			"rows": []map[string]any{
				{
					"id":      "slo-1",
					"name":    "latency",
					"goal":    "0.99",
					"enabled": true,
				},
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, sloCmd, []string{"slo", "list", "--format", "json"})
	require.NoError(t, err)

	var result []any
	require.NoError(t, json.Unmarshal([]byte(output), &result))
	require.Len(t, result, 1)
	first := result[0].(map[string]any)
	assert.Equal(t, "slo-1", first["id"])
}
