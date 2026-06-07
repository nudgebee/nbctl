package cmd

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowApplyCmd_Create(t *testing.T) {
	_ = os.Setenv("NBCTL_TESTING", "true")
	defer func() { _ = os.Unsetenv("NBCTL_TESTING") }()

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

	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth/token" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"token": "fake-token", "expiry": 3600})
			return
		}
		if r.URL.Path == "/api/graphql" {
			var reqBody struct {
				Query     string         `json:"query"`
				Variables map[string]any `json:"variables"`
			}
			_ = json.NewDecoder(r.Body).Decode(&reqBody)

			w.Header().Set("Content-Type", "application/json")
			if strings.Contains(reqBody.Query, "ListWorkflows") {
				// Verify name variable is passed
				if reqBody.Variables["name"] != "New Workflow" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{
					"data": map[string]any{
						"workflow_list": map[string]any{
							"workflows": []any{},
						},
					},
				})
			} else if strings.Contains(reqBody.Query, "CreateWorkflow") {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"data": map[string]any{
						"workflow_create": map[string]any{
							"id": "wf-new-1",
						},
					},
				})
			} else {
				// Fallback or error
				w.WriteHeader(http.StatusBadRequest)
			}
			return
		}
		http.NotFound(w, r)
	}

	defaults := map[string]any{
		"api-key":    "dummy",
		"username":   "dummy-user",
		"account-id": "dummy-account",
	}

	output, err := testutil.RunWithMockServer(http.HandlerFunc(handler), defaults, workflowCmd, []string{"workflow", "apply", tmpFile.Name()})
	require.NoError(t, err)

	assert.Contains(t, output, "Workflow created with ID: wf-new-1")
}

func TestWorkflowApplyCmd_Update(t *testing.T) {
	_ = os.Setenv("NBCTL_TESTING", "true")
	defer func() { _ = os.Unsetenv("NBCTL_TESTING") }()

	// Create a temporary YAML file
	tmpFile, err := os.CreateTemp("", "workflow-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	yamlContent := `
name: Existing Workflow
definition:
  version: v1
`
	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth/token" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"token": "fake-token", "expiry": 3600})
			return
		}
		if r.URL.Path == "/api/graphql" {
			var reqBody struct {
				Query     string         `json:"query"`
				Variables map[string]any `json:"variables"`
			}
			_ = json.NewDecoder(r.Body).Decode(&reqBody)

			w.Header().Set("Content-Type", "application/json")
			if strings.Contains(reqBody.Query, "ListWorkflows") {
				if reqBody.Variables["name"] != "Existing Workflow" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{
					"data": map[string]any{
						"workflow_list": map[string]any{
							"workflows": []any{
								map[string]any{
									"id":   "wf-existing-1",
									"name": "Existing Workflow",
								},
							},
						},
					},
				})
			} else if strings.Contains(reqBody.Query, "UpdateWorkflow") {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"data": map[string]any{
						"workflow_update": map[string]any{
							"id": "wf-existing-1",
						},
					},
				})
			} else {
				// Fail if it tries to create
				w.WriteHeader(http.StatusBadRequest)
			}
			return
		}
		http.NotFound(w, r)
	}

	defaults := map[string]any{
		"api-key":    "dummy",
		"username":   "dummy-user",
		"account-id": "dummy-account",
	}

	output, err := testutil.RunWithMockServer(http.HandlerFunc(handler), defaults, workflowCmd, []string{"workflow", "apply", tmpFile.Name()})
	require.NoError(t, err)

	assert.Contains(t, output, "Workflow updated with ID: wf-existing-1")
}
