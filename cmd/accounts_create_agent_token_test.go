package cmd

import (
	"strings"
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
)

func TestAccountsCreateAgentToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockData := map[string]any{
			"agents_create_token": map[string]any{
				"access_key": "new-agent-token",
			},
		}
		cmd := accountsCreateAgentTokenCmd
		args := []string{"accounts", "create-agent-token", "test-account-id"}
		output, err := testutil.RunWithSimpleGraphQL(mockData, cmd, args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(output, "new-agent-token") {
			t.Fatalf("expected output to contain 'new-agent-token', got %s", output)
		}
	})
}
