package cmd

import (
	"encoding/json"
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowGetExecutionCmd(t *testing.T) {
	mockResponse := map[string]any{
		"workflow_get_execution": map[string]any{
			"id":             "exec-1",
			"workflow_id":    "wf-1",
			"triggered_by":   "user@example.com",
			"status":         "COMPLETED",
			"start_time":     "2026-06-01T00:00:00Z",
			"close_time":     "2026-06-01T00:01:00Z",
			"inputs":         map[string]any{"key": "value"},
			"version_number": 3,
			"version_name":   "v3",
			"tasks": []map[string]any{
				{
					"id":         "task-1",
					"type":       "shell",
					"status":     "SUCCEEDED",
					"start_time": "2026-06-01T00:00:10Z",
					"end_time":   "2026-06-01T00:00:50Z",
					"attempt":    1,
				},
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, workflowCmd, []string{"workflow", "get-execution", "wf-1", "exec-1"})
	require.NoError(t, err)

	assert.Contains(t, output, "exec-1")
	assert.Contains(t, output, "wf-1")
	assert.Contains(t, output, "COMPLETED")
	assert.Contains(t, output, "user@example.com")
	assert.Contains(t, output, "Tasks (1)")
	assert.Contains(t, output, "task-1")
	assert.Contains(t, output, "SUCCEEDED")
}

func TestWorkflowGetExecutionCmd_JSON(t *testing.T) {
	mockResponse := map[string]any{
		"workflow_get_execution": map[string]any{
			"id":          "exec-1",
			"workflow_id": "wf-1",
			"status":      "COMPLETED",
			"tasks":       []map[string]any{},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, workflowCmd, []string{"workflow", "get-execution", "wf-1", "exec-1", "--format", "json"})
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &result))
	assert.Equal(t, "exec-1", result["id"])
	assert.Equal(t, "wf-1", result["workflow_id"])
	assert.Equal(t, "COMPLETED", result["status"])
}
