package cmd

import (
	"testing"

	"nudgebee.com/nbctl/pkg/testutil"
)

func TestLogsListLabelValues_Unit(t *testing.T) {
	mockData := map[string]any{
		"fetchLogLabelValues": []map[string]any{
			{
				"value":      "value1",
				"attributes": "{}",
			},
		},
	}

	got, err := testutil.RunWithSimpleGraphQL(mockData, rootCmd, []string{"logs", "list-label-values", "--account-id", "1", "--label-name", "test"})
	if err != nil {
		t.Fatalf("logsListLabelValuesCmd.RunE failed: %v", err)
	}
	if got == "" {
		t.Fatalf("expected output, got empty string")
	}
}

func TestLogsListLabelValues_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	got, err := testutil.RunCommandCaptureOutput(rootCmd, []string{"logs", "list-label-values", "--account-id", "1", "--label-name", "test"})
	if err != nil {
		t.Fatalf("integration logsListLabelValuesCmd failed: %v", err)
	}
	if got == "" {
		t.Fatalf("expected non-empty output from integration run")
	}
}
