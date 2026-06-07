package cmd

import (
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
)

func TestOptimizationsList_Unit(t *testing.T) {
	mockData := map[string]any{
		"recommendation": map[string]any{
			"rows": []map[string]any{
				{
					"workload_name":        "test-workload",
					"namespace":            "test-namespace",
					"account_id":           "1",
					"recommendation_count": 1,
				},
			},
		},
	}

	got, err := testutil.RunWithSimpleGraphQL(mockData, rootCmd, []string{"optimizations", "list"})
	if err != nil {
		t.Fatalf("optimizationsListCmd.RunE failed: %v", err)
	}
	if got == "" {
		t.Fatalf("expected output, got empty string")
	}
}

