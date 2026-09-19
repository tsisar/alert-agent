# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

Primary: SRE / DevOps engineers who own the alerting stack. They work at a desktop,
usually with Grafana open in a neighbouring tab, and come to this UI to configure how
the agent behaves — not during the panic of an incident, but around it. Mobile access
is rare and not a design driver (confirmed by the user, 2026-09-18).

## Product Purpose

Alert Agent receives Grafana/Alertmanager webhooks, investigates each firing alert with
an LLM that can call Grafana MCP tools, and posts a written investigation to Telegram or
Slack. The web UI is the control room for that behaviour: it decides which alerts are
matched by which scenario, what the LLM is told to do, which tools it may use, where the
report goes, and how long it may take.

## Positioning

The agent turns an alert into an investigated report rather than a forwarded notification.
Its configuration surface is therefore not a notification-routing table: every scenario is
a small standing instruction for an autonomous investigator — match, prompt, tool budget,
time budget, destination.

## Operating Context

- Self-hosted Go binary (Docker / Helm), usually on the team's own cluster next to Grafana.
- Configuration lives in a database (sqlite/postgres/mysql) and is also importable and
  exportable as YAML, so the UI is one of two equal editing paths and must not lose
  fidelity against the file.
- Scenarios are evaluated in `order_index` order; the first match wins, and a scenario with
  no match labels is a catch-all. Ordering is therefore semantic, not cosmetic.
- Tool names come live from the connected MCP servers and are namespaced `server__tool`.
  A scenario can reference a tool that no longer exists after an MCP server changes.
- Alert volume is bursty; the agent processes one alert at a time per instance.

## Capabilities and Constraints

- Screens today: scenario list, scenario create/edit/copy form, prompt templates. Plus
  two JSON endpoints the UI consumes: `GET /api/tools`, `GET /api/grafana/alerts`.
- Scenario fields: name, order index, match labels (key=value map), prompt, allowed tools,
  Telegram channel, Slack channel, timeout, priority (normal/high/critical), send images.
- Prompts: four global templates — system, summary, resolved, paused.
- Stack is fixed by the codebase: Go + `templ` server-rendered HTML, htmx for interaction,
  hand-written CSS in `internal/web/static/app.css`, all embedded in the binary.
- There is no run/investigation history in storage, so the UI cannot show past
  investigations without new backend work. Do not fabricate that data.
- The app ships as one binary; anything the UI needs at runtime should be embedded rather
  than fetched from a third-party CDN where that is practical.

## Brand Commitments

Name "Alert Agent". Light and dark themes are both required, remembered per browser
(confirmed by the user, 2026-09-18). No other visual constraint — palette, typography and
component language are free to be replaced.

## Evidence on Hand

- Real seeded content: the default catch-all scenario and the four prompt templates in
  `internal/storage/seed/`.
- Live tool names and Grafana alert rules via MCP when a server is configured; empty when
  it is not. Both states must be designed for.
- No usage metrics, no customer names, no benchmarks. Nothing of the sort may be invented.

## Product Principles

1. Configuration is consequential: every screen must make the effect of a change legible
   before it is saved, because a wrong match rule silently misroutes real incidents.
2. Order and matching are the product's core logic, not form fields — show them as logic.
3. The UI and the YAML are the same truth; neither may lose information the other keeps.
4. Live data (tools, alert rules) can be absent or stale; every surface that uses it needs
   an honest empty and error state.
5. The audience is expert. Prefer density, keyboard reach and direct manipulation over
   guided hand-holding.

## Accessibility & Inclusion

No standard was mandated. Baseline expectation: full keyboard operation, visible focus,
and WCAG AA contrast in both themes.
