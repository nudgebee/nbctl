package cmd

import (
	"encoding/json"
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditLogsListCmd(t *testing.T) {
	mockResponse := map[string]any{
		"audits_v2": map[string]any{
			"rows": []map[string]any{
				{
					"user_id":        "user-1",
					"account_id":     "acc-1",
					"event_time":     "2026-06-01T00:00:00Z",
					"event_category": "AUTO_PILOT",
					"event_type":     "AUTOPILOT_UPDATE",
					"event_status":   "SUCCESS",
					"event_target":   "target-1",
					"event_action":   "UPDATE",
					"transaction_id": "txn-1",
				},
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, auditLogsCmd, []string{"audit-logs", "list"})
	require.NoError(t, err)

	assert.Contains(t, output, "user-1")
	assert.Contains(t, output, "AUTO_PILOT")
	assert.Contains(t, output, "AUTOPILOT_UPDATE")
	assert.Contains(t, output, "SUCCESS")
}

func TestAuditLogsListCmd_JSON(t *testing.T) {
	mockResponse := map[string]any{
		"audits_v2": map[string]any{
			"rows": []map[string]any{
				{
					"user_id":        "user-1",
					"event_time":     "2026-06-01T00:00:00Z",
					"event_category": "AUTO_PILOT",
					"event_type":     "AUTOPILOT_UPDATE",
					"event_action":   "UPDATE",
					"event_status":   "SUCCESS",
				},
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, auditLogsCmd, []string{"audit-logs", "list", "--format", "json"})
	require.NoError(t, err)

	var result []any
	require.NoError(t, json.Unmarshal([]byte(output), &result))
	require.Len(t, result, 1)
	first := result[0].(map[string]any)
	assert.Equal(t, "user-1", first["user_id"])
	assert.Equal(t, "AUTOPILOT_UPDATE", first["event_type"])
}
