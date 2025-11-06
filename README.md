# nbctl: Nudgebee CLI

`nbctl` is a powerful command-line interface for seamless interaction with the Nudgebee API. It allows you to manage various Nudgebee resources directly from your terminal, providing a convenient way to automate tasks, query data, and integrate with your workflows.

## Features

*   **Comprehensive API Access**: Interact with Nudgebee accounts, events, logs, metrics, optimizations, and traces.
*   **Nubi Integration**: Create and manage Nubi (Nudgebee's AI assistant) configurations.
*   **Flexible Output**: View results in human-readable text or machine-parseable JSON format.
*   **Shell Autocompletion**: Generate completion scripts for Bash, Zsh, Fish, and PowerShell for enhanced productivity.
*   **Configurable Logging**: Control the verbosity and level of logging for debugging and monitoring.
*   **Profile Management**: Easily switch between different Nudgebee environments or user accounts using named profiles.

## Installation

To install `nbctl`, ensure you have Go installed (version 1.16 or higher). Then, run the following command:

```bash
go install github.com/nudgebee/nbctl@latest
```

This will download, compile, and install the `nbctl` executable in your Go binary path (usually `$GOPATH/bin` or `$HOME/go/bin`). Make sure this directory is in your system's `PATH` to run `nbctl` from any location.

### Download Binary

You can also download the pre-compiled binary directly from the [GitHub Releases page](https://github.com/nudgebee/nbctl/releases/latest).

**Linux (64-bit):**

```bash
curl -LO https://github.com/nudgebee/nbctl/releases/latest/download/nbctl-linux-amd64
chmod +x nbctl-linux-amd64
sudo mv nbctl-linux-amd64 /usr/local/bin/nbctl
```

**macOS (Apple Silicon):**

```bash
curl -LO https://github.com/nudgebee/nbctl/releases/latest/download/nbctl-darwin-arm64
chmod +x nbctl-darwin-arm64
sudo mv nbctl-darwin-arm64 /usr/local/bin/nbctl
```

After downloading, make sure the binary is executable and moved to a directory included in your system's `PATH` (e.g., `/usr/local/bin`).

## Configuration

Before using most `nbctl` commands, you need to configure your Nudgebee API credentials. `nbctl` supports managing multiple configuration profiles, allowing you to easily switch between different Nudgebee environments or user accounts.

`nbctl` stores its configuration in a YAML file located at `~/.nudgebee/config.yaml`.

### Managing Configuration Profiles

#### `nbctl configure add [profile-name]`

This command interactively guides you through setting up a new configuration profile or updating an existing one. If `profile-name` is not provided, it defaults to `default`. You will be prompted for:

*   **Nudgebee API Endpoint**: The URL of the Nudgebee API (e.g., `https://api.nudgebee.com`).
*   **Nudgebee API Key**: Your personal API key for authentication.
*   **Nudgebee Username**: Your Nudgebee account username (e.g., your email).
*   **Default Account ID**: The ID of the Nudgebee account you wish to interact with by default.

After collecting the information, `nbctl` will attempt to validate your credentials by making a test API call.

Example:

```bash
nbctl configure add my-dev-profile
```

#### `nbctl configure set-current <profile-name>`

Sets the specified profile as the active one. All subsequent `nbctl` commands will use the credentials and settings from this profile.

Example:

```bash
nbctl configure set-current my-prod-profile
```

#### `nbctl configure list`

Displays a list of all configured profiles, indicating which one is currently active.

Example:

```bash
nbctl configure list
```

#### `nbctl configure current`

Shows the name of the currently active configuration profile.

Example:

```bash
nbctl configure current
```

### Configuration File Location

The default configuration file is `~/.nudgebee/config.yaml`.

### Persistent Flags

The following flags can be used with any `nbctl` command:

*   `--log-level <level>`: Sets the logging level. Accepted values are `debug`, `info` (default), `warn`, and `error`.
*   `--verbose`: Enables verbose logging, including detailed GraphQL requests and responses. Useful for debugging API interactions.
*   `--format <format>`: Specifies the output format for command results. Currently, `json` is supported in addition to the default human-readable `text` format.

    Example:
    ```bash
    nbctl accounts list --format json
    ```

## Usage

The general syntax for `nbctl` commands is:

```bash
nbctl [command] [subcommand] [flags]
```

### Global Commands

#### `nbctl completion [bash|zsh|fish|powershell]`

Generates shell completion scripts for your preferred shell. This significantly improves productivity by allowing you to autocomplete commands, subcommands, and flags by pressing `Tab`.

**To load completions for your current session:**

```bash
# Bash
source <(nbctl completion bash)

# Zsh
source <(nbctl completion zsh)

# Fish
nbctl completion fish | source

# PowerShell
nbctl completion powershell | Out-String | Invoke-Expression
```

**To make completions persistent across sessions:**

```bash
# Bash (Linux)
nbctl completion bash > /etc/bash_completion.d/nbctl

# Bash (macOS)
nbctl completion bash > /usr/local/etc/bash_completion.d/nbctl

# Zsh
echo "autoload -U compinit; compinit" >> ~/.zshrc
nbctl completion zsh > "${fpath[1]}/_nbctl"
# You will need to start a new shell for this setup to take effect.

# Fish
nbctl completion fish > ~/.config/fish/completions/nbctl.fish

# PowerShell
nbctl completion powershell > nbctl.ps1
# Then, source this file from your PowerShell profile.
```

#### `nbctl version`

Prints the version number of the `nbctl` CLI tool.

Example:

```bash
nbctl version
```

#### `nbctl configure`

### Resource Management Commands

#### `nbctl accounts`

Manages Nudgebee cloud accounts.

##### `nbctl accounts create`

Creates a new Nudgebee account.

*   **Flags**:
    *   `--name <string>` (required): The name of the account.
    *   `--cloud-provider <string>` (optional): The cloud provider for the account. If `--account-type` is `kubernetes`, this defaults to `K8s`. Otherwise, it is required.
    *   `--account-type <string>` (required): The type of the account (e.g., "kubernetes").
    *   `--env <string>` (optional): The environment of the account. Accepted values are `dev` (default) or `prod`.

Example:

```bash
nbctl accounts create --name "my-k8s-account" --account-type "kubernetes"
```

```bash
nbctl accounts create --name "my-aws-account" --cloud-provider "aws" --account-type "cloud" --env "prod"
```

##### `nbctl accounts create-agent-token <account-id>`

Creates an agent token for a specified Kubernetes account. This token is used by the Nudgebee agent to authenticate with the Nudgebee API.

*   **Arguments**:
    *   `<account-id>` (required): The unique identifier of the Kubernetes account for which to create the agent token.

Example:

```bash
nbctl accounts create-agent-token 3163f884-4c09-4f18-be2c-ca769c14c3f4
```

##### `nbctl accounts enable <id>`

Enables a Nudgebee account by setting its status to "enabled".

*   **Arguments**:
    *   `<id>` (required): The unique identifier of the account to enable.

Example:

```bash
nbctl accounts enable 123e4567-e89b-12d3-a456-426614174000
```

##### `nbctl accounts disable <id>`

Disables a Nudgebee account by setting its status to "disabled".

*   **Arguments**:
    *   `<id>` (required): The unique identifier of the account to disable.

Example:

```bash
nbctl accounts disable 123e4567-e89b-12d3-a456-426614174000
```

##### `nbctl accounts get <id>`

Retrieves detailed information for a specific Nudgebee account.

*   **Arguments**:
    *   `<id>` (required): The unique identifier of the account.

Example:

```bash
nbctl accounts get 123e4567-e89b-12d3-a456-426614174000
```

##### `nbctl accounts list`

Lists all Nudgebee accounts, with options to filter the results.

*   **Flags**:
    *   `--status <status>`: Filters accounts by their current status (e.g., "active", "inactive").
    *   `--cloud-provider <provider>`: Filters accounts by the cloud provider (e.g., "aws", "azure", "gcp").
    *   `--name <name>`: Filters accounts by name, supporting partial matches.

Example:

```bash
nbctl accounts list --status active --cloud-provider aws --name "production"
```

#### `nbctl events`

Manages Nudgebee events, allowing you to list, search, and describe them. If no subcommand is provided, it defaults to `list`.

*   **Persistent Flags (for `events` and its subcommands)**:
    *   `--account-id <id>`: Filters events by a specific account ID. This flag is required for most event operations.
    *   `--start-time <RFC3339>`: Filters events that occurred after or at this time. Defaults to 24 hours ago.
    *   `--end-time <RFC3339>`: Filters events that occurred before or at this time. Defaults to the current time.
    *   `--limit <int>`: Limits the number of events returned. Default is 10.
    *   `--offset <int>`: Specifies an offset for pagination, skipping the first `N` events. Default is 0.
    *   `--subject <string>`: Filters events by subject (case-insensitive partial match).
    *   `--status <string>`: Filters events by their status.
    *   `--title <string>`: Filters events by title (case-insensitive partial match).
    *   `--priority <string>`: Filters events by their priority.

##### `nbctl events get <id>`

Retrieves detailed information for a specific Nudgebee event.

*   **Arguments**:
    *   `<id>` (required): The unique identifier of the event.

Example:

```bash
nbctl events get 987e6543-e21f-43d2-b109-876543210000 --account-id 123e4567-e89b-12d3-a456-426614174000
```

##### `nbctl events list`

Lists Nudgebee events, with various filtering and pagination options.

*   **Flags**: All persistent flags defined for `nbctl events` can be used with `nbctl events list`.

Example:

```bash
nbctl events list --account-id 123e4567-e89b-12d3-a456-426614174000 --status "open" --start-time "2023-10-26T00:00:00Z" --limit 50
```

#### `nbctl logs`

Queries logs from the Nudgebee API.

##### `nbctl logs list-label-values`

Lists all possible values for a given log label within a specified time range and account.

*   **Flags**:
    *   `--label-name <name>` (required): The name of the log label for which to retrieve values.
    *   `--account-id <id>`: Filters by a specific account ID. If not provided, it attempts to read it from the configuration.
    *   `--start-time <RFC3339>`: Filters logs starting from this time. Defaults to 1 hour ago.
    *   `--end-time <RFC3339>`: Filters logs up to this time. Defaults to the current time.

Example:

```bash
nbctl logs list-label-values --label-name "app" --account-id 123e4567-e89b-12d3-a456-426614174000
```

##### `nbctl logs list-labels`

Lists all available log labels within a specified time range and account.

*   **Flags**:
    *   `--account-id <id>`: Filters by a specific account ID. If not provided, it attempts to read it from the configuration.
    *   `--start-time <RFC3339>`: Filters logs starting from this time. Defaults to 1 hour ago.
    *   `--end-time <RFC3339>`: Filters logs up to this time. Defaults to the current time.

Example:

```bash
nbctl logs list-labels --account-id 123e4567-e89b-12d3-a456-426614174000 --start-time "2023-10-26T00:00:00Z"
```

##### `nbctl logs query`

Queries logs from the Nudgebee API based on various filters.

*   **Flags**:
    *   `--account-id <id>` (required): The account ID to query logs from. If not provided, it attempts to read it from the configuration.
    *   `--start-time <RFC3339>`: Filters logs starting from this time. Defaults to 1 hour ago.
    *   `--end-time <RFC3339>`: Filters logs up to this time. Defaults to the current time.
    *   `--query <string>`: The log query string (e.g., `level=error`, `app=my-app`).
    *   `--limit <int>`: Limits the number of log entries returned. Default is 100.
    *   `--offset <int>`: Specifies an offset for pagination. Default is 0.
    *   `--only-message`: If set, only the log messages are displayed, without timestamp, severity, or labels.

Example:

```bash
nbctl logs query --account-id 123e4567-e89b-12d3-a456-426614174000 --query "level=error" --start-time "2023-10-26T00:00:00Z" --limit 20 --only-message
```

#### `nbctl metrics`

Queries metrics from the Nudgebee API.

##### `nbctl metrics list-label-values`

Lists all possible values for a given metric label within a specified account.

*   **Flags**:
    *   `--label <name>` (required): The name of the metric label for which to retrieve values.
    *   `--account-id <id>`: Filters by a specific account ID. If not provided, it attempts to read it from the configuration.

Example:

```bash
nbctl metrics list-label-values --label "instance" --account-id 123e4567-e89b-12d3-a456-426614174000
```

##### `nbctl metrics list-labels`

Lists all available labels for a given metric within a specified account.

*   **Flags**:
    *   `--metric <name>` (required): The name of the metric for which to retrieve labels.
    *   `--account-id <id>`: Filters by a specific account ID. If not provided, it attempts to read it from the configuration.

Example:

```bash
nbctl metrics list-labels --metric "node_cpu_usage" --account-id 123e4567-e89b-12d3-a456-426614174000
```

##### `nbctl metrics list-metrics`

Lists all available metrics within a specified account.

*   **Flags**:
    *   `--account-id <id>`: Filters by a specific account ID. If not provided, it attempts to read it from the configuration.

Example:

```bash
nbctl metrics list-metrics --account-id 123e4567-e89b-12d3-a456-426614174000
```

##### `nbctl metrics query`

Queries metrics from the Nudgebee API based on a PromQL-like query string and various filters.

*   **Flags**:
    *   `--query <query-string>` (required): The PromQL-like query string to fetch metrics (e.g., `node_cpu_usage{instance="my-instance"}`).
    *   `--account-id <id>`: The account ID to query metrics from. If not provided, it attempts to read it from the configuration.
    *   `--start-time <RFC3339>`: Filters metrics starting from this time. Defaults to 1 hour ago.
    *   `--end-time <RFC3339>`: Filters metrics up to this time. Defaults to the current time.
    *   `--metric-provider <provider>`: Filters metrics by a specific metric provider.
    *   `--only-metric`: If set, only the metric names are displayed, without attributes.

Example:

```bash
nbctl metrics query --account-id 123e4567-e89b-12d3-a456-426614174000 --query "node_memory_usage_bytes" --start-time "2023-10-26T00:00:00Z"
```

#### `nbctl nubi`

Starts an interactive shell session with Nudgebee AI to ask questions and get insights.

*   **Usage**: `nbctl nubi [account-id]`
*   **Arguments**:
    *   `[account-id]` (optional): The account ID to use for the Nubi session. If not provided, `nbctl` will attempt to use the `default-account-id` from your configuration.
*   **Prerequisites**: Your `account-id` and `username` must be configured (see `nbctl configure`).

**Interactive Features (within the Nubi shell):**

The Nubi shell supports various slash commands to manage your session and interact with the AI:

*   `/help`: Displays a list of all available slash commands and their descriptions.
*   `/bookmarks [add|remove|list] [conversationId]`: Manages your bookmarked conversations.
    *   `add [conversationId]`: Bookmarks the current or specified conversation.
    *   `remove [conversationId]`: Removes a bookmark from the current or specified conversation.
    *   `list`: Lists all your bookmarked conversations.
*   `/conversation <id>`: Switches the current interactive session to a previously existing conversation identified by its ID.
*   `/history [n]`: Shows a list of your last `n` conversations. Defaults to 10 if `n` is not specified.
*   `/account <id>`: Switches the active Nudgebee account for the current Nubi session.
*   `/agents`: Lists all available AI agents that Nubi can utilize.
*   `/tools`: Lists all available tools that Nubi can use.
*   `/functions`: Lists all available functions that Nubi can execute.
*   `/exit`: Exits the Nubi interactive shell.

**Example:**

```bash
nbctl nubi
```

```bash
nbctl nubi my-dev-account-id
```

#### `nbctl optimizations`

Manages Nudgebee optimizations. If no subcommand is provided, it defaults to `list`.

##### `nbctl optimizations get <id>`

Retrieves detailed information for a specific Nudgebee optimization.

*   **Arguments**:
    *   `<id>` (required): The unique identifier of the optimization.

Example:

```bash
nbctl optimizations get 123e4567-e89b-12d3-a456-426614174000
```

##### `nbctl optimizations list`

Lists Nudgebee optimizations, with various filtering and pagination options.

*   **Flags**:
    *   `--account-id <id>` (required): The account ID to query optimizations from. If not provided, it attempts to read it from the configuration.
    *   `--category <string>`: Filters optimizations by category.
    *   `--rule-name <string>`: Filters optimizations by rule name.
    *   `--status <string>`: Filters optimizations by status.
    *   `--limit <int>`: Limits the number of optimizations returned. Default is 10.
    *   `--offset <int>`: Specifies an offset for pagination. Default is 0.

Example:

```bash
nbctl optimizations list --account-id 123e4567-e89b-12d3-a456-426614174000 --category "cost" --status "active"
```

#### `nbctl traces`

Queries traces from the Nudgebee API.

##### `nbctl traces get <trace_id>`

Retrieves detailed information for a specific Nudgebee trace.

*   **Arguments**:
    *   `<trace_id>` (required): The unique identifier of the trace.

Example:

```bash
nbctl traces get 123e4567-e89b-12d3-a456-426614174000
```

##### `nbctl traces list`

Lists Nudgebee traces.

*   **Flags**: None currently defined in the code.

Example:

```bash
nbctl traces list
```

## Testing

Unit tests use the helpers in `pkg/testutil` to mock the API and avoid network calls. Integration tests are gated and will only run when explicitly enabled.

- Run unit tests:

```bash
go test ./...
```

- Run integration tests (manual):

```bash
export NUDGEBEE_INTEGRATION=1
export NUDGEBEE_ENDPOINT=https://app.nudgebee.com
export NUDGEBEE_API_KEY=<your_api_key>
go test ./... -run Integration
```
