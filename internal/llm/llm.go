package llm

import "context"

// Provider sends a single chat completion request and returns the model's response.
// The tool use loop is orchestrated by the caller, not by the provider.
type Provider interface {
	ChatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
}

type ChatRequest struct {
	Model        string
	SystemPrompt string
	Messages     []Message
	Tools        []Tool
	MaxTokens    int
}

type ChatResponse struct {
	Content      string
	ToolCalls    []ToolCall
	Thinking     []ThinkingBlock // reasoning blocks that must be echoed back on the next turn (Anthropic)
	FinishReason FinishReason
	PromptTokens int
}

// ThinkingBlock holds a provider's reasoning output. Some providers (Anthropic)
// require these blocks to be preserved and sent back unchanged on the following
// turn when the assistant also called tools. Providers whose reasoning is hidden
// (OpenAI reasoning models) produce none.
type ThinkingBlock struct {
	Text      string
	Signature string
	Redacted  bool   // true for encrypted/redacted reasoning; Text/Signature are empty
	Data      string // opaque payload for redacted reasoning
}

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type Message struct {
	Role       Role
	Content    string
	ToolCalls  []ToolCall      // present when Role == RoleAssistant and model calls tools
	ToolCallID string          // present when Role == RoleTool
	Thinking   []ThinkingBlock // present when Role == RoleAssistant and the provider returned reasoning
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments string // raw JSON
}

// ToolResult holds the output of a single tool invocation.
type ToolResult struct {
	Content string
	Images  [][]byte
}

type Tool struct {
	Name        string
	Description string
	Parameters  map[string]any // JSON Schema
}

type FinishReason string

const (
	FinishReasonStop     FinishReason = "stop"
	FinishReasonToolCall FinishReason = "tool_call"
)
