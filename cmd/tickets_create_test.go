package cmd

import (
	"encoding/json"
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetTicketsCreateFlags clears every flag on the shared ticketsCreateCmd
// so flag values parsed by a prior test don't leak into the next one.
// Cobra stores parsed values on the *Command, not on test-scoped state,
// so tests must reset them explicitly to be order-independent.
func resetTicketsCreateFlags(t *testing.T) {
	t.Helper()
	for _, f := range []string{
		"account-id", "integration-id", "title", "project-key", "description",
		"ticket-type", "severity", "assignee", "reference-id", "source",
		"additional-fields",
	} {
		require.NoError(t, ticketsCreateCmd.Flags().Set(f, ""))
	}
}

func TestTicketsCreateCmd(t *testing.T) {
	resetTicketsCreateFlags(t)

	mockResponse := map[string]any{
		"tickets_create": map[string]any{
			"data": map[string]any{
				"insert_tickets_one": map[string]any{
					"id":        "nb-1",
					"ticket_id": "JIRA-123",
					"url":       "https://example.atlassian.net/browse/JIRA-123",
					"action":    "created",
					"message":   "ok",
				},
			},
		},
	}

	args := []string{
		"tickets", "create",
		"--integration-id", "int-1",
		"--title", "Pod crashloop",
		"--project-key", "OPS",
	}
	output, err := testutil.RunWithSimpleGraphQL(mockResponse, ticketsCmd, args)
	require.NoError(t, err)

	assert.Contains(t, output, "Ticket created")
	assert.Contains(t, output, "JIRA-123")
	assert.Contains(t, output, "https://example.atlassian.net/browse/JIRA-123")
}

func TestTicketsCreateCmd_JSON(t *testing.T) {
	resetTicketsCreateFlags(t)

	mockResponse := map[string]any{
		"tickets_create": map[string]any{
			"data": map[string]any{
				"insert_tickets_one": map[string]any{
					"id":        "nb-1",
					"ticket_id": "JIRA-123",
					"url":       "https://example.atlassian.net/browse/JIRA-123",
				},
			},
		},
	}

	args := []string{
		"tickets", "create",
		"--integration-id", "int-1",
		"--title", "Pod crashloop",
		"--project-key", "OPS",
		"--format", "json",
	}
	output, err := testutil.RunWithSimpleGraphQL(mockResponse, ticketsCmd, args)
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &result))
	assert.Equal(t, "JIRA-123", result["ticket_id"])
}

func TestTicketsCreateCmd_ServerError(t *testing.T) {
	resetTicketsCreateFlags(t)

	mockResponse := map[string]any{
		"tickets_create": map[string]any{
			"data": map[string]any{
				"insert_tickets_one": map[string]any{
					"id":    "nb-1",
					"error": "Jira: project OPS not found",
				},
			},
		},
	}

	args := []string{
		"tickets", "create",
		"--integration-id", "int-1",
		"--title", "Pod crashloop",
		"--project-key", "OPS",
	}
	_, err := testutil.RunWithSimpleGraphQL(mockResponse, ticketsCmd, args)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Jira: project OPS not found")
}

func TestTicketsCreateCmd_MissingRequiredArg(t *testing.T) {
	resetTicketsCreateFlags(t)

	args := []string{"tickets", "create", "--title", "x", "--project-key", "OPS"}
	_, err := testutil.RunWithSimpleGraphQL(map[string]any{}, ticketsCmd, args)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--integration-id is required")
}

func TestTicketsCreateCmd_NullResponse(t *testing.T) {
	resetTicketsCreateFlags(t)

	mockResponse := map[string]any{
		"tickets_create": map[string]any{
			"data": map[string]any{
				"insert_tickets_one": nil,
			},
		},
	}

	args := []string{
		"tickets", "create",
		"--integration-id", "int-1",
		"--title", "x",
		"--project-key", "OPS",
	}
	_, err := testutil.RunWithSimpleGraphQL(mockResponse, ticketsCmd, args)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "server returned no ticket data")
}

func TestTicketsCreateCmd_BadAdditionalFields(t *testing.T) {
	resetTicketsCreateFlags(t)

	args := []string{
		"tickets", "create",
		"--integration-id", "int-1",
		"--title", "x",
		"--project-key", "OPS",
		"--additional-fields", "not-json",
	}
	_, err := testutil.RunWithSimpleGraphQL(map[string]any{}, ticketsCmd, args)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "valid JSON")
}
