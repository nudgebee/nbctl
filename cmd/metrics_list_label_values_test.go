package cmd

import (
	"testing"

	"nudgebee.com/nbctl/pkg/testutil"
)

func TestMetricsListLabelValues_Unit(t *testing.T) {
	mockData := map[string]any{
		"metrics_list_label_values": []map[string]any{
			{
				"value":      "value1",
				"attributes": "{}",
			},
		},
	}

	got, err := testutil.RunWithSimpleGraphQL(mockData, rootCmd, []string{"metrics", "list-label-values", "--account-id", "1", "--label", "test"})
	if err != nil {
		t.Fatalf("metricsListLabelValuesCmd.RunE failed: %v", err)
	}
	if got == "" {
		t.Fatalf("expected output, got empty string")
	}
}

func TestMetricsListLabelValues_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	got, err := testutil.RunCommandCaptureOutput(rootCmd, []string{"metrics", "list-label-values", "--account-id", "1", "--label", "test"})
	if err != nil {
		t.Fatalf("integration metricsListLabelValuesCmd failed: %v", err)
	}
	if got == "" {
		t.Fatalf("expected non-empty output from integration run")
	}
}
