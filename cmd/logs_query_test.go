package cmd

import (
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
)

func TestLogsQuery_Unit(t *testing.T) {
	mockData := map[string]any{
		"logs_query": []map[string]any{
			{
				"timestamp": "2025-10-15T10:00:00Z",
				"severity":  "info",
				"message":   "Log message 1",
				"labels":    "{}",
			},
		},
	}

	got, err := testutil.RunWithSimpleGraphQL(mockData, rootCmd, []string{"logs", "query", "--account-id", "1", "--query", "test"})
	if err != nil {
		t.Fatalf("logsQueryCmd.RunE failed: %v", err)
	}
	if got == "" {
		t.Fatalf("expected output, got empty string")
	}
}

func TestLogsQuery_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	got, err := testutil.RunCommandCaptureOutput(rootCmd, []string{"logs", "query", "--account-id", "1", "--query", "test"})
	if err != nil {
		t.Fatalf("integration logsQueryCmd failed: %v", err)
	}
	if got == "" {
		t.Fatalf("expected non-empty output from integration run")
	}
}
