package cmd

import (
	"strings"
	"testing"

	"nudgebee.com/nbctl/pkg/testutil"
)

// Unit test: mock the token and GraphQL endpoints with httptest.Server
func TestAccountsList_Unit(t *testing.T) {
	// mock server that handles token and graphql paths
	mockData := map[string]any{
		"cloud_accounts": []map[string]any{
			{
				"id":                  "1",
				"account_name":        "ac1",
				"account_type":        "type1",
				"cloud_provider":      "aws",
				"status":              "active",
				"created_at":          "2025-10-17T00:00:00Z",
				"agents":              []any{},
				"cloud_account_attrs": []any{},
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
	expectedColumns := []string{"Agents", "Attributes"}
	for _, col := range expectedColumns {
		if !strings.Contains(got, col) {
			t.Errorf("expected output to contain column %q, got %q", col, got)
		}
	}
}

// Integration test: run only when NUDGEBEE_INTEGRATION=1 and required env vars
// (NUDGEBEE_ENDPOINT and NUDGEBEE_API_KEY) are provided. This test relies on
// `config.InitConfig()` which will make viper pick up `NUDGEBEE_`-prefixed
// environment variables (the project config already calls viper.AutomaticEnv()).
func TestAccountsList_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	got, err := testutil.RunCommandCaptureOutput(rootCmd, []string{"accounts", "list"})
	if err != nil {
		t.Fatalf("integration accountsListCmd failed: %v", err)
	}
	if got == "" {
		t.Fatalf("expected non-empty output from integration run")
	}
}
