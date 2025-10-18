package cmd

import (
	"testing"

	"nudgebee.com/nbctl/pkg/testutil"
)

func TestEventsGet_Unit(t *testing.T) {
	mockData := map[string]any{
		"events": map[string]any{
			"rows": []map[string]any{
				{
					"id":           "1",
					"title":        "Event 1",
					"priority":     "high",
					"status":       "new",
					"subject_name": "subj1",
					"created_at":   "2025-10-15T10:00:00Z",
				},
			},
		},
	}

	got, err := testutil.RunWithSimpleGraphQL(mockData, rootCmd, []string{"events", "get", "1"})
	if err != nil {
		t.Fatalf("eventsGetCmd.RunE failed: %v", err)
	}
	if got == "" {
		t.Fatalf("expected output, got empty string")
	}
}

func TestEventsGet_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	// This test requires an existing event with ID "1" in the test environment.
	// If that's not the case, this test will fail.
	got, err := testutil.RunCommandCaptureOutput(rootCmd, []string{"events", "get", "1"})
	if err != nil {
		t.Fatalf("integration eventsGetCmd failed: %v", err)
	}
	if got == "" {
		t.Fatalf("expected non-empty output from integration run")
	}
}
