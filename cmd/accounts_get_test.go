package cmd

import (
	"strings"
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
)

func TestAccountsGet_Unit(t *testing.T) {
	mockData := map[string]any{
		"cloud_accounts": map[string]any{
			"rows": []map[string]any{
				{
					"id":              "00000000-0000-0000-0000-000000000001",
					"account_name":    "ac1",
					"account_type":    "cloud",
					"cloud_provider":  "aws",
					"status":          "active",
					"account_number":  "1234567890",
					"created_at":      "2025-10-17T00:00:00Z",
					"synced_at":       "2025-10-17T00:00:00Z",
					"agent_synced_at": "2025-10-17T00:00:00Z",
				},
			},
		},
		"agents": map[string]any{
			"rows": []any{},
		},
	}

	got, err := testutil.RunWithSimpleGraphQL(mockData, rootCmd, []string{"accounts", "get", "00000000-0000-0000-0000-000000000001"})
	if err != nil {
		t.Fatalf("accountsGetCmd.RunE failed: %v", err)
	}
	if got == "" {
		t.Fatalf("expected output, got empty string")
	}

	// Check for the new fields in the output
	expectedFields := []string{"AccountNumber", "CreatedAt", "SyncedAt", "AgentSyncedAt"}
	for _, field := range expectedFields {
		if !strings.Contains(got, field) {
			t.Errorf("expected output to contain %q, got %q", field, got)
		}
	}
}

func TestAccountsGet_Unit_K8s(t *testing.T) {
	mockData := map[string]any{
		"cloud_accounts": map[string]any{
			"rows": []map[string]any{
				{
					"id":           "00000000-0000-0000-0000-000000000001",
					"account_name": "k8s-1",
					"account_type": "kubernetes",
					"status":       "active",
				},
			},
		},
		"agents": map[string]any{
			"rows": []map[string]any{
				{
					"version":           "1.0.0",
					"status":            "CONNECTED",
					"last_connected_at": "2025-10-17T00:00:00Z",
					"k8s_provider":      "minikube",
					"k8s_version":       "1.23.0",
					"connection_status": `{"relayConnection": true}`,
				},
			},
		},
	}

	got, err := testutil.RunWithSimpleGraphQL(mockData, rootCmd, []string{"accounts", "get", "00000000-0000-0000-0000-000000000001"})
	if err != nil {
		t.Fatalf("accountsGetCmd.RunE failed: %v", err)
	}
	if got == "" {
		t.Fatalf("expected output, got empty string")
	}

	// Check for the new fields in the output
	expectedFields := []string{"Agents", "Features", "ScheduledJobs"}
	for _, field := range expectedFields {
		if !strings.Contains(got, field) {
			t.Errorf("expected output to contain %q, got %q", field, got)
		}
	}
}

