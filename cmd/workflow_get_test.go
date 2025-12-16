package cmd

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"nudgebee.com/nbctl/pkg/testutil"
)

func TestWorkflowGetCmd(t *testing.T) {
	mockResponse := map[string]interface{}{
		"workflow_get": map[string]interface{}{
			"id":                    "wf-1",
			"name":                  "Test Workflow",
			"status":                "ACTIVE",
			"last_execution_status": "COMPLETED",
			"created_at":            "2023-01-01T00:00:00Z",
			"definition": map[string]interface{}{
				"version": "v1",
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, workflowCmd, []string{"workflow", "get", "wf-1"})
	require.NoError(t, err)

	assert.Contains(t, output, "wf-1")
	assert.Contains(t, output, "Test Workflow")
	assert.Contains(t, output, "Definition:")
}

func TestWorkflowGetCmd_JSON(t *testing.T) {
	mockResponse := map[string]interface{}{
		"workflow_get": map[string]interface{}{
			"id":                    "wf-1",
			"name":                  "Test Workflow",
			"status":                "ACTIVE",
			"last_execution_status": "COMPLETED",
			"created_at":            "2023-01-01T00:00:00Z",
			"definition": map[string]interface{}{
				"version": "v1",
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, workflowCmd, []string{"workflow", "get", "wf-1", "--format", "json"})
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err)
	assert.Equal(t, "wf-1", result["id"])
}
