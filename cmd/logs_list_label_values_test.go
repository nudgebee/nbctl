package cmd

import (
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
)

func TestLogsListLabelValues_Unit(t *testing.T) {
	mockData := map[string]any{
		"logs_list_label_values": []map[string]any{
			{
				"value":      "value1",
				"attributes": "{}",
			},
		},
	}

	got, err := testutil.RunWithSimpleGraphQL(mockData, rootCmd, []string{"logs", "list-label-values", "--account-id", "1", "--label-name", "test"})
	if err != nil {
		t.Fatalf("logsListLabelValuesCmd.RunE failed: %v", err)
	}
	if got == "" {
		t.Fatalf("expected output, got empty string")
	}
}

