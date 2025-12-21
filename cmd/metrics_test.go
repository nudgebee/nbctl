package cmd

import (
	"strings"
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
)

func TestMetrics_NoSubcommand(t *testing.T) {
	restore := testutil.WithViper(map[string]any{
		"endpoint":   "dummy",
		"api-key":    "dummy",
		"username":   "dummy",
		"account-id": "dummy",
	})
	defer restore()

	got, err := testutil.RunCommandCaptureOutput(rootCmd, []string{"metrics"})
	if err != nil {
		t.Fatalf("metricsCmd.Run failed: %v", err)
	}

	if !strings.Contains(got, "Usage:") {
		t.Fatalf("expected output to contain 'Usage:', got %q", got)
	}
}
