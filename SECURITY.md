# Security Policy

## Supported Versions

`nbctl` is in pre-1.0 development. Security fixes are applied to the
latest minor release on the `main` branch. Older tagged releases are
not maintained.

## Reporting a Vulnerability

If you believe you've found a security vulnerability in `nbctl`,
**please do not open a public GitHub issue**.

Instead, email **legal@nudgebee.com** with:

- A description of the issue and its impact.
- Steps to reproduce, or a proof of concept.
- The version of `nbctl` (`nbctl version`) and your OS / architecture.
- Whether the issue is exploitable remotely or requires local access.

You can also use GitHub's private vulnerability reporting:
<https://github.com/nudgebee/nbctl/security/advisories/new>.

## Response Timeline

- **Acknowledgement**: within 3 business days of receipt.
- **Triage and severity classification**: within 7 business days.
- **Fix and coordinated disclosure**: timeline is agreed with the
  reporter based on severity and complexity. We aim for a fix within
  30 days for high-severity issues.

We will credit reporters in the release notes unless anonymity is
requested.

## Scope

In scope:

- Vulnerabilities in the `nbctl` binary itself (e.g. credential
  handling, command injection, path traversal, supply-chain).
- Vulnerabilities in `nbctl`'s build/release pipeline.

Out of scope (report to legal@nudgebee.com but not under this policy):

- Vulnerabilities in the Nudgebee server-side API that `nbctl` calls.
- Vulnerabilities in third-party dependencies (please open an upstream
  report; we will then pick up the fix via Dependabot).
