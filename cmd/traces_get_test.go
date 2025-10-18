package cmd

import (
	"testing"

	"nudgebee.com/nbctl/pkg/testutil"
)

func TestTracesGet_Unit(t *testing.T) {
	mockData := map[string]any{
		"traces_by_pk": map[string]any{
			"trace_id":       "1",
			"service_name":   "test-service",
			"operation_name": "test-operation",
			"start_time":     "2025-10-15T10:00:00Z",
			"duration":       100,
			"status_code":    "OK",
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

func TestTracesGet_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	got, err := testutil.RunCommandCaptureOutput(rootCmd, []string{"traces", "get", "1"})
	if err != nil {
		t.Fatalf("integration tracesGetCmd failed: %v", err)
	}
	if got == "" {
		t.Fatalf("expected non-empty output from integration run")
	}
}
