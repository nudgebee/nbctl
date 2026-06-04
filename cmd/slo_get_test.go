package cmd

import (
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSLOGetCmd(t *testing.T) {
	mockResponse := map[string]any{
		"slo_config_v2": map[string]any{
			"rows": []map[string]any{
				{
					"id":                 "slo-1",
					"name":               "latency",
					"workload_name":      "api-server",
					"workload_namespace": "prod",
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

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, sloCmd, []string{"slo", "get", "slo-1"})
	require.NoError(t, err)

	assert.Contains(t, output, "ID:        slo-1")
	assert.Contains(t, output, "Name:      latency")
	assert.Contains(t, output, "Workload:  api-server")
	assert.Contains(t, output, "Goal:      0.99")
}

func TestSLOGetCmd_NotFound(t *testing.T) {
	mockResponse := map[string]any{
		"slo_config_v2": map[string]any{
			"rows": []map[string]any{},
		},
	}

	_, err := testutil.RunWithSimpleGraphQL(mockResponse, sloCmd, []string{"slo", "get", "missing-id"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
