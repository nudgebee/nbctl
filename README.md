# nbctl: Nudgebee CLI

[![CI](https://github.com/nudgebee/nbctl/actions/workflows/ci.yml/badge.svg)](https://github.com/nudgebee/nbctl/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/nudgebee/nbctl)](https://github.com/nudgebee/nbctl/releases/latest)
[![License](https://img.shields.io/github/license/nudgebee/nbctl)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/nudgebee/nbctl)](go.mod)

`nbctl` is a powerful command-line interface for seamless interaction with the Nudgebee API. It allows you to manage various Nudgebee resources directly from your terminal, providing a convenient way to automate tasks, query data, and integrate with your workflows.

## Features

*   **Comprehensive API Access**: Interact with Nudgebee accounts, events, logs, metrics, traces, optimizations, security recommendations, SLOs, workflows, tickets, and audit logs.
*   **Workflow Lifecycle**: Apply, trigger, pause/resume, cancel, and inspect workflow executions from the command line.
*   **Events & Troubleshooting**: List events, inspect event-manager rules and triage rules, and explore deduplication chains.
*   **Tickets**: Create tickets in external systems (Jira, ServiceNow, PagerDuty, GitHub, GitLab, Zenduty) through configured integrations.
*   **Nubi Integration**: Create and manage Nubi (Nudgebee's AI assistant) configurations and run an interactive AI shell.
*   **MCP Server**: Expose all `nbctl` capabilities as Model Context Protocol tools for use with Claude Desktop and other LLM clients.
*   **Flexible Output**: View results in human-readable text or machine-parseable JSON format.
*   **Shell Autocompletion**: Generate completion scripts for Bash, Zsh, Fish, and PowerShell for enhanced productivity.
*   **Configurable Logging**: Control the verbosity and level of logging for debugging and monitoring.
*   **Profile Management**: Easily switch between different Nudgebee environments or user accounts using named profiles.

## Installation

To install `nbctl`, ensure you have Go installed (version 1.25 or higher). Then, run the following command:

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

## Quickstart

Once `nbctl` is on your `PATH`, get to your first command in under a minute:

```bash
# 1. Configure your credentials (interactive — you'll need an API key from your Nudgebee profile)
nbctl configure add default

# 2. Verify everything works
nbctl accounts list

# 3. Try something useful — events from the last 24 hours
nbctl events list --limit 10
```

Useful follow-ups from here:

```bash
# Inspect a specific event (replace with an ID from the list above)
nbctl events get <event-id>

# Look at the rules that fire events
nbctl events list-rules --limit 10

# See what optimization recommendations exist for your cluster
nbctl optimizations list --limit 10

# Trigger a workflow
nbctl workflow list
nbctl workflow trigger <workflow-id>
```

Most commands accept `--format json` for scripting and `--help` for full flag documentation. See the [Usage](#usage) section below for the full command reference.

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

## Model Context Protocol (MCP) Integration

`nbctl` can act as an MCP server, allowing LLM clients (like Claude Desktop or Gemini) to interact with your Nudgebee resources using natural language.

### Prerequisites

1.  Configure `nbctl` with your credentials using `nbctl configure add`.
2.  Ensure `nbctl` is in your `PATH`.

### Claude Desktop Configuration

Add the following to your Claude Desktop configuration file (usually `~/Library/Application Support/Claude/claude_desktop_config.json` on macOS or `%APPDATA%\Claude\claude_desktop_config.json` on Windows):

```json
{
  "mcpServers": {
    "nudgebee": {
      "command": "nbctl",
      "args": ["mcp"]
    }
  }
}
```

Once configured, restart Claude Desktop. You can then ask questions like "List my Nudgebee accounts" or "Show me high severity security recommendations".

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

### Resource Management Commands

#### `nbctl admin`

Perform administrative tasks on the Nudgebee platform.

##### `nbctl admin users`

Manages user accounts.

###### `nbctl admin users list`

Lists all users with options to filter and paginate results.

*   **Flags**:
    *   `--offset <int>`: Offset for pagination. Default is 0.
    *   `--limit <int>`: Limit for pagination. Default is 20.
    *   `--name <string>`: Filter by name (case-insensitive like).
    *   `--username <string>`: Filter by username (case-insensitive like).
    *   `--status <string>`: Filter by status (exact match).
    *   `--id <string>`: Filter by user ID (exact match).

Example:

```bash
nbctl admin users list --limit 50 --status active
```

###### `nbctl admin users get`

Retrieves details for a single user by ID or username.

*   **Flags**:
    *   `--id <string>`: The unique identifier of the user.
    *   `--username <string>`: The username of the user.
    *   *Note: Either `--id` or `--username` is required.*

Example:

```bash
nbctl admin users get --username "john.doe@example.com"
```

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

#### `nbctl audit-logs`

Query audit events recorded for the current account.

##### `nbctl audit-logs list`

Lists audit events, ordered by most recent first.

*   **Flags**:
    *   `--account-id <id>`: Account ID (overrides the active profile).
    *   `--username <string>`: Filter by username (exact match).
    *   `--category <string>`: Filter by event category (e.g. `AUTO_PILOT`).
    *   `--type <string>`: Filter by event type.
    *   `--action <string>`: Filter by event action (e.g. `UPDATE`).
    *   `--status <string>`: Filter by event status (e.g. `SUCCESS`).
    *   `--start-time <RFC3339>`: Filter events after this timestamp.
    *   `--end-time <RFC3339>`: Filter events before this timestamp.
    *   `--limit <int>`: Maximum number of events to return. Default is 25.
    *   `--offset <int>`: Pagination offset. Default is 0.

Example:

```bash
nbctl audit-logs list --category AUTO_PILOT --status SUCCESS --limit 50
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

##### `nbctl events list-rules`

Lists event-manager rules (Prometheus / alert-manager style rules that determine which conditions raise an event).

*   **Flags**:
    *   `--account-id <id>`: Account ID (overrides the active profile).
    *   `--name <string>`: Filter by alert name (case-insensitive partial match).
    *   `--limit <int>`: Maximum number of rules to return. Default is 25.
    *   `--offset <int>`: Pagination offset. Default is 0.

Example:

```bash
nbctl events list-rules --name "PostgreSQL" --limit 10
```

##### `nbctl events list-triage-rules`

Lists event triage rules — both system rules and user-defined rules that classify or route incoming events.

*   **Flags**:
    *   `--account-id <id>`: Account ID (overrides the active profile).
    *   `--rule-type <string>`: Filter by rule type (e.g. `scoring`, `suppression`).
    *   `--enabled`: Filter by enabled status. Pass `--enabled` to filter enabled rules; `--enabled=false` for disabled.

Example:

```bash
nbctl events list-triage-rules --enabled --rule-type scoring
```

##### `nbctl events list-duplicates <event-id>`

Shows the deduplication chain for an event — every occurrence the system has linked together via fingerprint matching. Prints a friendly message when the event has no duplicates.

*   **Arguments**:
    *   `<event-id>` (required): The unique identifier of the event.

Example:

```bash
nbctl events list-duplicates 2934c445-f776-4a8b-9234-a1faefe71058
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

Starts an interactive shell session or executes a single query with Nudgebee AI to ask questions and get insights.

*   **Usage**: `nbctl nubi [account-id] [flags]`
*   **Arguments**:
    *   `[account-id]` (optional): The account ID to use for the Nubi session. If not provided, `nbctl` will attempt to use the `default-account-id` from your configuration.
*   **Flags**:
    *   `-q, --query <string>`: Execute a single query non-interactively and exit.
    *   `--async`: Trigger the query asynchronously without waiting for the response (used with `--query`).
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

**Examples:**

*Start interactive session:*
```bash
nbctl nubi
nbctl nubi my-dev-account-id
```

*Single query mode (synchronous):*
```bash
nbctl nubi -q "What is the health status of my services?"
```

*Single query mode (asynchronous):*
```bash
nbctl nubi -q "Analyze latest deployment logs" --async
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

#### `nbctl security`

Manage and query security-related resources.

##### `nbctl security list-image-cves`

Lists security recommendations and CVEs for container images.

*   **Flags**:
    *   `--account-id <id>` (required): The account ID. If not provided, it attempts to read it from the configuration.
    *   `--limit <int>`: Number of recommendations to retrieve. Default is 10.
    *   `--offset <int>`: Offset for pagination. Default is 0.
    *   `--severity <string>`: Filter by severity (e.g., Critical, High, Medium, Low).
    *   `--namespace <string>`: Filter by namespace (case-insensitive like).
    *   `--workload <string>`: Filter by workload name (case-insensitive like).
    *   `--cve <string>`: Filter by CVE ID (case-insensitive like).
    *   `--package-id <string>`: Filter by package ID (case-insensitive like).
    *   `--image <string>`: Filter by image name (case-insensitive like).

Example:

```bash
nbctl security list-image-cves --severity Critical --namespace production
```

#### `nbctl slo`

Query Service Level Objective (SLO) configurations and compliance reports.

##### `nbctl slo list`

Lists SLO configurations for the current account.

*   **Flags**:
    *   `--account-id <id>`: Account ID (overrides the active profile).
    *   `--workload <string>`: Filter by workload name (exact match).
    *   `--namespace <string>`: Filter by workload namespace (exact match).
    *   `--enabled`: Filter by enabled status. Pass `--enabled` for enabled SLOs; `--enabled=false` for disabled.

Example:

```bash
nbctl slo list --namespace nudgebee-test --enabled
```

##### `nbctl slo get <id>`

Fetches a single SLO configuration. Use `--report` to also fetch the latest compliance report (burn rate, event budgets, status).

*   **Arguments**:
    *   `<id>` (required): The unique identifier of the SLO config.
*   **Flags**:
    *   `--account-id <id>`: Account ID (overrides the active profile).
    *   `--report`: Also fetch the latest compliance report.

Example:

```bash
nbctl slo get a5f07f40-d2f0-4edf-91c7-96ddb0b25ae8 --report
```

#### `nbctl tickets`

List ticket integration configurations and create tickets in external systems (Jira, ServiceNow, PagerDuty, GitHub, GitLab, Zenduty).

##### `nbctl tickets list-configurations`

Lists configured ticket integrations for the current account. Filters to ticket-capable integration types by default.

*   **Flags**:
    *   `--type <string>`: Filter by a single integration type (`jira`, `github`, `gitlab`, `servicenow`, `pagerduty`, `zenduty`). Validated client-side.
    *   `--status <string>`: Filter by status (e.g. `enabled`).

Example:

```bash
nbctl tickets list-configurations --type jira --status enabled
```

##### `nbctl tickets create`

Creates a ticket via a configured integration.

*   **Flags**:
    *   `--integration-id <id>` (required): The integration configuration ID. Use `tickets list-configurations` to find it.
    *   `--title <string>` (required): Ticket title.
    *   `--project-key <string>` (required): Project / board key in the external system.
    *   `--description <string>`: Ticket description / body.
    *   `--ticket-type <string>`: Issue type (e.g. `Bug`, `Task`).
    *   `--severity <string>`: Severity / priority.
    *   `--assignee <string>`: Assignee identifier.
    *   `--reference-id <string>`: Reference linking this ticket to a Nudgebee event.
    *   `--source <string>`: Source system identifier.
    *   `--additional-fields <json>`: Additional integration-specific fields as a JSON object.

Example:

```bash
nbctl tickets create \
  --integration-id 100d48eb-6503-4bf1-8474-be3441804284 \
  --title "Pod crashloop in prod" \
  --project-key OPS \
  --severity high \
  --reference-id <event-id>
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

##### `nbctl traces query`

Queries Nudgebee traces over a time range with optional filters. Defaults to the last 24 hours and returns up to 50 spans ordered by timestamp (descending).

*   **Flags**:
    *   `--workload-name <strings>`: Filter by workload name. A single value matches as an `ilike` substring; pass the flag multiple times for an `in` match.
    *   `--span-name <strings>`: Filter by span name (`in` match).
    *   `--trace-id <strings>`: Filter by trace id (`in` match).
    *   `--resource <string>`: Filter by resource (`like` substring).
    *   `--status-code <string>`: Filter by status code (exact match).
    *   `--http-status-code <strings>`: Filter by HTTP status code (`in` match).
    *   `--start-time <RFC3339>`: Start time. Defaults to 24 hours ago.
    *   `--end-time <RFC3339>`: End time. Defaults to the current time.

Example:

```bash
nbctl traces query --workload-name traefik --start-time 2026-05-24T00:00:00Z
```

#### `nbctl integrations`

Lists tenant integrations (Jira, Slack, PagerDuty, webhooks, etc.).

##### `nbctl integrations list`

Lists configured integrations.

*   **Flags**:
    *   `--status <string>`: Filter by status (exact match).
    *   `--type <string>`: Filter by integration type (exact match, e.g. `jira`, `pagerduty_webhook`).
    *   `--name <string>`: Filter by name (`ilike` substring).
    *   `--limit <int>`: Pagination limit. Default is 20.
    *   `--offset <int>`: Pagination offset. Default is 0.

Example:

```bash
nbctl integrations list --type jira --status enabled
```

#### `nbctl workflow`

Manage workflows: list, inspect, apply (create or update from YAML), trigger executions, pause/resume, cancel in-flight executions, and inspect execution history.

##### `nbctl workflow list`

Lists all workflows.

*   **Flags**:
    *   `--account-id <id>`: Account ID (overrides the active profile).
    *   `--status <string>`: Filter by workflow status (e.g. `ACTIVE`, `PAUSED`).
    *   `--last-execution-status <string>`: Filter by last execution status.
    *   `--type <string>`: Filter by workflow type.

Example:

```bash
nbctl workflow list --status ACTIVE
```

##### `nbctl workflow get <id>`

Fetches workflow details including the full definition.

Example:

```bash
nbctl workflow get 6e8e1a30-a306-4635-90c5-ac16611f3677
```

##### `nbctl workflow apply <file>`

Creates or updates a workflow from a YAML file (upsert).

Example:

```bash
nbctl workflow apply ./my-workflow.yaml
```

##### `nbctl workflow validate <file>`

Server-side validation of a workflow YAML without creating or updating anything.

Example:

```bash
nbctl workflow validate ./my-workflow.yaml
```

##### `nbctl workflow delete <id>`

Deletes a workflow by ID.

Example:

```bash
nbctl workflow delete 6e8e1a30-a306-4635-90c5-ac16611f3677
```

##### `nbctl workflow trigger <id>`

Triggers a workflow execution. Returns the new execution ID.

*   **Flags**:
    *   `-i, --input <key=value>`: Pass workflow inputs. Repeatable.

Example:

```bash
nbctl workflow trigger 6e8e1a30-a306-4635-90c5-ac16611f3677 -i name=alice -i count=3
```

##### `nbctl workflow pause <id>` / `nbctl workflow resume <id>`

Pause an active workflow (stops accepting new triggers) or resume a paused one.

Example:

```bash
nbctl workflow pause 6e8e1a30-a306-4635-90c5-ac16611f3677
nbctl workflow resume 6e8e1a30-a306-4635-90c5-ac16611f3677
```

##### `nbctl workflow list-executions <workflow-id>`

Lists execution history for a specific workflow with optional filters and pagination.

*   **Flags**:
    *   `--account-id <id>`: Account ID (overrides the active profile).
    *   `--status <string>`: Filter by execution status (e.g. `COMPLETED`, `FAILED`).
    *   `--trigger-type <string>`: Filter by trigger type.
    *   `--limit <int>`: Maximum number of executions to return.
    *   `--page-token <string>`: Pagination token from a previous response.

Example:

```bash
nbctl workflow list-executions 6e8e1a30-a306-4635-90c5-ac16611f3677 --status COMPLETED --limit 10
```

##### `nbctl workflow get-execution <workflow-id> <execution-id>`

Fetches a single workflow execution including its tasks, inputs, outputs, and any errors.

Example:

```bash
nbctl workflow get-execution 6e8e1a30-a306-4635-90c5-ac16611f3677 019e8cde-38f4-75e2-93cb-de8278183926
```

##### `nbctl workflow cancel-execution <workflow-id> <execution-id>`

Cancels a specific in-progress execution.

Example:

```bash
nbctl workflow cancel-execution 6e8e1a30-a306-4635-90c5-ac16611f3677 019e8cde-38f4-75e2-93cb-de8278183926
```

## Testing

Unit tests use the helpers in `pkg/testutil` to mock the API and avoid network calls. Integration (end-to-end) tests live behind a build tag and are only compiled when explicitly requested.

- Run unit tests:

```bash
make test
# or
go test ./...
```

- Run end-to-end tests against a live Nudgebee environment:

```bash
export NUDGEBEE_ENDPOINT=https://app.nudgebee.com
export NUDGEBEE_API_KEY=<your_api_key>
make test-e2e
# or
NUDGEBEE_INTEGRATION=1 go test -tags=e2e ./...
```

Individual e2e tests will `t.Skip` themselves if `NUDGEBEE_ENDPOINT` or `NUDGEBEE_API_KEY` is unset, so they're safe to run without env vars (they just won't actually exercise anything).

## Contributing

We welcome contributions. See [CONTRIBUTING.md](./CONTRIBUTING.md)
for the development setup, commit conventions, and pull-request
workflow. All participants are expected to follow our
[Code of Conduct](./CODE_OF_CONDUCT.md).

## Security

For vulnerability reporting, see [SECURITY.md](./SECURITY.md).
**Do not open public issues for security problems.**

## License

`nbctl` is released under the Apache License 2.0. See
[LICENSE](./LICENSE).
