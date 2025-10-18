package cmd

import (
	"testing"

	"nudgebee.com/nbctl/pkg/testutil"
)

func TestMetricsListMetrics_Unit(t *testing.T) {
	mockData := map[string]any{
		"fetchMetricsList": []map[string]any{
			{
				"metric":     "metric1",
				"attributes": "{}",
			},
		},
	}

	got, err := testutil.RunWithSimpleGraphQL(mockData, rootCmd, []string{"metrics", "list-metrics", "--account-id", "1"})
	if err != nil {
		t.Fatalf("metricsListMetricsCmd.RunE failed: %v", err)
	}
	if got == "" {
		t.Fatalf("expected output, got empty string")
	}
}

func TestMetricsListMetrics_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	got, err := testutil.RunCommandCaptureOutput(rootCmd, []string{"metrics", "list-metrics", "--account-id", "1"})
	if err != nil {
		t.Fatalf("integration metricsListMetricsCmd failed: %v", err)
	}
	if got == "" {
		t.Fatalf("expected non-empty output from integration run")
	}
}
