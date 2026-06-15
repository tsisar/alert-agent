# Configuration

## Main Configuration

The file `configs/config.yaml` is the main service configuration. Viper looks for `config.yaml` in:

1. `.` (current directory)
2. `./configs`
3. `/etc/alert-agent`

You can also specify the path explicitly via CLI.

```yaml
http:
  addr: ":8080"              # HTTP server address

mcp:
  config_path: ".mcp.json"   # Path to MCP servers config

database:
  driver: "sqlite"           # Database driver: sqlite, postgres, mysql
  dsn: "./alert-agent.db"    # File path (sqlite) or connection string (pg/mysql)

llm:
  provider: "openai"         # LLM provider: openai | anthropic
  api_key: ""                # API key (prefer env var)
  base_url: ""               # Custom endpoint (for compatible APIs)
  model: ""                  # Model (empty = provider default: openai→gpt-4o, anthropic→claude-opus-4-8)
  max_tokens: 32768          # Token limit per response
  reasoning_effort: ""       # Reasoning/thinking effort: empty=off, low|medium|high (see note below)
  summary_max_tokens: 4096   # Token limit for summary/notify calls (increase for reasoning models)
  context_limit: 0           # Model context window (0 = no limit)

telegram:
  bot_token: ""              # Telegram bot token (prefer env var)
  default_channel: ""        # Default Telegram chat ID

slack:
  bot_token: ""              # Slack bot token (prefer env var)
  default_channel: ""        # Default Slack channel ID

redis:
  url: ""                    # Redis URL (empty = in-memory, e.g. "redis://localhost:6379")

alert_cooldown: "48h"        # Dedup cooldown for repeated alerts
```

## Environment Variables

All parameters can be overridden via env vars. A `.env` file is loaded automatically if present.

| Env var                             | Config key                 | Default          | Description                                                   |
|-------------------------------------|----------------------------|------------------|---------------------------------------------------------------|
| `HTTP_ADDR`                         | `http.addr`                | `:8080`          | Server address                                                |
| `LLM_PROVIDER`                      | `llm.provider`             | `openai`         | LLM provider: `openai` or `anthropic`                         |
| `LLM_API_KEY` or `OPENAI_API_KEY`   | `llm.api_key`              | —                | LLM API key                                                   |
| `LLM_BASE_URL` or `OPENAI_BASE_URL` | `llm.base_url`             | —                | Custom base URL                                               |
| `LLM_MODEL`                         | `llm.model`                | provider default | Model name (empty → `gpt-4o` / `claude-opus-4-8`)             |
| `LLM_MAX_TOKENS`                    | `llm.max_tokens`           | `32768`          | Max tokens per response                                       |
| `LLM_REASONING_EFFORT`              | `llm.reasoning_effort`     | — (off)          | Reasoning/thinking effort: `low`, `medium`, `high` (see note) |
| `LLM_SUMMARY_MAX_TOKENS`            | `llm.summary_max_tokens`   | `4096`           | Max tokens for summary/notify (increase for reasoning models) |
| `LLM_CONTEXT_LIMIT`                 | `llm.context_limit`        | `0`              | Model context window (0 = no limit)                           |
| `MCP_CONFIG_PATH`                   | `mcp.config_path`          | `.mcp.json`      | Path to MCP config                                            |
| `DATABASE_DRIVER`                   | `database.driver`          | `sqlite`         | Database driver (sqlite, postgres, mysql)                     |
| `DATABASE_DSN`                      | `database.dsn`             | `alert-agent.db` | Database connection string                                    |
| `TELEGRAM_BOT_TOKEN`                | `telegram.bot_token`       | —                | Telegram bot token                                            |
| `TELEGRAM_DEFAULT_CHANNEL`          | `telegram.default_channel` | —                | Default Telegram chat ID                                      |
| `SLACK_BOT_TOKEN`                   | `slack.bot_token`          | —                | Slack bot token                                               |
| `SLACK_DEFAULT_CHANNEL`             | `slack.default_channel`    | —                | Default Slack channel ID                                      |
| `REDIS_URL`                         | `redis.url`                | —                | Redis URL for dedup/queue (empty = in-memory)                 |
| `ALERT_COOLDOWN`                    | `alert_cooldown`           | `48h`            | Dedup cooldown for repeated alerts                            |
| `LOG_SAVE`                          | —                          | `false`          | Write logs to a file (handled by `extended-log-go`)           |
| `LOG_LEVEL`                         | —                          | `debug`          | Log level (`debug`, `info`, `warn`, `error`)                  |
| `LOG_TIMEZONE`                      | —                          | `UTC`            | Timezone for log timestamps                                   |
| `LOG_SHOW_CALLER`                   | —                          | `true`           | Include caller file:line in log lines                         |

Priority: env var > config.yaml > default.

### Reasoning / thinking (`reasoning_effort`)

Empty disables it. A non-empty value (`low`, `medium`, `high`) enables the
provider's reasoning mode:

- **OpenAI** — sent as the `reasoning_effort` request parameter. Only valid for
  reasoning models (`o`-series, `gpt-5`); enabling it on a non-reasoning model
  such as `gpt-4o` will error.
- **Anthropic** — enables adaptive thinking with the given `output_config.effort`
  (Claude also accepts `xhigh` and `max`). Reasoning blocks are preserved across
  the tool-use loop automatically.

## MCP Configuration

The `.mcp.json` file defines the MCP servers the agent connects to.

```json
{
  "mcpServers": {
    "grafana": {
      "type": "sse",
      "url": "https://mcp-grafana.example.com/sse"
    }
  }
}
```

| Field  | Description                             |
|--------|-----------------------------------------|
| `type` | Transport type. Only `sse` is supported |
| `url`  | URL of the MCP server SSE endpoint      |

You can specify multiple servers — the Manager will connect to all of them and aggregate their tools.

## Database

Scenarios and LLM prompts are stored in a database. On first startup, the database is auto-migrated and
seeded with default data embedded in the binary (from `internal/storage/seed/`).

### Supported Drivers

| Driver     | DSN example                                                   |
|------------|---------------------------------------------------------------|
| `sqlite`   | `./alert-agent.db` or `/data/alert-agent.db`                  |
| `postgres` | `host=localhost user=agent password=secret dbname=alertagent` |
| `mysql`    | `agent:secret@tcp(localhost:3306)/alertagent?parseTime=true`  |

SQLite is the default (embedded, zero-config). For production with multiple replicas, use PostgreSQL or MySQL.

## Scenarios

Scenarios are stored in the database and define how the agent investigates each alert type.
Default scenarios are seeded on first startup from embedded YAML (`internal/storage/seed/scenarios.yaml`).

### Scenario Fields

| Field               | Type              | Required | Description                                 |
|---------------------|-------------------|----------|---------------------------------------------|
| `name`              | string            | yes      | Unique scenario name                        |
| `match`             | map[string]string | yes      | Labels for matching. Empty `{}` = catch-all |
| `prompt`            | string            | yes      | Context and instructions for the LLM        |
| `tools`             | []string          | no       | List of allowed MCP tools. Empty = all      |
| `channels.telegram` | string            | no       | Telegram chat ID (use `"-"` to disable)     |
| `channels.slack`    | string            | no       | Slack channel ID (use `"-"` to disable)     |
| `timeout`           | duration          | yes      | Time limit (`1m`, `3m`, `30s`, etc.)        |
| `priority`          | string            | yes      | `normal`, `high`, or `critical`             |
| `send_images`       | bool              | no       | Send screenshots to chat                    |

### Matching Logic

1. Scenarios are checked in `order_index` order (ascending)
2. All labels in `match` must exactly match the alert's `commonLabels` (AND)
3. First match wins
4. A scenario with an empty match acts as a catch-all (checked last)
5. If nothing matches and there is no catch-all — the webhook returns `202` without investigation

### Channel Resolution

Each channel (Telegram, Slack) follows this resolution order:

| Value in scenario | Behavior                      |
|-------------------|-------------------------------|
| (not set)         | Falls back to default channel |
| `"-"`             | Channel explicitly disabled   |
| `"C0A699M79EF"`   | Uses the specified channel    |

This allows per-scenario channel control. For example, to send only to Telegram:

```yaml
channels:
  telegram: "-4910737562"
  slack: "-"
```

### Scenario Philosophy

A scenario is a set of **recommendations** for the LLM, not a rigid script:

- The LLM can skip suggested steps if they are not relevant
- The LLM can investigate things not mentioned in the scenario
- The LLM has access to all tools (unless `tools` restricts them)
- Investigation depth is adaptive — the LLM decides where to dig deeper

It is like giving a new on-call engineer context: "when you see this alert, it's usually worth checking X and Y".
