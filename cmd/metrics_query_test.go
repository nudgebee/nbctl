package cmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"nudgebee.com/nbctl/pkg/testutil"
)

func TestMetricsQueryCmd(t *testing.T) {
	mockData := map[string]any{
		"metrics_query": map[string]any{
			"results": []map[string]any{
				{
					"query_key": "test_query",
					"payload": []map[string]any{
						{
							"metric":     map[string]string{"__name__": "test_metric"},
							"timestamps": []float64{1666100000},
							"values":     []float64{42},
						},
					},
				},
			},
		},
	}

	// Execute the command with the mock GraphQL response
	queryJSON := `{"query":"test_metric"}`
	args := []string{
		"metrics", "query",
		"--account-id", "test-account",
		"--query", queryJSON,
		"--start-time", "2022-10-18T12:00:00Z",
		"--end-time", "2022-10-18T12:05:00Z",
	}

	got, err := testutil.RunWithSimpleGraphQL(mockData, rootCmd, args)
	assert.NoError(t, err)
	assert.True(t, strings.Contains(got, "test_metric"))
	assert.True(t, strings.Contains(got, "42"))
}
