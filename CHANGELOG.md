# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning 2.0.0](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Anthropic (Claude) LLM provider built on the official Messages API, selectable
  via `LLM_PROVIDER=anthropic`. Streams responses and accumulates them to avoid
  HTTP timeouts on large `max_tokens` values.
- Reasoning/thinking effort for both providers via `LLM_REASONING_EFFORT`
  (`low`, `medium`, `high`). OpenAI sends `reasoning_effort` (reasoning models
  only); Anthropic enables adaptive thinking with reasoning blocks preserved
  across the tool-use loop.
- Copy button in the web UI to duplicate an existing scenario.

- Match probe: `POST /api/match-probe` answers which scenario would catch a given
  set of alert labels, using the same matcher the webhook path uses. Surfaced in
  the web UI as *Test an alert*.

- Authentication for MCP servers (e.g. Grafana behind an auth proxy): an
  optional `headers` map in `.mcp.json` is sent with every request of the
  connection and re-applied on reconnect. Header values may reference
  environment variables as `${VAR}`, so the token can come from `.env` or a
  Kubernetes Secret (`secret.MCP_AUTH_TOKEN` in the Helm chart) instead of the
  config file; a referenced but unset variable fails at startup.
- `http` (Streamable HTTP) MCP transport alongside `sse`.
- Reorder scenarios from the list with up/down arrows (`POST /scenarios/{id}/move`);
  the whole list is renumbered so evaluation order is explicit.

### Changed

- Web UI redesigned as a conventional tool UI so every action is easy to find:
  scenarios in a table with always-visible Edit/Duplicate/Delete, fallback
  (catch-all) scenarios grouped separately with a warning when none or several
  exist, a "Test an alert" card that highlights the matching scenario, a keyboard
  filter, a label pair editor, an MCP tool picker that flags tools no connected
  server offers, inline validation errors, a sticky Save bar with unsaved-changes
  state, prompt templates as tabs with per-tab unsaved state, and an
  inline delete confirmation in place of the browser dialog.
- The web UI uses the system font stack; no web fonts are loaded or embedded, so it
  renders correctly in an air-gapped cluster.
- `LLM_MODEL` now defaults to empty, letting each provider pick its own default
  (`gpt-4o` for OpenAI, `claude-opus-4-8` for Anthropic).
- Web UI theme palette switched from purple to blue.
- Container image is now built multi-arch for `linux/amd64` and `linux/arm64`.
- The favicon and the top-bar mark are one icon (a bell with a spark), served from
  a single `static/favicon.svg`.
- The prompts page uses the same floating save bar as the scenario editor, one per
  template tab.

### Fixed

- Saving a scenario with an invalid field no longer fails silently: htmx does not
  swap error responses, so a rejected save previously looked like nothing happened.
  The editor is now re-rendered with the submitted values and a stated reason.
- Investigation timeout no longer drops images collected before the deadline.
- Scenarios are hard-deleted so a name can be reused after deletion.
- Scenario YAML import/export preserves the `order_index` ordering.
- Scenario prompt preview is truncated to a single clean line in the web UI.
- The MCP tool picker and Grafana rule import dialogs scroll their list; before, a
  long list was cut off and the mouse wheel scrolled the page behind the dialog.
- The scenario editor uses the same width as the other pages, and the layout no
  longer shifts sideways when moving between a short page and a long one.
- After saving a prompt template, focus returns to the text field.

### Performance

- Enabled Anthropic prompt caching: the tools + system prompt and the growing
  conversation prefix are cached (5m TTL), so each tool-use loop iteration
  re-reads prior turns from cache instead of reprocessing them at full price.
