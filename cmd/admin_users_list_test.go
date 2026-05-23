package cmd

import (
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
)

func TestAdminUsersList(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockData := map[string]any{
			"users": map[string]any{
				"rows": []any{},
			},
		}
		cmd := adminUsersListCmd
		args := []string{"admin", "users", "list"}
		_, err := testutil.RunWithSimpleGraphQL(mockData, cmd, args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
