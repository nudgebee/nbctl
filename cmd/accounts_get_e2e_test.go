//go:build e2e

package cmd

import (
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
)

func TestAccountsGet_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	// This test requires an existing account with ID "1" in the test environment.
	// If that's not the case, this test will fail.
	// We can make this more robust by first creating an account and then getting it.
	// For now, we assume the account exists.
	got, err := testutil.RunCommandCaptureOutput(rootCmd, []string{"accounts", "get", "1"})
	if err != nil {
		t.Fatalf("integration accountsGetCmd failed: %v", err)
	}
	if got == "" {
		t.Fatalf("expected non-empty output from integration run")
	}
}
