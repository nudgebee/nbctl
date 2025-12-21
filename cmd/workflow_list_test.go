package cmd

import (
	"encoding/json"
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowListCmd(t *testing.T) {
	mockResponse := map[string]interface{}{
		"workflow_list": map[string]interface{}{
			"workflows": []map[string]interface{}{
				{
					"id":                    "wf-1",
					"name":                  "Test Workflow",
					"status":                "ACTIVE",
					"last_execution_status": "COMPLETED",
					"created_at":            "2023-01-01T00:00:00Z",
				},
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, workflowCmd, []string{"workflow", "list"})
	require.NoError(t, err)

	assert.Contains(t, output, "wf-1")
	assert.Contains(t, output, "Test Workflow")
	assert.Contains(t, output, "ACTIVE")
}

func TestWorkflowListCmd_JSON(t *testing.T) {
	mockResponse := map[string]interface{}{
		"workflow_list": map[string]interface{}{
			"workflows": []map[string]interface{}{
				{
					"id":                    "wf-1",
					"name":                  "Test Workflow",
					"status":                "ACTIVE",
					"last_execution_status": "COMPLETED",
					"created_at":            "2023-01-01T00:00:00Z",
				},
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, workflowCmd, []string{"workflow", "list", "--format", "json"})
	require.NoError(t, err)

	var result []interface{}
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	wf := result[0].(map[string]interface{})
	assert.Equal(t, "wf-1", wf["id"])
}
