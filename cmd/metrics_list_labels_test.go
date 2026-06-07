package cmd

import (
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
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

