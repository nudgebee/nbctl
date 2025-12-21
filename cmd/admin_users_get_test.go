package cmd

import (
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
)

func TestAdminUsersGet(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockData := map[string]any{
			"users_by_pk": map[string]any{
				"id": "test-user-id",
			},
		}
		cmd := adminUsersGetCmd
		args := []string{"admin", "users", "get", "--id", "test-user-id"}
		_, err := testutil.RunWithSimpleGraphQL(mockData, cmd, args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
