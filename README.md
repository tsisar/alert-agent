# Alert Agent

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go)](go.mod)
[![Status](https://img.shields.io/badge/status-beta-orange)]()

An autonomous alert investigation service for Grafana. Receives alerts via webhook,
investigates them using an LLM with Grafana MCP tools (Prometheus queries, Loki logs,
dashboard screenshots), and delivers structured reports to Telegram and Slack.

```
Grafana Alertmanager ──► Webhook ──► Dedup ──► Queue ──► Agent (LLM + MCP) ──► Telegram / Slack
                                                              │
                                                    Prometheus, Loki,
                                                    Dashboard screenshots
```

> **Status:** beta. Used in production, but the API and DB schema may still
> change. Pin to a tagged release if you adopt it.

## Architecture

### Data flow

```
                         ┌─────────────────────────────────────────────────┐
                         │                  Alert Agent                    │
                         │                                                 │
  Grafana Alertmanager   │   ┌─────────┐   ┌───────┐   ┌───────┐           │
  ─── POST /webhook ────►│   │ Handler │──►│ Dedup │──►│ Queue │           │
                         │   └─────────┘   └───────┘   └───┬───┘           │
                         │       │                         │               │
                         │       │ resolved/paused         │ firing        │
                         │       ▼                         ▼               │
                         │  ┌──────────┐          ┌──────────────┐         │
                         │  │ Notify() │          │    Run()     │         │
                         │  │ 1 LLM    │          │  Tool-use    │         │
                         │  │ call     │          │  loop        │◄──► MCP servers
                         │  └────┬─────┘          │  (max 20     │    (SSE)
                         │       │                │  iterations) │         │
                         │       │                └──────┬───────┘         │
                         │       │                       │                 │
                         │       ▼                       ▼                 │
                         │  ┌─────────────────────────────────┐            │
                         │  │           sendReport()          │            │
                         │  │  1. Summary message             │            │
                         │  │  2. Full report (.md document)  │            │
                         │  │  3. Panel screenshots (PNG)     │            │
                         │  └───────────┬─────────────────────┘            │
                         │              │                                  │
                         └──────────────┼──────────────────────────────────┘
                                        │
                              ┌─────────┴─────────┐
                              ▼                   ▼
                          Telegram             Slack
```

### Agent investigation loop

```
Run(ctx, payload, scenario)
│
├── 1. Load prompts from DB (system, summary)
├── 2. Filter MCP tools for scenario (allowlist)
├── 3. Build user message: alert JSON + scenario prompt + time budget
│
├── 4. Tool-use loop (max 20 iterations)
│      │
│      ├── Context cancelled? ──► timeoutFinish()
│      │
│      ├── Context budget approaching? ──► trimMessages()
│      │   (replace oldest tool results with "[trimmed]")
│      │
│      ├── ChatCompletion(system prompt, messages, tools)
│      │
│      ├── finish_reason == "stop"?
│      │   └── YES ──► extract report + generate summary ──► return Result
│      │
│      └── finish_reason == "tool_call"?
│          └── For each tool call (max 10 per iteration):
│              ├── CallTool(name, args) via MCP
│              ├── Collect images from result
│              └── Append tool result to messages
│
└── 5. Max iterations reached ──► timeoutFinish()
       └── Inject "Time is up" message
       └── Final LLM call without tools
       └── Return partial report
```

### Webhook handler decision tree

```
POST /webhook
│
├── Invalid JSON ──► 400 Bad Request
│
├── No matching scenario ──► 202 {"status":"no matching scenario"}
│
├── status == "resolved"
│   ├── Clear dedup entry
│   ├── Enqueue status notification job
│   └── 202 {"status":"resolved"}
│
├── state == "paused"
│   ├── Enqueue status notification job
│   └── 202 {"status":"paused"}
│
├── Dedup: within cooldown?
│   └── YES ──► 202 {"status":"suppressed"}
│
└── Enqueue investigation job
    └── 202 {"status":"accepted"}
```

## How it works

1. **Grafana fires an alert** and sends a webhook payload to `/webhook`.
2. **Handler matches** the alert to a scenario by `commonLabels` (first match wins, ordered by `order_index`).
3. **Resolved/paused alerts** get a short LLM-generated notification (one LLM call, no investigation).
4. **Firing alerts** pass through deduplication — if the same `groupKey` was processed within the cooldown period (default `48h`), the alert is suppressed.
5. **Accepted alerts are queued** and processed sequentially by a single worker to prevent interleaved messages.
6. **The agent investigates** — an LLM executes a tool-use loop: queries Prometheus, reads logs, captures dashboard screenshots, following the scenario's instructions.
7. **A report is delivered** to Telegram and/or Slack: summary message + full `.md` report + PNG screenshots.

The webhook always responds immediately with `202 Accepted`. Investigation runs asynchronously.

## Quick start

### Prerequisites

- Go 1.25+
- [MCP Grafana](https://github.com/grafana/mcp-grafana) server (SSE endpoint)
- LLM API key (OpenAI or any OpenAI-compatible provider)
- Telegram bot token and/or Slack bot token (optional but recommended)

### Setup

```bash
git clone https://github.com/tsisar/alert-agent.git
cd alert-agent

cp .env.example .env
# Edit .env:
#   LLM_API_KEY=sk-...
#   TELEGRAM_BOT_TOKEN=
#   SLACK_BOT_TOKEN=
#   ALERT_COOLDOWN=48h
```

Configure the MCP server endpoint in `.mcp.json`:

```json
{
  "mcpServers": {
    "grafana": {
      "type": "sse",
      "url": "http://mcp-grafana.example.com/sse"
    }
  }
}
```

### Run

```bash
make run
```

The server listens on `:8080` by default. The web UI is available at `/scenarios`
and `/prompts`.

### Smoke test

```bash
make test-webhook
```

## Scenarios

Scenarios define how the agent investigates each alert type. They are stored in the
database, managed through the web UI, and can be exported/imported as YAML.

### Fields

| Field               | Type              | Required | Description                                              |
|---------------------|-------------------|----------|----------------------------------------------------------|
| `name`              | string            | yes      | Unique scenario name                                     |
| `match`             | map[string]string | yes      | Labels for matching. Empty `{}` = catch-all              |
| `prompt`            | string            | yes      | Investigation instructions for the LLM                   |
| `tools`             | []string          | no       | Allowed MCP tools (namespaced `server__tool`). Empty = all |
| `channels.telegram` | string            | no       | Telegram chat ID (`"-"` to disable, empty = use default) |
| `channels.slack`    | string            | no       | Slack channel ID (`"-"` to disable, empty = use default) |
| `timeout`           | duration          | yes      | Investigation time limit (`2m`, `5m`, …)                 |
| `priority`          | string            | yes      | `normal`, `high`, or `critical`                          |
| `send_images`       | bool              | no       | Attach panel screenshots to the report                   |

### Matching

1. Scenarios are checked in `order_index` order (ascending).
2. All labels in `match` must match the alert's `commonLabels` (AND logic).
3. First match wins.
4. Empty match = catch-all (place last by giving it a high `order_index`).

### Channel resolution

| Value in scenario | Behavior                      |
|-------------------|-------------------------------|
| (not set)         | Falls back to default channel |
| `"-"`             | Channel explicitly disabled   |
| `"C0A699M79EF"`   | Uses the specified channel    |

### Philosophy

A scenario provides **recommendations**, not a rigid script. The LLM can skip steps,
investigate things not mentioned, or dig deeper where it sees fit. Think of it as
context for a new on-call engineer: "when you see this alert, check X and Y".

## Web UI

The built-in web UI provides:

- **Scenario management** — create, edit, delete, reorder scenarios.
- **Prompt editor** — customize system, summary, resolved, and paused prompts.
- **YAML export/import** — backup and restore scenario configurations.
- **Grafana import** — pull alert rules from Grafana to auto-fill match labels.
- **MCP tool picker** — browse discovered tools with descriptions.

Tech stack: Go templates ([Templ](https://templ.guide/)), [HTMX](https://htmx.org/),
and a custom CSS layer based on [Material Design 3](https://m3.material.io/) tokens.
No Node.js build step — CSS, JS, and templates are embedded in the Go binary.

## Deduplication and queue

### Dedup

Prevents repeated investigations for the same firing alert. The dedup key is a SHA256 hash of the
Grafana `groupKey`. Two backends:

| Backend   | Config             | Behavior                                  |
|-----------|--------------------|-------------------------------------------|
| In-memory | (default)          | `sync.Mutex` + map, lost on restart       |
| Redis     | `REDIS_URL` is set | `SET NX` with TTL, shared across replicas |

Resolved alerts clear their dedup entry, allowing re-investigation if the alert fires again.

### Queue

Alerts are processed sequentially by a single worker to prevent interleaved messages in chats.

| Backend   | Config             | Details                                                              |
|-----------|--------------------|----------------------------------------------------------------------|
| In-memory | (default)          | Buffered channel (100 jobs), dropped if full                         |
| Redis     | `REDIS_URL` is set | Redis Streams consumer group with `XAUTOCLAIM` retries and a DLQ     |

When using Redis, jobs that fail are re-delivered up to **3 times** before being moved
to `alert-agent:dlq`. A fallback failure message is sent so alerts are never silently
dropped.

## Notification flow

Each completed investigation produces three messages per channel:

1. **Summary** — 1–2 sentence message (max 280 chars).
2. **Full report** — Markdown document attached as a file.
3. **Screenshots** — Panel images as photos (if `send_images` is enabled).

Resolved and paused alerts produce only a single short message.

## Configuration

Main config: `configs/config.yaml`. All values can be overridden via environment
variables. A `.env` file is loaded automatically.

```yaml
http:
  addr: ":8080"

mcp:
  config_path: ".mcp.json"

database:
  driver: "sqlite"              # sqlite, postgres, mysql
  dsn: "alert-agent.db"

llm:
  provider: "openai"
  model: "gpt-4o"
  max_tokens: 32768             # tool-use loop budget
  summary_max_tokens: 4096      # final summary call
  context_limit: 0              # model context window (0 = no trimming)

telegram:
  bot_token: ""
  default_channel: ""

slack:
  bot_token: ""
  default_channel: ""

redis:
  url: ""                       # empty = in-memory; e.g. "redis://localhost:6379"

alert_cooldown: "48h"
```

Common environment variables: `LLM_API_KEY`, `LLM_MODEL`, `LLM_BASE_URL`,
`TELEGRAM_BOT_TOKEN`, `SLACK_BOT_TOKEN`, `REDIS_URL`, `ALERT_COOLDOWN`,
`DATABASE_DRIVER`, `DATABASE_DSN`, `MCP_CONFIG_PATH`, `LOG_LEVEL`.

See [docs/configuration.md](docs/configuration.md) for the full reference.

## Deployment

### Docker

```bash
make image          # tags as $(REGISTRY)/$(APP):$(VERSION)

# Or build with a tag of your choice
docker build -t alert-agent:dev .

docker run -d \
  --name alert-agent \
  -p 8080:8080 \
  -e LLM_API_KEY=sk-... \
  -e TELEGRAM_BOT_TOKEN=... \
  -e SLACK_BOT_TOKEN=xoxb-... \
  -v "$(pwd)/.mcp.json:/.mcp.json" \
  -v alert-agent-data:/data \
  -e DATABASE_DSN=/data/alert-agent.db \
  alert-agent:dev
```

### Docker Compose

A Compose stack is bundled at [`deployment/compose/`](deployment/compose/)
(see its [README](deployment/compose/README.md)):

```bash
cd deployment/compose
docker compose up -d
```

### Kubernetes (Helm)

A Helm chart is bundled at [`deployment/helm/chart/`](deployment/helm/chart/)
(see its [README](deployment/helm/chart/README.md) for the full value set):

```bash
helm install alert-agent ./deployment/helm/chart \
  --set secret.LLM_API_KEY=sk-... \
  --set secret.TELEGRAM_BOT_TOKEN=... \
  --set secret.SLACK_BOT_TOKEN=xoxb-...
```

See [docs/deployment.md](docs/deployment.md) for full options.

## HTTP endpoints

### Webhook

| Method | Path       | Description                    |
|--------|------------|--------------------------------|
| `POST` | `/webhook` | Receive Grafana alerts         |
| `GET`  | `/healthz` | Health check, returns `200 ok` |

### Web UI

| Method   | Path                     | Description                  |
|----------|--------------------------|------------------------------|
| `GET`    | `/scenarios`             | Scenario list                |
| `GET`    | `/scenarios/new`         | Create scenario form         |
| `POST`   | `/scenarios`             | Create scenario              |
| `GET`    | `/scenarios/{id}/edit`   | Edit scenario form           |
| `PUT`    | `/scenarios/{id}`        | Update scenario              |
| `DELETE` | `/scenarios/{id}`        | Delete scenario              |
| `GET`    | `/scenarios/export.yaml` | Export all scenarios as YAML |
| `POST`   | `/scenarios/import`      | Import scenarios from YAML   |
| `GET`    | `/prompts`               | Prompt editor                |
| `PUT`    | `/prompts/{key}`         | Update prompt template       |

### API

| Method | Path                  | Description                       |
|--------|-----------------------|-----------------------------------|
| `GET`  | `/api/tools`          | List discovered MCP tools (JSON)  |
| `GET`  | `/api/grafana/alerts` | Fetch alert rules from Grafana    |

## Project structure

```
cmd/alert-agent/             Entrypoint, Cobra CLI
internal/
  agent/                     Agentic loop — LLM + tools orchestration
  config/                    Config loading (Viper + env)
  llm/                       LLM provider interfaces
    openai/                  OpenAI-compatible API implementation
  mcp/                       MCP client and server manager (SSE transport)
  model/                     Grafana webhook payload structures
  notify/                    Notifier interface
    telegram/                Telegram bot (MarkdownV2, photos, documents)
    slack/                   Slack bot (mrkdwn, file uploads)
  scenario/                  Domain types and label matching
  server/                    HTTP server with graceful shutdown
  storage/                   GORM database layer (scenarios, prompts)
    seed/                    Default seed data (embedded YAML)
  web/                       Web UI (templ + HTMX + Material 3 CSS)
    templates/               templ templates
    static/                  CSS, JS, favicon (embedded in binary)
  webhook/                   Webhook handler, dedup, queue (in-mem + Redis Streams)
  version/                   Build version constants
configs/                     Configuration files
docs/                        Documentation
```

## Makefile

| Target                | Description                             |
|-----------------------|-----------------------------------------|
| `make run`            | Run locally                             |
| `make test`           | Run tests with coverage                 |
| `make lint`           | Run golangci-lint                       |
| `make build`          | Production binary (optimized, stripped) |
| `make dev`            | Dev binary (with debug symbols)         |
| `make generate`       | Generate templ templates                |
| `make image`          | Build Docker image                      |
| `make push`           | Push image to the configured registry   |
| `make build-and-push` | clean → test → build → image → push     |
| `make release`        | Tag a release from the `release` branch |
| `make test-webhook`   | Send a test webhook to localhost:8080   |
| `make clean`          | Remove build artifacts                  |

## Documentation

- [Architecture](docs/architecture.md) — system design, components, data flow
- [Agent loop](docs/agent-loop.md) — how the LLM investigation loop works
- [Configuration](docs/configuration.md) — config options and environment variables
- [MCP integration](docs/mcp-integration.md) — how MCP servers are wired up
- [Dependencies](docs/dependencies.md) — required and optional infrastructure
- [Deployment](docs/deployment.md) — local, Docker, and Kubernetes deployment
- [Release process](docs/release-process.md) — tagging and CI release flow

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for branch
naming, commit conventions, and local development setup. Security issues should
be reported privately — see [SECURITY.md](SECURITY.md).

## License

Released under the [MIT License](LICENSE).