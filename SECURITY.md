# Security Policy

## Supported versions

Alert Agent is in beta. Only the latest `1.x` release receives security
updates.

| Version | Supported          |
|---------|--------------------|
| 1.x     | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a vulnerability

**Please do not file public GitHub issues for security problems.** A public
issue would expose users to the problem before a fix is available.

Instead, report privately through GitHub's
[private vulnerability reporting](https://docs.github.com/en/code-security/security-advisories/guidance-on-reporting-and-writing/privately-reporting-a-security-vulnerability)
feature: go to the repository's **Security** tab → **Report a vulnerability**.

When reporting, please include:

- A description of the vulnerability and its impact.
- The version (or commit SHA) where you found it.
- Steps to reproduce, or a proof-of-concept.
- Any suggested mitigation or fix, if you have one.

You can expect:

- An acknowledgement within **5 business days**.
- A status update within **14 days**, with either a planned fix and timeline
  or an explanation of why we believe it isn't actually a vulnerability.
- A coordinated disclosure once a fix is available, including a CVE if the
  issue warrants one.

## Scope

In scope:

- The Alert Agent service itself (`cmd/alert-agent`, all packages under
  `internal/`).
- The default scenario and prompt seed data.
- The Docker image published from this repository.

Out of scope:

- Vulnerabilities in third-party MCP servers (e.g., `mcp-grafana`) — report
  those upstream.
- Vulnerabilities in your own LLM provider, Grafana, Telegram, or Slack.
- Configuration mistakes by the operator (leaked API keys in env vars,
  exposed `/webhook` endpoint, etc.).

## What we consider a vulnerability

- Remote code execution, SQL injection, command injection in the agent or
  Web UI.
- Authentication / authorization bypass on the Web UI or API.
- Webhook handler vulnerabilities (e.g., DoS via unbounded payload, parser
  crashes, dedup bypass).
- Sensitive data leakage in logs, reports, or notifications.
- Supply-chain risks introduced by direct dependencies.
