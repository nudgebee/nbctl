package cmd

import (
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
)

func TestMetricsListMetrics_Unit(t *testing.T) {
	mockData := map[string]any{
		"metrics_list": []map[string]any{
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

