package cmd

import (
	"strings"
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
)

func TestIntegrationsList_Unit(t *testing.T) {
	mockData := map[string]any{
		"integrations": map[string]any{
			"rows": []map[string]any{
				{
					"id":                      "00000000-0000-0000-0000-000000000001",
					"name":                    "jira-prod",
					"type":                    "jira",
					"source":                  "ticketing",
					"status":                  "active",
					"created_by_display_name": "Alice",
					"updated_at":              "2025-10-17T00:00:00Z",
				},
			},
		},
	}

	got, err := testutil.RunWithSimpleGraphQL(mockData, rootCmd, []string{"integrations", "list"})
	if err != nil {
		t.Fatalf("integrationsListCmd.RunE failed: %v", err)
	}
	if !strings.Contains(got, "jira-prod") {
		t.Fatalf("expected output to contain 'jira-prod', got %s", got)
	}
}

