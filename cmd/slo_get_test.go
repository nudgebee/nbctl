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

func TestSLOGetCmd_WithReport(t *testing.T) {
	t.Cleanup(func() { _ = sloGetCmd.Flags().Set("report", "false") })

	mockResponse := map[string]any{
		"slo_config_v2": map[string]any{
			"rows": []map[string]any{
				{
					"id":                 "slo-1",
					"name":               "latency",
					"workload_name":      "api-server",
					"workload_namespace": "prod",
					"goal":               "0.99",
					"enabled":            true,
				},
			},
		},
		"slo_report_v2": map[string]any{
			"rows": []map[string]any{
				{
					"id":                     "rep-1",
					"config_id":              "slo-1",
					"status":                 "HEALTHY",
					"error_budget_burn_rate": 0.42,
					"events_count":           1000,
					"good_events_count":      995,
					"bad_events_count":       5,
					"updated_at":             "2026-06-01T00:00:00Z",
				},
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, sloCmd, []string{"slo", "get", "slo-1", "--report"})
	require.NoError(t, err)

	assert.Contains(t, output, "ID:        slo-1")
	assert.Contains(t, output, "Latest Report")
	assert.Contains(t, output, "HEALTHY")
	assert.Contains(t, output, "995 / 5 / 1000")
}

func TestSLOGetCmd_WithReportButNoReportRows(t *testing.T) {
	t.Cleanup(func() { _ = sloGetCmd.Flags().Set("report", "false") })

	mockResponse := map[string]any{
		"slo_config_v2": map[string]any{
			"rows": []map[string]any{
				{"id": "slo-1", "name": "latency", "workload_name": "api-server"},
			},
		},
		"slo_report_v2": map[string]any{
			"rows": []map[string]any{},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, sloCmd, []string{"slo", "get", "slo-1", "--report"})
	require.NoError(t, err)

	assert.Contains(t, output, "ID:        slo-1")
	assert.NotContains(t, output, "Latest Report")
	assert.NotContains(t, output, "warning")
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
