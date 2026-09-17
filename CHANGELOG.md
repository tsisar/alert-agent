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

### Changed

- Web UI rebuilt as a signal panel: scenarios read as ordered paths from matched
  labels through investigation to delivery channel, with a keyboard filter, a label
  pair editor, an MCP tool picker that flags tools no connected server offers, inline
  validation errors, an unsaved-changes guard, per-stage prompt dirty state, and an
  inline delete confirmation in place of the browser dialog.
- Fonts are self-hosted and embedded in the binary; the UI no longer requests Google
  Fonts, so it renders correctly in an air-gapped cluster.
- `LLM_MODEL` now defaults to empty, letting each provider pick its own default
  (`gpt-4o` for OpenAI, `claude-opus-4-8` for Anthropic).
- Web UI theme palette switched from purple to blue.
- Container image is now built multi-arch for `linux/amd64` and `linux/arm64`.

### Fixed

- Saving a scenario with an invalid field no longer fails silently: htmx does not
  swap error responses, so a rejected save previously looked like nothing happened.
  The editor is now re-rendered with the submitted values and a stated reason.
- Investigation timeout no longer drops images collected before the deadline.
- Scenarios are hard-deleted so a name can be reused after deletion.
- Scenario YAML import/export preserves the `order_index` ordering.
- Scenario prompt preview is truncated to a single clean line in the web UI.

### Performance

- Enabled Anthropic prompt caching: the tools + system prompt and the growing
  conversation prefix are cached (5m TTL), so each tool-use loop iteration
  re-reads prior turns from cache instead of reprocessing them at full price.
