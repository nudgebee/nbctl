package cmd

import (
	"encoding/json"
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowListExecutionsCmd(t *testing.T) {
	mockResponse := map[string]any{
		"workflow_list_executions": map[string]any{
			"next_page_token": "next-token-1",
			"executions": []map[string]any{
				{
					"id":                 "exec-1",
					"workflow_id":        "wf-1",
					"parent_workflow_id": nil,
					"status":             "COMPLETED",
					"trigger_type":       "MANUAL",
					"triggered_by":       "user@example.com",
					"start_time":         "2026-06-01T00:00:00Z",
					"close_time":         "2026-06-01T00:01:00Z",
				},
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, workflowCmd, []string{"workflow", "list-executions", "wf-1"})
	require.NoError(t, err)

	assert.Contains(t, output, "exec-1")
	assert.Contains(t, output, "COMPLETED")
	assert.Contains(t, output, "MANUAL")
	assert.Contains(t, output, "user@example.com")
	assert.Contains(t, output, "next-token-1")
}

func TestWorkflowListExecutionsCmd_PaginationToken(t *testing.T) {
	mockResponse := map[string]any{
		"workflow_list_executions": map[string]any{
			"next_page_token": "tok-page-2",
			"executions": []map[string]any{
				{
					"id":           "exec-1",
					"workflow_id":  "wf-1",
					"status":       "COMPLETED",
					"trigger_type": "MANUAL",
					"triggered_by": "u@example.com",
					"start_time":   "2026-06-01T00:00:00Z",
					"close_time":   "2026-06-01T00:01:00Z",
				},
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, workflowCmd, []string{"workflow", "list-executions", "wf-1"})
	require.NoError(t, err)

	assert.Contains(t, output, "exec-1")
	assert.Contains(t, output, "Next page token: tok-page-2")
}

func TestWorkflowListExecutionsCmd_JSON(t *testing.T) {
	mockResponse := map[string]any{
		"workflow_list_executions": map[string]any{
			"next_page_token": nil,
			"executions": []map[string]any{
				{
					"id":           "exec-1",
					"workflow_id":  "wf-1",
					"status":       "COMPLETED",
					"trigger_type": "MANUAL",
					"triggered_by": "user@example.com",
					"start_time":   "2026-06-01T00:00:00Z",
					"close_time":   "2026-06-01T00:01:00Z",
				},
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, workflowCmd, []string{"workflow", "list-executions", "wf-1", "--format", "json"})
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &result))

	execs, ok := result["executions"].([]any)
	require.True(t, ok, "expected executions array in JSON output, got %v", result)
	require.Len(t, execs, 1)
	first := execs[0].(map[string]any)
	assert.Equal(t, "exec-1", first["id"])
	assert.Equal(t, "COMPLETED", first["status"])
}
