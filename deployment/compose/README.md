# alert-agent · Docker Compose

A minimal Compose setup for running Alert Agent on a single host. Suitable
for staging, evaluation, or small production deployments.

## Quick start

```bash
# 1. From the repo root, prepare the env and MCP config:
cp .env.example .env
$EDITOR .env                              # at least LLM_API_KEY

# 2. Edit .mcp.json to point at your MCP server(s):
$EDITOR .mcp.json

# 3. Start the stack:
cd deployment/compose
docker compose up -d

# 4. Tail logs:
docker compose logs -f alert-agent
```

The service listens on `:8080` (webhook + Web UI). Test it:

```bash
curl http://localhost:8080/healthz
```

## What it deploys

- **alert-agent** — the agent itself, listening on port 8080.
- A **named volume** (`alert-agent-data`) holding the SQLite database at
  `/data/alert-agent.db`.
- A bind-mount of `../../.mcp.json` into `/etc/alert-agent/mcp.json` (read-only).

By default the image is pulled from `ghcr.io/tsisar/alert-agent:latest`.
To build from source instead, uncomment the `build:` block in `compose.yaml`
and run `docker compose build` before `up`.

## Enabling Redis

For cross-restart dedup, queue durability, and the DLQ failure path, switch
to Redis. In `compose.yaml`, uncomment:

- the `redis:` service block
- the `REDIS_URL: redis://redis:6379` env var on `alert-agent`
- the `depends_on` block on `alert-agent`
- the `alert-agent-redis` named volume

Then `docker compose up -d`.

## Updating

```bash
docker compose pull
docker compose up -d
```

## Tearing down

```bash
docker compose down                  # keep volumes
docker compose down -v               # also delete SQLite + Redis data
```
