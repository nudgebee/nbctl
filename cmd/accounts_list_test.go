package cmd

import (
	"strings"
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
)

// Unit test: mock the token and GraphQL endpoints with httptest.Server
func TestAccountsList_Unit(t *testing.T) {
	// mock server that handles token and graphql paths
	mockData := map[string]any{
		"cloud_accounts": map[string]any{
			"rows": []map[string]any{
				{
					"id":             "1",
					"account_name":   "ac1",
					"account_type":   "type1",
					"cloud_provider": "aws",
					"status":         "active",
					"created_at":     "2025-10-17T00:00:00Z",
				},
			},
		},
	}

	got, err := testutil.RunWithSimpleGraphQL(mockData, rootCmd, []string{"accounts", "list"})
	if err != nil {
		t.Fatalf("accountsListCmd.RunE failed: %v", err)
	}
	if got == "" {
		t.Fatalf("expected output, got empty string")
	}

	// Check for the new columns in the output
	expectedColumns := []string{"Account Name", "Cloud Provider"}
	for _, col := range expectedColumns {
		if !strings.Contains(got, col) {
			t.Errorf("expected output to contain column %q, got %q", col, got)
		}
	}
}
