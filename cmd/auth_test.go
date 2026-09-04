package cmd

import (
	"strings"
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
)

func TestAuthGroupsList_Unit(t *testing.T) {
	mockData := map[string]any{
		"usergroups_list": map[string]any{
			"rows": []map[string]any{
				{
					"id":           "ug-1",
					"name":         "DevOps",
					"description":  "DevOps Team",
					"group_roles":  `[{"role":"tenant_admin"}]`,
					"member_count": 5,
					"created_at":   "2026-01-01T00:00:00Z",
				},
				{
					"id":           "ug-2",
					"name":         "SRE",
					"description":  "SRE Team",
					"group_roles":  []map[string]any{{"role": "account_admin"}},
					"member_count": 3,
					"created_at":   "2026-01-02T00:00:00Z",
				},
			},
		},
	}

	got, err := testutil.RunWithSimpleGraphQL(mockData, rootCmd, []string{"auth", "groups", "list"})
	if err != nil {
		t.Fatalf("auth groups list failed: %v", err)
	}
	if got == "" {
		t.Fatalf("expected output, got empty string")
	}

	expected := []string{"Group Name", "Assigned Roles", "DevOps", "tenant_admin", "SRE", "account_admin"}
	for _, exp := range expected {
		if !strings.Contains(got, exp) {
			t.Errorf("expected output to contain %q, got %q", exp, got)
		}
	}
}

func TestAuthRolesList_Unit(t *testing.T) {
	mockData := map[string]any{
		"roles_list": []map[string]any{
			{
				"display_name": "Admin",
				"value":        "tenant_admin",
			},
		},
		"customroles_list": map[string]any{
			"roles": []map[string]any{
				{
					"id":          "cr-1",
					"name":        "Security Auditor",
					"description": "Read-only access to audit logs",
				},
			},
		},
	}

	got, err := testutil.RunWithSimpleGraphQL(mockData, rootCmd, []string{"auth", "roles", "list"})
	if err != nil {
		t.Fatalf("auth roles list failed: %v", err)
	}
	if got == "" {
		t.Fatalf("expected output, got empty string")
	}

	expected := []string{"Built-in", "Admin", "tenant_admin", "Custom", "Security Auditor"}
	for _, exp := range expected {
		if !strings.Contains(got, exp) {
			t.Errorf("expected output to contain %q, got %q", exp, got)
		}
	}
}

func TestAuthUsersList_Unit(t *testing.T) {
	mockData := map[string]any{
		"users_list_by_tenant": map[string]any{
			"rows": []map[string]any{
				{
					"id":           "user-1",
					"username":     "alice@example.com",
					"display_name": "Alice Smith",
					"status":       "active",
					"created_at":   "2026-01-01T00:00:00Z",
				},
			},
		},
	}

	got, err := testutil.RunWithSimpleGraphQL(mockData, rootCmd, []string{"auth", "users", "list"})
	if err != nil {
		t.Fatalf("auth users list failed: %v", err)
	}
	if got == "" {
		t.Fatalf("expected output, got empty string")
	}

	expected := []string{"Username", "Display Name", "alice@example.com", "Alice Smith"}
	for _, exp := range expected {
		if !strings.Contains(got, exp) {
			t.Errorf("expected output to contain %q, got %q", exp, got)
		}
	}
}

func TestAuthUsersGet_Unit(t *testing.T) {
	mockData := map[string]any{
		"users_list_by_tenant": map[string]any{
			"rows": []map[string]any{
				{
					"id":           "user-1",
					"username":     "alice@example.com",
					"display_name": "Alice Smith",
					"status":       "active",
					"created_at":   "2026-01-01T00:00:00Z",
				},
			},
		},
	}

	got, err := testutil.RunWithSimpleGraphQL(mockData, rootCmd, []string{"auth", "users", "get", "alice@example.com"})
	if err != nil {
		t.Fatalf("auth users get failed: %v", err)
	}
	if got == "" {
		t.Fatalf("expected output, got empty string")
	}

	if !strings.Contains(got, "alice@example.com") {
		t.Errorf("expected output to contain alice@example.com, got %q", got)
	}
}

func TestAuthRolesCreate_Unit(t *testing.T) {
	mockData := map[string]any{
		"customroles_create": map[string]any{
			"id": "cr-100",
		},
	}

	got, err := testutil.RunWithSimpleGraphQL(mockData, rootCmd, []string{
		"auth", "roles", "create", "test-role",
		"--description", "test role description",
		"--permission", "events:read",
		"--permission", "logs",
	})
	if err != nil {
		t.Fatalf("auth roles create failed: %v", err)
	}
	if got == "" {
		t.Fatalf("expected output, got empty string")
	}

	expected := []string{"cr-100", "test-role", "created"}
	for _, exp := range expected {
		if !strings.Contains(got, exp) {
			t.Errorf("expected output to contain %q, got %q", exp, got)
		}
	}
}
