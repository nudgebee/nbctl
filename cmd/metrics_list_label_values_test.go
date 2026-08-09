package cmd

import (
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
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
