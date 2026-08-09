package cmd

import (
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
)

func TestEventsList_Unit(t *testing.T) {
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

	got, err := testutil.RunWithSimpleGraphQL(mockData, rootCmd, []string{"events", "list"})
	if err != nil {
		t.Fatalf("eventsListCmd.RunE failed: %v", err)
	}
	if got == "" {
		t.Fatalf("expected output, got empty string")
	}
}
