# Agent Loop

## Overview

Agent is the central component that orchestrates the interaction between the LLM and MCP tools.
It implements the **tool-use loop** pattern: the LLM analyzes data, calls tools, receives results,
and continues the analysis — until it produces a final report.

Code: `internal/agent/agent.go`

## Structure

```go
type Agent struct {
    provider llm.Provider                // LLM for chat completion
    executor ToolExecutor                // MCP tools
    llmCfg   config.LLMConfig            // Max tokens, context budget, etc.
    prompts  storage.PromptRepository    // System/summary/resolved/paused prompts (DB-backed)
}

type Result struct {
    Summary string    // 1-2 sentence summary (OK/WARNING/CRITICAL)
    Report  string    // Markdown report
    Images  [][]byte  // Screenshots from tool calls
}
```

The Agent has two entry points:

- `Run(ctx, payload, scenario)` — full investigation loop for **firing** alerts.
- `Notify(ctx, payload, scenario)` — single LLM call for **resolved** / **paused**
  alerts. No tool-use loop; produces a short notification only.

## Execution Flow

```
Run(ctx, payload, scenario)
│
├─ 1. Filter tools for the scenario
│     └─ If scenario.Tools is not empty → FilteredExecutor
│     └─ Otherwise → all MCP tools
│
├─ 2. Build the user message
│     └─ Alert JSON + Scenario prompt + Time budget + Priority
│
├─ 3. Tool-use loop (max 20 iterations, up to 10 tool calls per iteration)
│     │
│     ├─ Check ctx.Err() → timeout? → timeoutFinish()
│     │
│     ├─ If llmCfg.ContextLimit > 0 → trimMessages()
│     │   (replace oldest tool results with "[trimmed]" placeholder)
│     │
│     ├─ ChatCompletion(system prompt, messages, tools)
│     │
│     ├─ FinishReason == "stop"?
│     │   └─ YES → summarize(report) → return Result{Summary, Report, Images}
│     │
│     └─ FinishReason == "tool_call"?
│         └─ For each tool call (silently truncated to 10):
│             ├─ executor.CallTool(name, args)
│             ├─ If error → content = "error: ..."
│             ├─ Collect images from result
│             └─ Append tool result to messages
│
└─ 4. Max iterations reached → timeoutFinish()
```

## Prompts

Four prompts drive the Agent's behavior. They are stored in the database
(`storage.PromptRepository`), seeded on first run from
`internal/storage/seed/prompts.yaml`, and **editable at runtime via the Web UI**
at `/prompts`:

| Key        | Used in                | Purpose                                        |
|------------|------------------------|------------------------------------------------|
| `system`   | `Run` (tool-use loop)  | Agent persona, rules, output format            |
| `summary`  | `Run` (final call)     | 1–2 sentence summary generation                |
| `resolved` | `Notify`               | Short notification for resolved alerts         |
| `paused`   | `Notify`               | Short notification for paused alerts           |

The default `system` prompt describes the investigation goal (understand the
alert, gather evidence, identify root cause, produce a markdown report) and
rules (follow scenario, capture only listed screenshots, never fabricate, etc.).
Because prompts live in the DB, you can tune behavior without redeploying.

## User Message

The Agent builds a single user message with all context:

```
## Alert

```json
{ ...webhook payload... }
```

## Scenario: postgres-high-connections

PostgreSQL high connections alert.
Investigation steps:
1. Query current connection count...
...

Time budget: 3m0s. Priority: high.
```

The entire JSON payload is included so the LLM can see labels, annotations, metric values,
timestamps, fingerprint — and decide for itself what is important.

## Tool Filtering

A scenario can restrict the set of tools via the `tools` field:

```go
type FilteredExecutor struct {
    executor ToolExecutor
    allowed  map[string]struct{}
}
```

- `Tools()` — returns only allowed tools (LLM does not see others)
- `CallTool()` — proxies to the original executor without filtering (in case the LLM somehow discovers another tool)

This reduces token costs and focuses the LLM on relevant tools.

## Timeout and Graceful Finish

There are two completion mechanisms:

### Context timeout

The timeout is taken from `scenario.Timeout` and set via `context.WithTimeout` in the webhook handler.
When the context is cancelled:
1. Agent catches `ctx.Err()` at the beginning of an iteration
2. Or the LLM call returns an error due to a cancelled context
3. In both cases — `timeoutFinish()` is called

### Max iterations

Hard limit — **20 iterations** of the tool-use loop. Each iteration allows up to **10 tool calls**.
Protection against infinite loops. After 20 iterations — `timeoutFinish()` is also called.

### timeoutFinish()

```go
func (a *Agent) timeoutFinish(ctx context.Context, messages []llm.Message) (*Result, error)
```

1. Adds a user message: _"Time is up. Provide your investigation report now based on what you have gathered so far."_
2. Makes one final LLM call **without tools** (so the LLM cannot stall)
3. Generates a summary from the report via a separate LLM call
4. Returns Result with summary, report, and collected images

## Summary Generation

After the investigation completes, a separate LLM call generates a 1-2 sentence summary:

```
Format: <OK/WARNING/CRITICAL> <alert name> at <value>. <root cause>.
```

- Max 280 characters, truncated to last sentence if over 500 chars
- Token budget for this call is configurable via `llm.summary_max_tokens` (default `4096`)
- Reasoning models (e.g. gpt-5-mini) need large budgets because reasoning tokens count toward `max_completion_tokens`
- If generation fails, summary is empty and the full report is sent as the message instead

## Image Collection

During the tool-use loop, the Agent collects images from tool results:

```go
if len(toolResult.Images) > 0 {
    images = append(images, toolResult.Images...)
}
```

Images come from MCP tools like `get_panel_image` — these are screenshots
of dashboard panels or entire dashboards in PNG format.

Collected images:
- Are returned in `Result.Images`
- The webhook handler sends them to Telegram as separate photos

## Tool Error Handling

If `CallTool` returns an error — the Agent **does not stop**:

```go
if callErr != nil {
    content = fmt.Sprintf("error: %v", callErr)
}
```

The error is passed to the LLM as a text tool result. The LLM sees that the tool
failed and can:
- Try a different tool
- Continue with other data
- Note the issue in the report

## LLM Provider

The Agent works with the `llm.Provider` abstraction, not with a specific API directly.
The provider is selected at startup from `llm.provider` (`openai` or `anthropic`).

Two implementations ship today:

- `internal/llm/openai/openai.go` — OpenAI Chat Completions API with function
  calling. Works with any OpenAI-compatible API via `base_url`.
- `internal/llm/anthropic/anthropic.go` — Anthropic Messages API (Claude). It
  streams and accumulates the response to avoid HTTP timeouts on large
  `max_tokens` values, and supports prompt caching plus adaptive thinking.

Format conversion (OpenAI):
- `llm.Tool` → OpenAI `FunctionDefinition`
- `llm.Message` with `ToolCalls` → OpenAI assistant message with function calls
- `llm.Message` with `ToolCallID` → OpenAI tool message
- OpenAI finish reason `"tool_calls"` → `llm.FinishReasonToolCall`

Format conversion (Anthropic):
- `llm.Tool` → Anthropic `tool` with an input JSON schema
- `llm.Message` with `ToolCalls` → assistant message with `tool_use` blocks
- consecutive `llm.Message` with `ToolCallID` → one user message of coalesced
  `tool_result` blocks (the Messages API requires alternating turns)
- `llm.ThinkingBlock` is echoed back, signatures intact, ahead of `tool_use`
- Anthropic stop reason `tool_use` → `llm.FinishReasonToolCall`

### Reasoning / thinking

When `reasoning_effort` is set, the provider enables its reasoning mode. For
Anthropic the returned thinking blocks are carried in `llm.ChatResponse.Thinking`
and replayed on the next assistant turn (signatures preserved) — the API rejects
a tool-use turn otherwise. OpenAI's reasoning is hidden, so no blocks are
returned. See [configuration.md](configuration.md#reasoning--thinking-reasoning_effort)
for the per-provider details.
