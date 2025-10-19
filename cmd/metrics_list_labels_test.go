package cmd

import (
	"testing"

	"nudgebee.com/nbctl/pkg/testutil"
)

func TestMetricsListLabels_Unit(t *testing.T) {
	mockData := map[string]any{
		"metrics_list_labels": []map[string]any{
			{
				"label":      "label1",
				"attributes": "{}",
			},
		},
	}

	got, err := testutil.RunWithSimpleGraphQL(mockData, rootCmd, []string{"metrics", "list-labels", "--account-id", "1", "--metric", "test"})
	if err != nil {
		t.Fatalf("metricsListLabelsCmd.RunE failed: %v", err)
	}
	if got == "" {
		t.Fatalf("expected output, got empty string")
	}
}

func TestMetricsListLabels_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	got, err := testutil.RunCommandCaptureOutput(rootCmd, []string{"metrics", "list-labels", "--account-id", "1", "--metric", "test"})
	if err != nil {
		t.Fatalf("integration metricsListLabelsCmd failed: %v", err)
	}
	if got == "" {
		t.Fatalf("expected non-empty output from integration run")
	}
}
