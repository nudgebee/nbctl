package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// StartMockServer starts an httptest.Server with the provided handler and
// returns its URL and a teardown function.
func StartMockServer(handler http.HandlerFunc) (string, func()) {
	srv := httptest.NewServer(handler)
	return srv.URL, func() {
		// give a tiny grace so in-flight requests can finish in tests
		time.Sleep(5 * time.Millisecond)
		srv.Close()
	}
}

// WithViper sets keys temporarily on viper and returns a restore function.
func WithViper(overrides map[string]any) func() {
	// snapshot old values
	old := map[string]any{}
	for k := range overrides {
		old[k] = viper.Get(k)
	}

	for k, v := range overrides {
		viper.Set(k, v)
	}

	return func() {
		for k := range overrides {
			if old[k] == nil {
				viper.Set(k, "")
			} else {
				viper.Set(k, old[k])
			}
		}
	}
}

// RunCommandCaptureOutput runs a cobra command and returns its stdout output
// as a string and an error from the command.
func RunCommandCaptureOutput(command *cobra.Command, args []string) (string, error) {
	var buf bytes.Buffer
	if command == nil {
		command = &cobra.Command{}
	}
	rootCmd := command.Root()
	if rootCmd == nil {
		rootCmd = command
	}

	// Reset format persistent flag to default "text"
	_ = rootCmd.PersistentFlags().Set("format", "text")
	// Also reset the global format object state
	format.GetFormat().Set("text")
	format.GetFormat().SetOutput(&buf)

	rootCmd.SetOut(&buf)

	// ensure viper/config is initialized so tests pick up env vars
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	if err != nil {
		// If SilenceErrors is true, we need to manually write the error to the buffer
		// so tests that check 'output' for error messages still pass.
		_, _ = fmt.Fprintln(rootCmd.OutOrStdout(), err)
	}
	return buf.String(), err
}

// RunWithMockServer starts an httptest server with handler, applies viper
// overrides, runs the provided cobra command, and tears down everything.
// It returns the captured output and any error from running the command.
func RunWithMockServer(handler http.HandlerFunc, viperOverrides map[string]any, cmd *cobra.Command, args []string) (string, error) {
	// Reset the singleton client to ensure it picks up the new mock configuration
	client.ResetClient()

	url, teardown := StartMockServer(handler)
	defer teardown()

	// ensure caller's overrides can refer to endpoint placeholder
	if viperOverrides == nil {
		viperOverrides = map[string]any{}
	}
	// if endpoint specified as empty string in overrides, set to server URL
	if _, ok := viperOverrides["endpoint"]; !ok {
		viperOverrides["endpoint"] = url
	}

	restore := WithViper(viperOverrides)
	defer restore()

	return RunCommandCaptureOutput(cmd, args)
}

// RunWithSimpleGraphQL is a convenience helper that starts a mock server which
// automatically handles the token endpoint and serves the provided mockData as
// the GraphQL response payload under the "data" field. It's suitable for the
// common case where tests only need to mock returned data.
func RunWithSimpleGraphQL(mockData any, cmd *cobra.Command, args []string) (string, error) {
	_ = os.Setenv("NBCTL_TESTING", "true")
	defer func() { _ = os.Unsetenv("NBCTL_TESTING") }()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth/token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"token": "fake-token", "expiry": 3600})
			return
		case "/api/graphql":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"data": mockData})
			return
		default:
			http.NotFound(w, r)
			return
		}
	})

	// Provide sensible defaults for tests: a dummy api-key and username. The
	// endpoint will be set by RunWithMockServer to the mock server URL.
	defaults := map[string]any{
		"api-key":    "dummy",
		"username":   "dummy-user",
		"account-id": "dummy-account",
	}

	return RunWithMockServer(handler, defaults, cmd, args)
}

// RunWithSimpleGraphQLAndInput runs a command with mock GraphQL and simulated stdin input.
func RunWithSimpleGraphQLAndInput(mockData any, cmd *cobra.Command, args []string, input string) (string, error) {
	_ = os.Setenv("NBCTL_TESTING", "true")
	defer func() { _ = os.Unsetenv("NBCTL_TESTING") }()

	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	_, _ = w.WriteString(input)
	_ = w.Close()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	return RunWithSimpleGraphQL(mockData, cmd, args)
}

// IntegrationEnabled returns true when integration tests are enabled via
// the NUDGEBEE_INTEGRATION env var.
func IntegrationEnabled() bool {
	return os.Getenv("NUDGEBEE_INTEGRATION") == "1"
}

// RequireIntegration checks environment and calls t.Skip when integration
// tests should not run. It also initializes config so viper reads env vars.
func RequireIntegration(t testing.TB) {
	if !IntegrationEnabled() {
		t.Skip("integration tests disabled; set NUDGEBEE_INTEGRATION=1 to enable")
	}

	// quick check for required env vars
	if os.Getenv("NUDGEBEE_ENDPOINT") == "" || os.Getenv("NUDGEBEE_API_KEY") == "" {
		t.Skip("integration tests skipped; NUDGEBEE_ENDPOINT and NUDGEBEE_API_KEY must be set")
	}
}
