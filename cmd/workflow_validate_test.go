package cmd

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowValidateCmd_Valid(t *testing.T) {
	// Create a temporary YAML file
	tmpFile, err := os.CreateTemp("", "workflow-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

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
	require.NoError(t, tmpFile.Close())

	mockResponse := map[string]interface{}{
		"workflow_check": map[string]interface{}{
			"message": "Workflow is valid",
		},
	}

	// We must use RunWithMockServer to control headers and ensure NBCTL_TESTING is set
	// RunWithSimpleGraphQL also sets NBCTL_TESTING, so it should be fine for success case
	output, err := testutil.RunWithSimpleGraphQL(mockResponse, workflowCmd, []string{"workflow", "validate", tmpFile.Name()})
	require.NoError(t, err)

	assert.Contains(t, output, "Workflow is valid")
}

func TestWorkflowValidateCmd_Invalid(t *testing.T) {
	// Create a temporary YAML file
	tmpFile, err := os.CreateTemp("", "workflow-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	yamlContent := `
name: New Workflow
definition:
  version: v1
`
	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	// Mock response with top-level GraphQL errors
	mockResponse := map[string]interface{}{
		"errors": []map[string]interface{}{
			{
				"message": "missing tasks",
			},
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth/token" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"token": "fake-token", "expiry": 3600})
			return
		}
		if r.URL.Path == "/api/graphql" {
			w.Header().Set("Content-Type", "application/json")
			// Return top-level errors
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(mockResponse)
			return
		}
		http.NotFound(w, r)
	})

	err = os.Setenv("NBCTL_TESTING", "true")
	require.NoError(t, err)
	defer func() { _ = os.Unsetenv("NBCTL_TESTING") }()

	_, err = testutil.RunWithMockServer(handler, map[string]any{
		"api-key":    "dummy",
		"username":   "dummy-user",
		"account-id": "dummy-account",
	}, workflowCmd, []string{"workflow", "validate", tmpFile.Name()})

	require.Error(t, err)
	// assert.Contains(t, err.Error(), "workflow validation failed")
	// Updated error message from generic client
	// assert.Contains(t, err.Error(), "graphql errors") // Removed this expectation as it is not present in GraphQLErrors
	assert.Contains(t, err.Error(), "missing tasks")
}

func TestWorkflowValidateCmd_JSON(t *testing.T) {
	// Create a temporary YAML file
	tmpFile, err := os.CreateTemp("", "workflow-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	yamlContent := `
name: New Workflow
definition:
  version: v1
`
	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	mockResponse := map[string]interface{}{
		"workflow_check": map[string]interface{}{
			"message": "Success",
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, workflowCmd, []string{"workflow", "validate", tmpFile.Name(), "--format", "json"})
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err)
	assert.Equal(t, "Success", result["message"])
}

func TestWorkflowValidateCmd_EmptyFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "workflow-empty-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	require.NoError(t, tmpFile.Close())

	mockResponse := map[string]interface{}{
		"workflow_check": map[string]interface{}{"message": "should not be called"},
	}

	_, err = testutil.RunWithSimpleGraphQL(mockResponse, workflowCmd, []string{"workflow", "validate", tmpFile.Name()})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty or not a YAML mapping")
}

func TestWorkflowValidateCmd_HasuraDrillDownError(t *testing.T) {
	// Create a temporary YAML file
	tmpFile, err := os.CreateTemp("", "workflow-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	yamlContent := `
name: New Workflow
definition:
  version: v1
`
	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	// Mock response matching the user provided JSON
	mockResponse := map[string]interface{}{
		"errors": []map[string]interface{}{
			{
				"message": "parsing Hasura.GraphQL.Execute.Action.Types.ActionWebhookErrorResponse(ActionWebhookErrorResponse) failed, key \"message\" not found",
				"extensions": map[string]interface{}{
					"path": "$",
					"code": "parse-failed",
					"internal": map[string]interface{}{
						"response": map[string]interface{}{
							"body": map[string]interface{}{
								"errors": []map[string]interface{}{
									{
										"message": "invalid workflow definition: missing triggers",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// We need to use RunWithMockServer to simulate top-level errors (not wrapped in data)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth/token" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"token": "fake-token", "expiry": 3600})
			return
		}
		if r.URL.Path == "/api/graphql" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(mockResponse)
			return
		}
		http.NotFound(w, r)
	})

	err = os.Setenv("NBCTL_TESTING", "true")
	require.NoError(t, err)
	defer func() { _ = os.Unsetenv("NBCTL_TESTING") }()

	_, err = testutil.RunWithMockServer(handler, map[string]any{
		"api-key":    "dummy",
		"username":   "dummy-user",
		"account-id": "dummy-account",
	}, workflowCmd, []string{"workflow", "validate", tmpFile.Name()})

	require.Error(t, err)
	// assert.Contains(t, err.Error(), "Workflow is invalid (backend validation failed):")
	// Updated error message
	assert.Contains(t, err.Error(), "Backend validation failed:")
	assert.Contains(t, err.Error(), "- invalid workflow definition: missing triggers")
}
