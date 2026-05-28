# Infrastructure Dependencies

Services required for Alert Agent to function fully.

## Required

### Grafana

Central UI for dashboards, alerting, and visualization.

- **Image:** `grafana/grafana:latest`
- **Port:** `3000`
- **Role:** alert source (webhook), host for dashboards and panels
- **Configuration for Alert Agent:**
  - Unified Alerting enabled (`GF_UNIFIED_ALERTING_ENABLED=true`)
  - Webhook contact point → `http://<alert-agent>:8080/webhook`
  - Image Renderer connected (for `get_panel_image`)

### Grafana Image Renderer

Renders panels/dashboards to PNG. Without it, the `get_panel_image` tool does not work.

- **Image:** `grafana/grafana-image-renderer:latest`
- **Port:** `8081`
- **Connection to Grafana:**
  - `GF_RENDERING_SERVER_URL=http://renderer:8081/render`
  - `GF_RENDERING_CALLBACK_URL=http://grafana:3000/`

### MCP Grafana Server

MCP server through which Alert Agent accesses Grafana tools.

- **Image:** `grafana/mcp-grafana:latest`
- **Transport:** SSE
- **Variables:**
  - `GRAFANA_URL` — Grafana address
  - `GRAFANA_API_KEY` — API key or service account token
- **Configuration in `.mcp.json`:**
  ```json
  {
    "mcpServers": {
      "grafana": {
        "type": "sse",
        "url": "http://mcp-grafana:8082/sse"
      }
    }
  }
  ```

### Prometheus

Metrics storage and querying. Used by the `query_prometheus`, `list_prometheus_metric_names`, `list_prometheus_label_values` tools and others.

- **Image:** `prom/prometheus:latest`
- **Port:** `9090`
- **Connection to Grafana:** add as a `prometheus` type datasource
- **Recommendations:**
  - `--storage.tsdb.retention.time=30d`
  - `--web.enable-lifecycle` for config reload without restart

### Loki

Log aggregation and querying. Used by the `query_loki_logs`, `query_loki_patterns`, `list_loki_label_names` tools and others.

- **Image:** `grafana/loki:latest`
- **Port:** `3100`
- **Connection to Grafana:** add as a `loki` type datasource

### LLM Provider

An OpenAI-compatible API for the agent to operate.

- Configured via the `llm:` section in `config.yaml`
- Variable: `LLM_API_KEY`
- Any OpenAI-compatible provider is supported (OpenAI, Azure OpenAI, vLLM, Ollama with OpenAI API)

## Optional

### Alertmanager

Alert routing from Prometheus. If alerts go through Grafana Unified Alerting, a separate Alertmanager is not needed.

- **Image:** `prom/alertmanager:latest`
- **Port:** `9093`
- **When needed:** if using Prometheus alerting rules instead of Grafana alerting

### Promtail

Log collection and forwarding to Loki.

- **Image:** `grafana/promtail:latest`
- **Role:** tail logs from files, Docker, journald → push to Loki
- **Alternatives:** Grafana Alloy, Fluentd, Vector

### Telegram Bot

For sending investigation results.

- Create a bot via [@BotFather](https://t.me/BotFather)
- Variable: `TELEGRAM_BOT_TOKEN`
- Channel/group ID in scenario's `channels.telegram` field (stored in database)

### Pyroscope

Continuous profiling. Used by the `query_pyroscope`, `list_pyroscope_profile_types` tools.

- **Image:** `grafana/pyroscope:latest`
- **Port:** `4040`
- **Connection to Grafana:** add as a `grafana-pyroscope-datasource` type datasource
- **When needed:** if scenarios include profiling (CPU, memory)

### Tempo

Distributed tracing. Used by the `find_slow_requests` tool.

- **Image:** `grafana/tempo:latest`
- **Port:** `3200`
- **Connection to Grafana:** add as a `tempo` type datasource
- **When needed:** if scenarios include slow request analysis

### Grafana OnCall

On-call management and escalations. Used by the `list_oncall_schedules`, `get_current_oncall_users`, `list_alert_groups` tools.

- **Options:** Grafana Cloud OnCall or self-hosted
- **When needed:** if scenarios include determining who is on call or escalation

### Grafana Incident

Incident management. Used by the `create_incident`, `list_incidents`, `add_activity_to_incident` tools.

- **Options:** Grafana Cloud or self-hosted
- **When needed:** if scenarios include automatic incident creation

### Grafana Sift

Automated analysis and investigation. Used by the `get_sift_investigation`, `find_error_pattern_logs`, `find_slow_requests` tools.

- **Availability:** Grafana Cloud
- **When needed:** if scenarios include automatic anomaly detection

## Minimum Set

For basic alert investigation with metrics, logs, and screenshots:

```
Grafana + Image Renderer + MCP Grafana + Prometheus + Loki + LLM Provider + Telegram Bot
```

## Service Relationships

```
Prometheus ──┐
Loki ────────┤
Tempo ───────┼──► Grafana ◄──── Image Renderer
Pyroscope ───┤       │
OnCall ──────┘       │
                     ▼
              MCP Grafana Server
                     │
                     ▼
              Alert Agent ──► Telegram / Slack
                     ▲
                     │
              Grafana Alerting (webhook)
```
