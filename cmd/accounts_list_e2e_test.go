//go:build e2e

package cmd

import (
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
)

// Integration test: run only when NUDGEBEE_INTEGRATION=1 and required env vars
// (NUDGEBEE_ENDPOINT and NUDGEBEE_API_KEY) are provided. This test relies on
// `config.InitConfig()` which will make viper pick up `NUDGEBEE_`-prefixed
// environment variables (the project config already calls viper.AutomaticEnv()).
func TestAccountsList_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	got, err := testutil.RunCommandCaptureOutput(rootCmd, []string{"accounts", "list"})
	if err != nil {
		t.Fatalf("integration accountsListCmd failed: %v", err)
	}
	if got == "" {
		t.Fatalf("expected non-empty output from integration run")
	}
}
