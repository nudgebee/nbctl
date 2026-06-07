//go:build e2e

package cmd

import (
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
)

func TestEventsGet_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	// This test requires an existing event with ID "1" in the test environment.
	// If that's not the case, this test will fail.
	got, err := testutil.RunCommandCaptureOutput(rootCmd, []string{"events", "get", "1"})
	if err != nil {
		t.Fatalf("integration eventsGetCmd failed: %v", err)
	}
	if got == "" {
		t.Fatalf("expected non-empty output from integration run")
	}
}
