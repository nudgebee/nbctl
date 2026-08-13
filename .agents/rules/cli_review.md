# CLI & Command Development Guardrails

When creating or modifying Cobra CLI commands, flags, or API integration logic, ALWAYS adhere to the following rules locally before opening or updating PRs:

1. **Dynamic Flag Retrieval**:
   - Never bind Cobra flags to package-level global variables (`var myFlag string`).
   - Register flags in `init()` using `cmd.Flags().String(...)` or `cmd.Flags().Bool(...)`.
   - Retrieve flag values dynamically inside `RunE` via `cmd.Flags().GetString("flag")` or `cmd.Flags().GetBool("flag")`.

2. **Flag Dependencies & Input Validation**:
   - If flag B requires flag A, validate using `cmd.Flags().Changed("A")` and return an explicit error (e.g. `--async requires --query / -q`).
   - If a flag is explicitly provided (via `cmd.Flags().Changed("flag")`), trim whitespace (`strings.TrimSpace`). If it is empty, return `fmt.Errorf("<flag> cannot be empty")` instead of silently falling back to default or interactive modes.

3. **Context Propagation & Exit Codes**:
   - Always pass `ctx` (from `cmd.Context()`) to downstream API calls. Never use `context.Background()`.
   - Check context errors with `errors.Is(err, context.Canceled)`.
   - Return `nil` from `RunE` after printing user-facing cancellation messages to prevent Cobra from outputting redundant `Error: context canceled` stack traces.

4. **Signal Handling**:
   - Background signal-handling goroutines listening for `SIGINT`/`SIGTERM` must ONLY call `cancel()`.
   - Never call `os.Exit(1)` or `spinner.Stop()` directly inside a background signal goroutine. Allow `defer cancel()` and deferred spinner cleanups on the main thread to execute.

5. **URL Construction**:
   - Trim trailing slashes from base endpoint URLs using `strings.TrimSuffix(endpoint, "/")` before building URLs.

6. **Test Isolation**:
   - Reset Cobra command flags (`f.Value.Set("")`, `f.Changed = false`) and `viper` settings in `t.Cleanup(...)` for every test function.
   - Always include test coverage for both synchronous and asynchronous query paths.
