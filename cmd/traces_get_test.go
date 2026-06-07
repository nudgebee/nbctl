package cmd

import (
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
)

func TestTracesGet_Unit(t *testing.T) {
	mockData := map[string]any{
		"traces_list": []map[string]any{
			{
				"trace_id":      "1",
				"span_id":       "s1",
				"workload_name": "test-workload",
				"timestamp":     "2025-10-15T10:00:00Z",
				"duration_ns":   100,
				"status_code":   "OK",
				"span_name":     "test-span",
			},
		},
	}

	got, err := testutil.RunWithSimpleGraphQL(mockData, rootCmd, []string{"traces", "get", "1"})
	if err != nil {
		t.Fatalf("tracesGetCmd.RunE failed: %v", err)
	}
	if got == "" {
		t.Fatalf("expected output, got empty string")
	}
}

