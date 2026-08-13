package cmd

import (
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNubiCmd_AsyncQuery(t *testing.T) {
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
