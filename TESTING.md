Testing guidelines for nbctl

This project includes helpers to make unit and integration tests easy and
consistent.

Helpers
-------
The package `pkg/testutil` exposes the following useful helpers:

- `RunWithSimpleGraphQL(mockData any, cmd *cobra.Command, args []string) (string, error)`
  - Convenience for mocking a GraphQL response. Automatically mocks `/api/auth/token` and
    returns `{ "data": mockData }` at `/api/graphql`. Useful for simple, static tests.

- `RunWithMockServer(handler http.HandlerFunc, viperOverrides map[string]any, cmd *cobra.Command, args []string) (string, error)`
  - More flexible: provide a handler to simulate complex behavior, and a small map of viper overrides (e.g. `api-key`, `username`). The helper sets the `endpoint` viper key to the test server URL and restores previous viper values after the test.

- `RunCommandCaptureOutput(cmd *cobra.Command, args []string) (string, error)`
  - Runs a cobra command and returns stdout output and error. It also initializes the configuration (so viper reads `NUDGEBEE_` env vars) before running.

- `RequireIntegration(t testing.TB)`
  - Call at the top of integration tests. It skips the test unless `NUDGEBEE_INTEGRATION=1` and required env vars (`NUDGEBEE_ENDPOINT` and `NUDGEBEE_API_KEY`) are present. It also ensures configuration is initialized.

Patterns
--------
- Unit tests (fast): use `RunWithSimpleGraphQL` where possible. Example:

```go
mockData := map[string]any{ /* shape returned by your query */ }
output, err := testutil.RunWithSimpleGraphQL(mockData, myCmd, []string{})
```

- Complex mocks: use `RunWithMockServer` and provide a handler that inspects request
  paths/headers and returns appropriate responses.

- Integration tests (manual): at the top of the test call `testutil.RequireIntegration(t)`
  so the test will gracefully skip unless running with the correct environment.

CI Guidance
-----------
- Run unit tests on every PR: `go test ./...`
- Run integration tests in a separate workflow/job when secrets are available, or
  using a manual workflow dispatch with environment variables set.

Examples: see `cmd/accounts_list_command_test.go` for a simple unit + gated integration example.
