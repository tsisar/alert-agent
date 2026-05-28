# MCP Integration

## Overview

Alert Agent uses the [Model Context Protocol (MCP)](https://modelcontextprotocol.io/)
to access Grafana tools. The Go service acts as a **bridge** between the LLM tool-use API
and MCP servers: the LLM decides which tools to call, and the Agent translates the calls into MCP protocol.

Code: `internal/mcp/`

## Architecture

```
LLM (OpenAI API)              Agent                    MCP Server (Grafana)
     │                          │                            │
     │── tool_call ────────────►│                            │
     │   name: query_prometheus │── CallToolRequest ────────►│
     │   args: {expr: "..."}    │                            │
     │                          │◄── CallToolResult ─────────│
     │                          │    [TextContent, ...]      │
     │◄── tool_result ──────────│                            │
     │   content: "..."         │                            │
     │                          │                            │
     │── tool_call ────────────►│                            │
     │   name: get_panel_image  │── CallToolRequest ────────►│
     │                          │◄── CallToolResult ─────────│
     │                          │    [ImageContent(base64)]  │
     │◄── tool_result ──────────│                            │
     │   + image stored         │                            │
```

## Components

### Client

`Client` — a connection to a single MCP server.

```go
type Client struct {
    name   string
    client *mcpclient.Client // mcp-go SSE client
}
```

**Lifecycle:**

1. Create SSE client: `mcpclient.NewSSEMCPClient(url)`
2. Connect: `client.Start(ctx)`
3. Initialize MCP session: `client.Initialize(ctx, req)` with protocol version and client info
4. Ready for `listTools()` and `callTool()`

**Supported transports:** only **SSE** (Server-Sent Events). stdio is not supported.

### Manager

`Manager` — aggregates multiple MCP servers and routes tool calls.

```go
type Manager struct {
    clients  []*Client
    toolMap  map[string]*Client // tool name → owning client
    toolList []llm.Tool
}
```

**Initialization:**

1. Loads the `.mcp.json` config
2. Creates a `Client` for each server
3. Calls `listTools()` on each server
4. Builds `toolMap` — mapping tool name → client
5. Aggregates all tools into a single list

**Manager implements `agent.ToolExecutor`:**

- `Tools()` — returns the combined tool list from all servers
- `CallTool(name, args)` — finds the owning server via `toolMap`, converts arguments, and makes the call

## Tool Conversion

MCP tool definitions → internal `llm.Tool` format → OpenAI function definitions.

```
MCP Tool                        llm.Tool                     OpenAI Function
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────────┐
│ Name            │────►│ Name             │────►│ Name                │
│ Description     │────►│ Description      │────►│ Description         │
│ InputSchema     │────►│ Parameters       │────►│ Parameters (JSON    │
│   Properties    │     │   map[string]any │     │   Schema)           │
│   Required      │     │                  │     │                     │
└─────────────────┘     └──────────────────┘     └─────────────────────┘
```

`convertTool()` extracts `Properties` and `Required` from the MCP `InputSchema`
and packs them into a `map[string]any` with `type: "object"`.

## Result Handling

An MCP `CallToolResult` contains an array of `Content` with different types:

### TextContent

Text data (JSON metrics, log entries, dashboard descriptions).
All text blocks are concatenated with `\n`.

### ImageContent

Base64-encoded PNG images (panel screenshots).
Decoded from base64 and stored as `[]byte`.

```go
func extractResult(result *mcp.CallToolResult) (*llm.ToolResult, error) {
    tr := &llm.ToolResult{}
    for _, c := range result.Content {
        switch v := c.(type) {
        case mcp.TextContent:
            tr.Content += v.Text
        case mcp.ImageContent:
            decoded, _ := base64.StdEncoding.DecodeString(v.Data)
            tr.Images = append(tr.Images, decoded)
        }
    }
    return tr, nil
}
```

### Errors

If `result.IsError == true`, the Manager returns an error with the result text.
The Agent passes this error to the LLM as a text tool result.

## MCP Configuration

File `.mcp.json`:

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

Multiple servers are supported — each gets a separate connection:

```json
{
  "mcpServers": {
    "grafana": {
      "type": "sse",
      "url": "https://mcp-grafana.example.com/sse"
    },
    "other-server": {
      "type": "sse",
      "url": "https://other-mcp.example.com/sse"
    }
  }
}
```

### Tool name namespacing

Tools are exposed to the Agent under a **namespaced** name of the form
`<server>__<tool>` — for example, `grafana__search_dashboards`. The Manager
prefixes every discovered tool with its server name, so collisions across
servers are not possible. The Agent (and scenarios that filter tools by name)
must use the namespaced form.

### Scenario tool filtering

A scenario can restrict the LLM to a subset of tools by listing their
namespaced names in its `tools` field. The Agent wraps the Manager in a
`FilteredExecutor` (`internal/agent/tools.go`) which:

- returns only the allowed tools from `Tools()` so the LLM never sees the rest;
- proxies `CallTool()` to the underlying Manager without filtering, so a
  forced-name call from a misbehaving LLM still works rather than silently
  failing — but normally the LLM only sees the allowlist.

## Automatic Reconnect

MCP SSE sessions can expire when the server restarts or the session TTL elapses.
The Client detects stale sessions and transparently reconnects:

1. A tool call or `listTools()` fails with a session error (404 "Could not find session", connection refused, EOF)
2. The Client closes the old SSE connection
3. Creates a new SSE client, starts it, and initializes a new MCP session
4. Retries the failed call once on the new session
5. If reconnect itself fails — both errors are returned

Reconnect is per-client and protected by a mutex to prevent concurrent reconnect attempts.
The `toolMap` in the Manager is not refreshed — tool names are assumed stable across restarts.

## Available Grafana Tools

Typical MCP tools from mcp-grafana (the full list depends on the server version):

| Tool                           | Description                              |
|--------------------------------|------------------------------------------|
| `query_prometheus`             | Execute a PromQL query                   |
| `query_loki_logs`              | Search logs via LogQL                    |
| `get_panel_image`              | Panel or dashboard screenshot (PNG)      |
| `search_dashboards`            | Search dashboards by name                |
| `get_dashboard_by_uid`         | Get a dashboard by UID                   |
| `get_dashboard_summary`        | Short dashboard summary                  |
| `get_annotations`              | Grafana annotations (deploys, etc.)      |
| `list_prometheus_metric_names` | List of available metrics                |
| `list_prometheus_label_values` | Prometheus label values                  |
| `list_loki_label_names`        | Available labels in Loki                 |
| `list_datasources`             | List of datasources                      |

The full list is automatically retrieved at startup via the `tools/list` RPC.
For the authoritative, up-to-date Grafana MCP tool catalog (parameters,
return values, recent additions), see the upstream documentation at
[github.com/grafana/mcp-grafana](https://github.com/grafana/mcp-grafana).
This project does not maintain a frozen copy because Grafana's MCP server
evolves independently — the Agent auto-discovers whatever tools the
configured servers expose.
