package cmd

import (
	"strings"
	"testing"

	"nudgebee.com/nbctl/pkg/testutil"
)

func TestAccountsEnable(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockData := map[string]any{
			"update_cloud_accounts": map[string]any{
				"affected_rows": 1,
			},
		}
		cmd := accountsEnableCmd
		args := []string{"accounts", "enable", "test-account-id"}
		output, err := testutil.RunWithSimpleGraphQL(mockData, cmd, args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(output, "Account test-account-id enabled") {
			t.Fatalf("expected output to contain 'Account test-account-id enabled', got %s", output)
		}
	})
}
