package cmd

import (
	"testing"

	"nudgebee.com/nbctl/pkg/testutil"
)

func TestLogsListLabels_Unit(t *testing.T) {
	mockData := map[string]any{
		"fetchLogLabel": []map[string]any{
			{
				"label":      "label1",
				"attributes": "{}",
			},
		},
	}

	got, err := testutil.RunWithSimpleGraphQL(mockData, rootCmd, []string{"logs", "list-labels", "--account-id", "1"})
	if err != nil {
		t.Fatalf("logsListLabelsCmd.RunE failed: %v", err)
	}
	if got == "" {
		t.Fatalf("expected output, got empty string")
	}
}

func TestLogsListLabels_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	got, err := testutil.RunCommandCaptureOutput(rootCmd, []string{"logs", "list-labels", "--account-id", "1"})
	if err != nil {
		t.Fatalf("integration logsListLabelsCmd failed: %v", err)
	}
	if got == "" {
		t.Fatalf("expected non-empty output from integration run")
	}
}
