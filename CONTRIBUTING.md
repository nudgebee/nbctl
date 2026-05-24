# Contributing to nbctl

Thanks for your interest in contributing to `nbctl`, the Nudgebee
command-line interface. This document explains how to file issues,
propose changes, and get a pull request merged.

By contributing, you agree that your contributions will be licensed
under the Apache License, Version 2.0 (see [LICENSE](./LICENSE)).

## Code of Conduct

All participation in this project is governed by our
[Code of Conduct](./CODE_OF_CONDUCT.md). Please read it before
engaging in issues, discussions, or pull requests. Report unacceptable
behavior to **legal@nudgebee.com**.

## Ways to Contribute

- **Report a bug** — open a GitHub issue using the bug template.
- **Request a feature** — open a GitHub issue using the feature
  template and describe the use case.
- **Improve documentation** — typo fixes, clarifications, and new
  guides are always welcome.
- **Submit code** — see the workflow below.

## Project Layout

`nbctl` is a single Go module that wraps the Nudgebee GraphQL API.

```
cmd/                Cobra command definitions (one file per command)
pkg/client/         GraphQL client, auth/token handling
pkg/config/         Profile + viper config loading
pkg/format/         Tabular / JSON output rendering
pkg/log/            Logger helpers
pkg/nubi/           Interactive AI shell client
pkg/testutil/       Test helpers (mock GraphQL server, integration gate)
.github/workflows/  CI (lint+test) and release (cross-platform builds)
Makefile            Common dev tasks
```

## Development Setup

Prerequisites:

- Go 1.25 or later
- `make`
- `golangci-lint` (installed automatically when you run `make lint`)

Clone and build:

```bash
git clone https://github.com/nudgebee/nbctl.git
cd nbctl
make build              # outputs dist/nbctl
./dist/nbctl --help
```

Common Makefile targets:

| Target | What it does |
|---|---|
| `make build` | Build a local binary at `dist/nbctl` |
| `make test` | Run unit tests with race detector and coverage |
| `make lint` | Run `golangci-lint` |
| `make fmt` | Run `gofmt -s -w .` |
| `make validate` | `fmt` + `lint` + `test` — run this before pushing |
| `make benchmark` | Run benchmarks |
| `make install` | `go install` to `$GOPATH/bin` |

## Development Workflow

1. **Fork** the repository and create a feature branch from `main`:
   ```bash
   git checkout -b feat/short-description
   ```
   Branch naming: `feat/...`, `fix/...`, `docs/...`, `refactor/...`,
   `chore/...`.

2. **Make your changes.** Keep the change focused — avoid mixing
   unrelated refactors into a feature PR.

3. **Validate locally before pushing:**
   ```bash
   make validate
   ```

4. **Write tests.** Bug fixes should include a regression test;
   new commands should include unit tests at minimum. Use the
   `pkg/testutil` helpers (`RunWithSimpleGraphQL`) to mock the API
   without hitting the network. See existing `cmd/*_test.go` files
   for examples.

   Integration tests are gated behind `NUDGEBEE_INTEGRATION=1` and
   require a real Nudgebee endpoint + API key.

5. **Commit** using the Conventional Commits format:
   ```
   <type>(<scope>): <short description>
   ```
   - **type**: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`,
     `test`, `chore`, `revert`, `ci`, `build`
   - **scope** (optional): the command or package touched (e.g.
     `events`, `nubi`, `client`, `format`).

   Examples:
   ```
   feat(events): add `events resolve` command
   fix(client): handle non-JSON error responses
   docs: clarify configuration file location
   ```

6. **Open a pull request** against `main`. CI will run lint + tests
   automatically.

## Pull Request Guidelines

- **Link an issue** when one exists (`Fixes #<n>` or `Refs #<n>`).
  For non-trivial changes, open an issue first so the approach can be
  discussed before code review.
- **Keep PRs small.** Aim for under ~400 lines of diff where
  possible. Split larger work into a series of PRs.
- **Fill in the PR template.** Describe the change and check the
  applicable "type of change" boxes.
- **All CI checks must pass** before review.
- **Be responsive to review feedback.** Push fixups as new commits;
  we'll squash on merge.

## Branch Model

- The `main` branch is the development branch that all
  contributions target.
- Open your pull request against `main`.
- Maintainers handle tagged releases.

## Security

If you believe you've found a security vulnerability, **do not open
a public issue**. Email **legal@nudgebee.com** with details. We will
acknowledge receipt within three business days and coordinate a fix
and disclosure timeline with you.

See [SECURITY.md](./SECURITY.md) for the full policy.

## Trademarks

The name "Nudgebee" and any associated logos are trademarks of
Nudgebee. The Apache 2.0 license does not grant trademark rights.

## Questions

Open a GitHub Discussion or issue, or email **legal@nudgebee.com**
for matters that are not appropriate for public discussion.
