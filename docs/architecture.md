# Architecture

## Overview

Alert Agent is a Go service that receives alerts from Grafana Alertmanager via webhook,
autonomously investigates them using an LLM (OpenAI or Anthropic) and Grafana MCP tools,
and sends a structured report with screenshots to Telegram and Slack.

Scenarios (matching rules + LLM prompts), notification prompts, and dedup/queue state are
persisted — scenarios and prompts in a relational database (SQLite/Postgres/MySQL),
deduplication and the work queue optionally in Redis.

## Components

```
Grafana Alertmanager
        │
        │ POST /webhook (JSON)
        ▼
┌──────────────────────────────────┐
│         Webhook Handler          │
│  Payload parsing, scenario       │
│  matching, dedup, 202 response   │
└──────┬─────────────────┬─────────┘
       │ resolved/paused │ firing
       │  (short notify) │
       │                 ▼
       │      ┌────────────────────────────┐
       │      │ Queue                      │
       │      │  - Redis Streams +         │
       │      │    consumer group "workers"│
       │      │  - or in-memory fallback   │
       │      │  - DLQ after 3 deliveries  │
       │      └──────────┬─────────────────┘
       │                 │ sequential worker
       ▼                 ▼
┌──────────────────────────────────┐
│             Agent                │
│  1. Build prompts (system +      │
│     scenario + alert + budget)   │
│  2. Tool-use loop (max 20 iter)  │◄────── LLM Provider (OpenAI / Anthropic)
│  3. Trim context as it grows     │
│  4. Collect images from tools    │
│  5. Summarize → Result           │──────► MCP Manager
└──────────────┬───────────────────┘            │
               │                                ▼
               │                      ┌──────────────────┐
               │                      │   MCP servers    │
               │                      │   (SSE)          │
               │                      │                  │
               │                      │  grafana, …      │
               │                      │  (auto-discover  │
               │                      │   namespaced     │
               │                      │   tools)         │
               │                      └──────────────────┘
               ▼
┌──────────────────────────────────┐
│           Notifiers              │
│  Telegram (MarkdownV2, files)    │
│  Slack    (mrkdwn, file uploads) │
└──────────────────────────────────┘

       ┌──────────────────────────┐
       │ Web UI (/scenarios,      │
       │ /prompts) — manage       │
       │ rules and prompts via    │
       │ HTMX, persisted in DB    │
       └──────────────────────────┘
```

## Data flow

### Firing alerts

1. **Grafana Alertmanager** sends a POST request with a JSON payload to `/webhook`.
2. **Webhook Handler** parses the payload and finds the first matching scenario by alert
   labels (scenarios are ordered by `order_index`; the first match wins; an empty
   `match` is the catch-all).
3. **Deduplicator** checks whether this firing alert was already processed within the
   cooldown period (default `48h`). If so, the alert is suppressed. Backend is either
   in-memory or Redis (`SET NX EX`).
4. The handler immediately responds with `202 Accepted` and enqueues the job. The queue
   backend is selected at startup:
   - **Redis Streams** (`alert-agent:stream`, consumer group `workers`) when `redis.url`
     is configured. `XAUTOCLAIM` re-delivers jobs idle for more than 10 minutes.
   - **In-memory** sequential queue otherwise.
5. A **single worker** consumes the queue and runs the **Agent**:
   - Builds the prompt from the system prompt + scenario prompt + alert payload + the
     remaining time budget.
   - Enters the **tool-use loop** (max 20 iterations, up to 10 tool calls per iteration).
   - On each iteration, if `llm.context_limit` is set, older tool results are replaced
     with a `[trimmed]` placeholder to fit the budget.
   - The LLM may call any tool exposed by the configured MCP servers (filtered to the
     scenario allowlist when set).
   - Stops on `stop` finish reason, scenario timeout, or max iterations. On timeout/
     max-iter, the LLM is asked to produce a final report from what it has gathered.
   - A separate LLM call produces the short notification summary.
6. **Notifiers** deliver the summary message, the full report as a `.md` document, and
   any screenshots to the channels resolved from the scenario (with a default fallback;
   `"-"` disables a channel, `""` falls back to the configured default).
7. **Retries and DLQ** — if the worker fails (panic, transient errors), the Redis stream
   re-delivers the job up to `maxDeliveries = 3`. After that the job is moved to
   `alert-agent:dlq` and **`notifyFailure`** sends a short fallback message so the alert
   is never silently dropped.

### Resolved / paused alerts

These bypass the queue and the full investigation loop. A short LLM call generates a
notification from the resolved/paused prompt, and the message is sent immediately.
Resolved alerts also clear the dedup entry for that group key so the next firing of the
same alert starts a fresh cooldown.

## Package structure

```
cmd/alert-agent/
  main.go
  cmd/
    root.go                 Cobra root, config loading
    serve.go                serve subcommand — wires everything up
    version.go              version subcommand

internal/
  config/                   Viper config loading (file + env, defaults)
  model/                    Grafana webhook payload structures
  scenario/                 Domain types and label matching
  storage/                  GORM database layer
    db.go                   Driver wiring (sqlite/postgres/mysql)
    models.go               Scenario, Prompt, DedupEntry
    scenario_repo.go        CRUD for scenarios
    prompt_repo.go          CRUD for prompts with cache + version
    seed/                   Embedded YAML for default scenarios and prompts
    seed.go                 Seeds the DB on first run
  webhook/                  HTTP handler for /webhook
    handler.go              Parses, matches, dedups, enqueues, notifyFailure
    dedup.go                In-memory dedup store
    queue.go                In-memory sequential queue
    redis.go                Redis dedup + Streams queue + DLQ + XAUTOCLAIM
  agent/                    Agentic loop — LLM + tools orchestration
    agent.go                Loop, context trimming, summarize, Notify path
    tools.go                FilteredExecutor (scenario allowlist)
  llm/                      LLM Provider interfaces
    openai/                 OpenAI-compatible implementation
    anthropic/              Anthropic (Claude) Messages API implementation
  mcp/                      MCP client and server manager
    client.go               SSE client per server, namespaced tool names
    config.go               .mcp.json loading
  notify/                   Notifier interface
    telegram/               MarkdownV2, photos, documents
    slack/                  mrkdwn, file uploads via Web API
  server/                   HTTP server with graceful shutdown
  web/                      Web UI for managing scenarios and prompts
    handler.go              Route registration (/scenarios, /prompts, /api/*)
    templates/              templ templates (.templ + generated .go)
    static/                 Embedded CSS, JS, favicon
  version/                  Build version constants

configs/
  config.yaml               Defaults; env vars override
```

## Key interfaces

### `llm.Provider`

```go
type Provider interface {
    ChatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
}
```

One call = one request to the LLM. The tool-use loop is orchestrated externally
(in the Agent), not inside the provider. Two providers implement this interface —
`openai` and `anthropic` — and each maps the agent's provider-agnostic messages,
tools, and thinking blocks onto its own API. Adding another provider means
implementing this single method.

### `agent.ToolExecutor`

```go
type ToolExecutor interface {
    Tools() []llm.Tool
    CallTool(ctx context.Context, name string, arguments json.RawMessage) (*llm.ToolResult, error)
}
```

An abstraction over MCP — the Agent does not depend on MCP directly. `ToolExecutor`
returns the list of available tools and executes calls. The MCP Manager implements
this interface; tool names are namespaced as `<server>__<tool>` to prevent collisions
across servers. `FilteredExecutor` wraps the manager to apply the scenario's tool
allowlist.

### `notify.Notifier`

```go
type Notifier interface {
    SendMessage(ctx context.Context, chatID string, text string) error
    SendPhoto(ctx context.Context, chatID string, photo []byte, caption string) error
    SendDocument(ctx context.Context, chatID string, doc []byte, filename string, caption string) error
}
```

A delivery abstraction — allows adding new channels without modifying the webhook
handler. Currently implemented: Telegram and Slack.

### `storage.ScenarioRepository` / `storage.PromptRepository`

```go
type ScenarioRepository interface {
    List(ctx context.Context) ([]Scenario, error)
    Get(ctx context.Context, id uint) (*Scenario, error)
    Create(ctx context.Context, s *Scenario) error
    Update(ctx context.Context, s *Scenario) error
    Delete(ctx context.Context, id uint) error
}

type PromptRepository interface {
    GetAll(ctx context.Context) (*PromptSet, error)
    Set(ctx context.Context, key, value string) error
}
```

Scenarios and prompts live in the database. Both repositories are accessed by the
webhook handler, the Agent, and the Web UI. The prompt repository caches the
loaded set and invalidates on writes.

## Design principles

- **Async investigation** — the webhook responds in <100 ms; investigation runs in
  the background within the scenario's time budget.
- **Sequential processing** — a single worker drains the queue to prevent
  interleaved messages in notification channels.
- **At-least-once delivery with DLQ** — Redis Streams + `XAUTOCLAIM` re-deliver
  stuck jobs; after `maxDeliveries=3` the job lands in the DLQ and a fallback
  failure notification is sent.
- **Deduplication** — repeated firings within the cooldown are suppressed; resolved
  alerts clear the entry.
- **Scenarios guide, LLM decides** — the scenario provides hints and an allowlist
  of tools, but the LLM is free to call any allowed tool.
- **DB-backed configuration** — scenarios and prompts are editable at runtime via
  the Web UI; no restart needed.
- **Graceful degradation** — on timeout or max iterations the Agent asks the LLM
  to write the report from what it gathered.
- **Channel disable convention** — in a scenario, `channel = "-"` explicitly
  disables a channel, `""` falls back to the configured default.

## See also

- [`agent-loop.md`](agent-loop.md) — the agent loop in detail.
- [`mcp-integration.md`](mcp-integration.md) — how MCP servers are wired up.
- [`configuration.md`](configuration.md) — config keys and env vars.
- [`deployment.md`](deployment.md) — how to deploy.