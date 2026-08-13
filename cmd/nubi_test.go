package cmd

import (
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNubiCmd_AsyncQuery(t *testing.T) {
	_ = nubiCmd.Flags().Set("query", "")
	_ = nubiCmd.Flags().Set("async", "false")

	mockResponse := map[string]interface{}{
		"ai_execute_investigation": map[string]interface{}{
			"data": map[string]interface{}{
				"response": "ok",
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, nubiCmd, []string{"nubi", "test-account-id", "-q", "hello", "--async"})
	require.NoError(t, err)

	assert.Contains(t, output, "Investigation triggered asynchronously.")
	assert.Contains(t, output, "Session ID:")
}

func TestNubiCmd_AsyncWithoutQuery(t *testing.T) {
	_ = nubiCmd.Flags().Set("query", "")
	_ = nubiCmd.Flags().Set("async", "false")

	_, err := testutil.RunWithSimpleGraphQL(nil, nubiCmd, []string{"nubi", "test-account-id", "--async"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--async requires --query / -q")
}
