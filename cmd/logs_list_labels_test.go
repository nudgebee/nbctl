package cmd

import (
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
)

func TestLogsListLabels_Unit(t *testing.T) {
	mockData := map[string]any{
		"logs_list_labels": []map[string]any{
			{
				"label":      "label1",
				"attributes": "{}",
			},
		},
	}

	got, err := testutil.RunWithSimpleGraphQL(mockData, rootCmd, []string{"logs", "list-labels", "--account-id", "1"})
	if err != nil {
		t.Fatalf("logsListLabelsCmd.RunE failed: %v", err)
	}
	if got == "" {
		t.Fatalf("expected output, got empty string")
	}
}
