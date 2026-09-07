package cmd

import (
	"encoding/json"
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNubiKbCmd_List(t *testing.T) {
	resetNubiFlags()
	viper.Set("account-id", "test-account-id")
	t.Cleanup(resetNubiFlags)

	mockResponse := map[string]any{
		"ai_list_kb": map[string]any{
			"data": []map[string]any{
				{
					"id":              "kb-101",
					"name":            "Payment Runbook",
					"kb_type":         "manual",
					"status":          "active",
					"document_count":  3,
					"last_loaded_at":  "2026-09-07T12:00:00Z",
					"data_size_bytes": 1024,
				},
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, nubiCmd, []string{"nubi", "kb", "list"})
	require.NoError(t, err)

	assert.Contains(t, output, "kb-101")
	assert.Contains(t, output, "Payment Runbook")
	assert.Contains(t, output, "manual")
	assert.Contains(t, output, "active")
}

func TestNubiKbCmd_List_PositionalAccountPrecedence(t *testing.T) {
	resetNubiFlags()
	viper.Set("account-id", "profile-account-id")
	t.Cleanup(resetNubiFlags)

	mockResponse := map[string]any{
		"ai_list_kb": map[string]any{
			"data": []map[string]any{
				{
					"id":   "kb-override",
					"name": "Overridden KB",
				},
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, nubiCmd, []string{"nubi", "kb", "list", "override-account-id"})
	require.NoError(t, err)
	assert.Contains(t, output, "kb-override")
}

func TestNubiKbCmd_List_JSON(t *testing.T) {
	resetNubiFlags()
	viper.Set("account-id", "test-account-id")
	t.Cleanup(resetNubiFlags)

	mockResponse := map[string]any{
		"ai_list_kb": map[string]any{
			"data": []map[string]any{
				{
					"id":             "kb-102",
					"name":           "K8s Troubleshooting",
					"kb_type":        "integration",
					"status":         "active",
					"document_count": 15,
				},
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, nubiCmd, []string{"nubi", "kb", "list", "-o", "json"})
	require.NoError(t, err)

	var result []map[string]any
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err)

	require.Len(t, result, 1)
	assert.Equal(t, "kb-102", result[0]["id"])
	assert.Equal(t, "K8s Troubleshooting", result[0]["name"])
}

func TestNubiKbCmd_Get(t *testing.T) {
	resetNubiFlags()
	viper.Set("account-id", "test-account-id")
	t.Cleanup(resetNubiFlags)

	mockResponse := map[string]any{
		"ai_get_kb": map[string]any{
			"data": map[string]any{
				"id":             "kb-101",
				"name":           "Payment Runbook",
				"description":    "SOP for payment gateway latency",
				"status":         "active",
				"kb_type":        "manual",
				"document_count": 3,
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, nubiCmd, []string{"nubi", "kb", "get", "kb-101"})
	require.NoError(t, err)

	assert.Contains(t, output, "kb-101")
	assert.Contains(t, output, "Payment Runbook")
	assert.Contains(t, output, "SOP for payment gateway latency")
}

func TestNubiKbCmd_Sync(t *testing.T) {
	resetNubiFlags()
	viper.Set("account-id", "test-account-id")
	t.Cleanup(resetNubiFlags)

	mockResponse := map[string]any{
		"ai_sync_kb": map[string]any{
			"data": "ok",
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, nubiCmd, []string{"nubi", "kb", "sync", "kb-101"})
	require.NoError(t, err)

	assert.Contains(t, output, "triggered")
	assert.Contains(t, output, "kb-101")
}

func TestNubiKbCmd_Enable_Disable(t *testing.T) {
	resetNubiFlags()
	viper.Set("account-id", "test-account-id")
	t.Cleanup(resetNubiFlags)

	mockResponse := map[string]any{
		"ai_update_kb_enabled": map[string]any{
			"data": "ok",
		},
	}

	t.Run("enable", func(t *testing.T) {
		output, err := testutil.RunWithSimpleGraphQL(mockResponse, nubiCmd, []string{"nubi", "kb", "enable", "kb-101"})
		require.NoError(t, err)
		assert.Contains(t, output, "enabled")
		assert.Contains(t, output, "kb-101")
	})

	t.Run("disable", func(t *testing.T) {
		output, err := testutil.RunWithSimpleGraphQL(mockResponse, nubiCmd, []string{"nubi", "kb", "disable", "kb-101"})
		require.NoError(t, err)
		assert.Contains(t, output, "disabled")
		assert.Contains(t, output, "kb-101")
	})
}
