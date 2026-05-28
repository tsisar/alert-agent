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
	FinishReason FinishReason
	PromptTokens int
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
	ToolCalls  []ToolCall // present when Role == RoleAssistant and model calls tools
	ToolCallID string     // present when Role == RoleTool
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
