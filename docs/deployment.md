# Deployment

## Local run

### Prerequisites

- Go 1.25+ (see `go.mod`)
- An LLM API key (OpenAI or any OpenAI-compatible endpoint)
- Access to at least one MCP server over SSE (e.g. `mcp-grafana`)
- Optional: Telegram bot token, Slack bot token (`chat:write`, `files:write`)
- Optional: Redis (for cross-restart dedup and queue durability)

### Environment

Copy the template and fill in real values:

```bash
cp .env.example .env
$EDITOR .env
```

Keys you almost always need:

```env
LLM_API_KEY=sk-...
LLM_BASE_URL=                          # blank = api.openai.com
LLM_MODEL=gpt-4o
TELEGRAM_BOT_TOKEN=
SLACK_BOT_TOKEN=
ALERT_COOLDOWN=48h
MCP_CONFIG_PATH=.mcp.json
```

See [`configuration.md`](configuration.md) for the full env-var surface.

### MCP servers config

`.mcp.json` (path overridable via `MCP_CONFIG_PATH`) lists MCP servers to discover
tools from at startup:

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

Only the `sse` transport is supported today.

### Running

```bash
# Via Make (regenerates templ code first)
make run

# Or directly
go run ./cmd/alert-agent serve

# With a custom config path
go run ./cmd/alert-agent serve --config ./my-config.yaml
```

The server listens on `:8080` by default. `make run` depends on `make generate`,
which regenerates the `*_templ.go` files from `.templ` sources (`make install-templ`
installs the tool if missing).

### Smoke-test webhook

```bash
make test-webhook
```

Sends a synthetic Grafana alert payload (inlined in the Makefile) to
`http://localhost:8080/webhook`.

## Makefile

| Target                | Description                                               |
|-----------------------|-----------------------------------------------------------|
| `make help`           | List all targets                                          |
| `make generate`       | Generate templ code (`internal/web/templates/*_templ.go`) |
| `make run`            | Run locally (regenerates templ first)                     |
| `make test`           | Run tests with coverage                                   |
| `make build`          | Production binary (stripped, with version ldflags)        |
| `make dev`            | Dev binary (with debug symbols)                           |
| `make image`          | Build Docker image                                        |
| `make push`           | Push image to the configured registry                     |
| `make print-image`    | Print the image tag that would be built                   |
| `make build-and-push` | clean → test → build → image → push                       |
| `make release`        | Tag a release from the `release` branch                   |
| `make format`         | gofmt                                                     |
| `make lint`           | golangci-lint                                             |
| `make clean`          | Remove artifacts                                          |
| `make test-webhook`   | Send a sample webhook to `localhost:8080`                 |

### Build configuration

```makefile
TARGETOS     ?= linux
TARGETARCH   ?= amd64
CGO_ENABLED  ?= 0
APP          ?= alert-agent
REGISTRY     ?= ghcr.io/tsisar          # override for your registry
```

The image version is derived automatically: `<git-describe-tag>-<short-sha>`
(e.g. `v0.1.0-a1b2c3d`). Override with `make image VERSION=v0.1.0`.

## Docker

### Image

Multi-stage build:

1. **Builder** — `golang:1.25-alpine`, builds with `-ldflags="-s -w"` (stripped)
   plus version/commit/date ldflags.
2. **Runtime** — `alpine:3.21` + `ca-certificates` + `tzdata`, runs
   `alert-agent serve`.

```bash
# Build with the project Makefile (tags as $(REGISTRY)/$(APP):$(VERSION))
make image

# Or build manually with a tag of your choice
docker build -t alert-agent:dev .

# Run
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

The default working directory in the image is `/`. The binary reads
`MCP_CONFIG_PATH` (default `.mcp.json`) relative to that. Mount your `.mcp.json`
at a path that matches the env var.

## Docker Compose

For single-host deployments, a Compose stack is bundled at
[`deployment/compose/`](../deployment/compose/) — see its
[README](../deployment/compose/README.md) for details.

```bash
cd deployment/compose
docker compose up -d
```

The stack reads `../../.env` and `../../.mcp.json`, persists SQLite in a named
volume, and includes a commented-out Redis service block that you can
uncomment to enable cross-restart dedup, queue durability, and the DLQ
failure path.

## Kubernetes (Helm)

A Helm chart is bundled at [`deployment/helm/chart/`](../deployment/helm/chart/).
See its [README](../deployment/helm/chart/README.md) for the full value set.

```bash
helm install alert-agent ./deployment/helm/chart \
  --set secret.LLM_API_KEY=sk-... \
  --set secret.TELEGRAM_BOT_TOKEN=... \
  --set secret.SLACK_BOT_TOKEN=xoxb-... \
  --set 'mcp.servers.grafana.url=http://mcp-grafana.monitoring:8000/sse'
```

The chart deploys a single-replica Deployment (with `strategy: Recreate` to
play nicely with the `ReadWriteOnce` PVC for SQLite), a ConfigMap rendered
from `mcp.servers`, an optional Secret, a ClusterIP Service on port 8080,
and an optional Ingress. The pod runs as non-root with a read-only root
filesystem and dropped capabilities.

For multi-replica HA, switch `env.DATABASE_DRIVER` to `postgres` or `mysql`
and provide an external DSN — the SQLite PVC can then be removed.

### Grafana Alertmanager

Configure a webhook contact point in Grafana Alerting:

```
URL: http://alert-agent.<namespace>.svc.cluster.local:8080/webhook
```

Enable `send_resolved: true` to get resolved notifications.

## HTTP endpoints

| Method | Path                  | Description                         |
|--------|-----------------------|-------------------------------------|
| `GET`  | `/healthz`            | Health check, returns `200 ok`      |
| `POST` | `/webhook`            | Receive Grafana alerts              |
| `GET`  | `/scenarios`          | Web UI — manage scenarios           |
| `GET`  | `/prompts`            | Web UI — manage prompts             |
| `GET`  | `/api/tools`          | JSON — list of discovered MCP tools |
| `GET`  | `/api/grafana/alerts` | JSON — alerts from Grafana          |

See [`architecture.md`](architecture.md) for the full route map.

### POST /webhook

**Request:** Grafana Alertmanager webhook payload (JSON, max 1 MB).

**Response:**

- `202 Accepted` + `{"status":"accepted","scenario":"…"}` — investigation started
- `202 Accepted` + `{"status":"suppressed","scenario":"…"}` — duplicate, suppressed within cooldown
- `202 Accepted` + `{"status":"resolved","scenario":"…"}` — short resolved notification sent, dedup cleared
- `202 Accepted` + `{"status":"paused","scenario":"…"}` — short paused notification sent
- `202 Accepted` + `{"status":"no matching scenario"}` — no scenario matched
- `400 Bad Request` — invalid JSON or size limit exceeded

The investigation runs asynchronously; the webhook response does not wait for
completion. Resolved and paused alerts use a short LLM call instead of the full
loop.

## Not yet implemented

- **Prometheus metrics** — no `/metrics` endpoint yet.
- **Interactive follow-up** — replying to investigations in Telegram/Slack.
- **Investigation history UI** — results are sent to channels but not retained
  in the database for post-mortem browsing.