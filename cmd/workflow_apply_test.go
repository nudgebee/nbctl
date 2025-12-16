package cmd

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"nudgebee.com/nbctl/pkg/testutil"
)

func TestWorkflowApplyCmd(t *testing.T) {
	// Create a temporary YAML file
	tmpFile, err := os.CreateTemp("", "workflow-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	yamlContent := `
name: New Workflow
definition:
  version: v1
  tasks:
    - id: task1
      type: core.print
`
	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	tmpFile.Close()

	mockResponse := map[string]interface{}{
		"workflow_create": map[string]interface{}{
			"id": "wf-new-1",
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, workflowCmd, []string{"workflow", "apply", tmpFile.Name()})
	require.NoError(t, err)

	assert.Contains(t, output, "Workflow created with ID: wf-new-1")
}

func TestWorkflowApplyCmd_JSON(t *testing.T) {
	// Create a temporary YAML file
	tmpFile, err := os.CreateTemp("", "workflow-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	yamlContent := `
name: New Workflow
definition:
  version: v1
`
	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	tmpFile.Close()

	mockResponse := map[string]interface{}{
		"workflow_create": map[string]interface{}{
			"id": "wf-new-1",
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, workflowCmd, []string{"workflow", "apply", tmpFile.Name(), "--format", "json"})
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err)
	assert.Equal(t, "wf-new-1", result["id"])
}
