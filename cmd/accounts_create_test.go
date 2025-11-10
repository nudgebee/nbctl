package cmd

import (
	"strings"
	"testing"

	"nudgebee.com/nbctl/pkg/testutil"
)

func TestAccountsCreate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockData := map[string]any{
			"cloud_accounts_insert_one": map[string]any{
				"id": "new-account-id",
			},
		}
		cmd := accountsCreateCmd
		args := []string{"accounts", "create", "--name", "test-account", "--account-type", "kubernetes"}
		output, err := testutil.RunWithSimpleGraphQL(mockData, cmd, args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(output, "new-account-id") {
			t.Fatalf("expected output to contain 'new-account-id', got %s", output)
		}
	})
}
