# alert-agent Helm chart

A Helm chart for deploying [Alert Agent](https://github.com/tsisar/alert-agent)
to Kubernetes.

## TL;DR

```bash
helm install alert-agent ./deployment/helm/chart \
  --set secret.LLM_API_KEY=sk-... \
  --set secret.TELEGRAM_BOT_TOKEN=... \
  --set secret.SLACK_BOT_TOKEN=xoxb-... \
  --set 'mcp.servers.grafana.url=http://mcp-grafana.monitoring:8000/sse'
```

## What this chart deploys

- A single-replica **Deployment** running the `alert-agent serve` command. The
  in-process queue is sequential per pod — running multiple replicas without
  Redis would interleave notification messages.
- A **PersistentVolumeClaim** (`ReadWriteOnce`) for the SQLite database. The
  Deployment uses `strategy: Recreate` to avoid multi-attach errors during
  rollouts.
- A **ConfigMap** rendered from `.Values.mcp.servers` and mounted at
  `$MCP_CONFIG_PATH` (default `/etc/alert-agent/mcp.json`).
- A **Secret** with `LLM_API_KEY`, `TELEGRAM_BOT_TOKEN`, `SLACK_BOT_TOKEN`.
  Rendered only if at least one secret value is non-empty.
- A **Service** on port 8080 (ClusterIP by default).
- An optional **Ingress** (`ingress.enabled: true`).
- A **ServiceAccount** with `automountServiceAccountToken: false`.

The pod runs as non-root (uid 65534), with `readOnlyRootFilesystem`, dropped
capabilities, and the `RuntimeDefault` seccomp profile.

## Values

The full set lives in [`values.yaml`](values.yaml). Highlights:

| Key                           | Default                              | Description                                                  |
|-------------------------------|--------------------------------------|--------------------------------------------------------------|
| `image.repository`            | `ghcr.io/tsisar/alert-agent`         | Container image                                              |
| `image.tag`                   | `latest`                             | Image tag — pin to a release in production                   |
| `replicas`                    | `1`                                  | Don't raise without Redis (sequential queue per pod)         |
| `env.LLM_MODEL`               | `gpt-4o`                             | LLM model                                                    |
| `env.LLM_MAX_TOKENS`          | `32768`                              | Token budget per LLM call in the tool-use loop               |
| `env.LLM_SUMMARY_MAX_TOKENS`  | `4096`                               | Token budget for the summary/notify call                     |
| `env.DATABASE_DRIVER`         | `sqlite`                             | `sqlite`, `postgres`, or `mysql`                             |
| `env.DATABASE_DSN`            | `/data/alert-agent.db`               | DSN; defaults assume the bundled PVC                         |
| `env.REDIS_URL`               | `""`                                 | Empty = in-memory dedup + queue; set to enable Redis Streams |
| `env.ALERT_COOLDOWN`          | `48h`                                | Dedup window for repeated firing alerts                      |
| `persistence.size`            | `64Mi`                               | PVC size                                                     |
| `persistence.storageClass`    | `""`                                 | Empty = cluster default                                      |
| `secret.LLM_API_KEY`          | `""`                                 | LLM provider API key                                         |
| `secret.TELEGRAM_BOT_TOKEN`   | `""`                                 | Telegram bot token                                           |
| `secret.SLACK_BOT_TOKEN`      | `""`                                 | Slack bot token (`chat:write`, `files:write`)                |
| `mcp.servers.<name>.type`     | `sse`                                | Only `sse` is supported today                                |
| `mcp.servers.<name>.url`      | `http://mcp-grafana.example.com/sse` | SSE endpoint of the MCP server                               |
| `ingress.enabled`             | `false`                              | Enable an `Ingress` (legacy/classic networking)              |
| `httpRoute.enabled`           | `false`                              | Enable a Gateway API `HTTPRoute` (kgateway/Cilium/Istio)     |
| `httpRoute.gateway.name`      | `kg`                                 | Parent Gateway resource name                                 |
| `httpRoute.gateway.namespace` | `kgateway-system`                    | Parent Gateway namespace                                     |
| `httpRoute.hostnames`         | `[alert-agent.example.com]`          | Hostnames the route accepts                                  |
| `httpRoute.paths`             | `[{path: /, type: PathPrefix}]`      | Path matches. Scope to `/webhook` to hide the Web UI         |

Leave any `env.*` value at `""` to fall back to the binary's compiled-in
default (see `configs/config.yaml` and `docs/configuration.md`).

## Grafana Alertmanager wiring

Inside the cluster, point a Grafana Alertmanager webhook contact point at:

```
URL: http://alert-agent.<namespace>.svc.cluster.local:8080/webhook
```

Enable `send_resolved: true` to receive resolved-alert notifications.

## Multiple MCP servers

Override `mcp.servers` with multiple entries:

```yaml
mcp:
  servers:
    grafana:
      type: sse
      url: http://mcp-grafana.monitoring:8000/sse
    tempo:
      type: sse
      url: http://mcp-tempo.monitoring:8000/sse
```

The Agent auto-discovers each server's tools at startup and exposes them
under a namespaced name (`<server>__<tool>`, e.g. `grafana__search_dashboards`).

## Using an external database

For multi-replica or HA deployments, switch off SQLite:

```yaml
env:
  DATABASE_DRIVER: "postgres"
  DATABASE_DSN: "host=postgres.db user=alertagent password=$(PG_PASS) dbname=alertagent sslmode=require"
```

The PVC is still created but unused if you do this — set
`persistence.size: 0` and either remove the PVC or accept the empty volume.
