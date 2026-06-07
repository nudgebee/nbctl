package cmd

import (
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
)

func TestOptimizationsGet_Unit(t *testing.T) {
	mockData := map[string]any{
		"recommendations_list": map[string]any{
			"rows": []map[string]any{
				{
					"id":                   "00000000-0000-0000-0000-000000000001",
					"workload_name":        "test-workload",
					"namespace":            "test-namespace",
					"account_id":           "1",
					"recommendation_count": 1,
					"recommendation":       "{}",
				},
			},
		},
	}

	got, err := testutil.RunWithSimpleGraphQL(mockData, rootCmd, []string{"optimizations", "get", "00000000-0000-0000-0000-000000000001"})
	if err != nil {
		t.Fatalf("optimizationsGetCmd.RunE failed: %v", err)
	}
	if got == "" {
		t.Fatalf("expected output, got empty string")
	}
}

