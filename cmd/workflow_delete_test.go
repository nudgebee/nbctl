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

func TestWorkflowDeleteCmd(t *testing.T) {
	_ = os.Setenv("NBCTL_TESTING", "true")
	defer func() { _ = os.Unsetenv("NBCTL_TESTING") }()

	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth/token" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"token": "fake-token", "expiry": 3600})
			return
		}
		if r.URL.Path == "/api/graphql" {
			var reqBody struct {
				Query string `json:"query"`
			}
			_ = json.NewDecoder(r.Body).Decode(&reqBody)

			w.Header().Set("Content-Type", "application/json")
			if strings.Contains(reqBody.Query, "deleteWorkflow") {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"data": map[string]any{
						"workflow_delete": map[string]any{
							"message": "Workflow deleted successfully",
						},
					},
				})
			} else {
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

	output, err := testutil.RunWithMockServer(http.HandlerFunc(handler), defaults, workflowCmd, []string{"workflow", "delete", "wf-123"})
	require.NoError(t, err)

	assert.Contains(t, output, "Workflow deleted successfully")
}

func TestWorkflowDeleteCmd_JSON(t *testing.T) {
	_ = os.Setenv("NBCTL_TESTING", "true")
	defer func() { _ = os.Unsetenv("NBCTL_TESTING") }()

	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth/token" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"token": "fake-token", "expiry": 3600})
			return
		}
		if r.URL.Path == "/api/graphql" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"workflow_delete": map[string]any{
						"message": "Workflow deleted successfully",
					},
				},
			})
			return
		}
		http.NotFound(w, r)
	}

	defaults := map[string]any{
		"api-key":    "dummy",
		"username":   "dummy-user",
		"account-id": "dummy-account",
	}

	output, err := testutil.RunWithMockServer(http.HandlerFunc(handler), defaults, workflowCmd, []string{"workflow", "delete", "wf-123", "--format", "json"})
	require.NoError(t, err)

	assert.JSONEq(t, `{"message": "Workflow deleted successfully"}`, output)
}
