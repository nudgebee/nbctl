package cmd

import (
	"encoding/json"
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

	// Test invalid permission format (empty module)
	_, errInvalid := testutil.RunWithSimpleGraphQL(mockData, rootCmd, []string{
		"auth", "roles", "create", "test-role",
		"--permission", ":read",
	})
	if errInvalid == nil {
		t.Fatalf("expected error for empty permission module, got nil")
	} else if !strings.Contains(errInvalid.Error(), "module cannot be empty") {
		t.Errorf("expected 'module cannot be empty' error, got: %v", errInvalid)
	}
}

func TestGroupRolesField_UnmarshalJSON(t *testing.T) {
	// Test direct array
	var g1 groupRolesField
	if err := json.Unmarshal([]byte(`[{"role":"admin","entity_type":"tenant","entity_id":"t-1"}]`), &g1); err != nil {
		t.Fatalf("unexpected error for direct array: %v", err)
	}
	if len(g1) != 1 || g1[0].Role != "admin" {
		t.Errorf("unexpected content for direct array: %+v", g1)
	}

	// Test stringified array
	var g2 groupRolesField
	if err := json.Unmarshal([]byte(`"[{\"role\":\"viewer\"}]"`), &g2); err != nil {
		t.Fatalf("unexpected error for stringified array: %v", err)
	}
	if len(g2) != 1 || g2[0].Role != "viewer" {
		t.Errorf("unexpected content for stringified array: %+v", g2)
	}

	// Test empty string
	var g3 groupRolesField
	if err := json.Unmarshal([]byte(`""`), &g3); err != nil {
		t.Fatalf("unexpected error for empty string: %v", err)
	}
	if len(g3) != 0 {
		t.Errorf("expected 0 items for empty string, got %d", len(g3))
	}

	// Test invalid structure returns meaningful array unmarshal error
	var g4 groupRolesField
	if err := json.Unmarshal([]byte(`[123]`), &g4); err == nil {
		t.Fatalf("expected error for invalid array element, got nil")
	} else if !strings.Contains(err.Error(), "cannot unmarshal number") {
		t.Errorf("expected error about number unmarshaling, got: %v", err)
	}
}
