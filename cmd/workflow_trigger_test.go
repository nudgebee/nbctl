package cmd

import (
	"encoding/json"
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowTriggerCmd(t *testing.T) {
	mockResponse := map[string]interface{}{
		"workflow_execute": map[string]interface{}{
			"id":           "wf-1",
			"execution_id": "exec-1",
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, workflowCmd, []string{"workflow", "trigger", "wf-1"})
	require.NoError(t, err)

	assert.Contains(t, output, "Workflow triggered")
	assert.Contains(t, output, "exec-1")
}

func TestWorkflowTriggerCmd_JSON(t *testing.T) {
	mockResponse := map[string]interface{}{
		"workflow_execute": map[string]interface{}{
			"id":           "wf-1",
			"execution_id": "exec-1",
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, workflowCmd, []string{"workflow", "trigger", "wf-1", "--format", "json"})
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err)
	assert.Equal(t, "exec-1", result["execution_id"])
}
