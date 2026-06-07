//go:build e2e

package cmd

import (
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
)

func TestMetricsListLabels_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	got, err := testutil.RunCommandCaptureOutput(rootCmd, []string{"metrics", "list-labels", "--account-id", "1", "--metric", "test"})
	if err != nil {
		t.Fatalf("integration metricsListLabelsCmd failed: %v", err)
	}
	if got == "" {
		t.Fatalf("expected non-empty output from integration run")
	}
}
