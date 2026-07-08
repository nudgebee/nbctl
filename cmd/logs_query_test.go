package cmd

import (
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
)

func TestLogsQuery_Unit(t *testing.T) {
	mockData := map[string]any{
		"logs_list": map[string]any{
			"logs": []map[string]any{
				{
					"timestamp": "2025-10-15T10:00:00Z",
					"severity":  "info",
					"message":   "Log message 1",
					"labels":    "{}",
				},
			},
			"query":    "test",
			"provider": "nudgebee",
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

