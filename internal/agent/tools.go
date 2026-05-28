package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tsisar/alert-agent/internal/llm"
	"github.com/tsisar/extended-log-go/log"
)

// ToolExecutor provides tool definitions for the LLM and executes tool calls.
type ToolExecutor interface {
	Tools() []llm.Tool
	CallTool(ctx context.Context, name string, arguments json.RawMessage) (*llm.ToolResult, error)
}

// FilteredExecutor wraps a ToolExecutor and exposes only the allowed tools.
// If the allow list is empty, all tools are exposed.
type FilteredExecutor struct {
	executor ToolExecutor
	allowed  map[string]struct{}
}

func NewFilteredExecutor(executor ToolExecutor, allowedTools []string) *FilteredExecutor {
	allowed := make(map[string]struct{}, len(allowedTools))
	for _, name := range allowedTools {
		allowed[name] = struct{}{}
	}
	return &FilteredExecutor{executor: executor, allowed: allowed}
}

func (f *FilteredExecutor) Tools() []llm.Tool {
	all := f.executor.Tools()
	if len(f.allowed) == 0 {
		return all
	}

	filtered := make([]llm.Tool, 0, len(f.allowed))
	for _, t := range all {
		if _, ok := f.allowed[t.Name]; ok {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

func (f *FilteredExecutor) CallTool(ctx context.Context, name string, arguments json.RawMessage) (*llm.ToolResult, error) {
	if len(f.allowed) > 0 {
		if _, ok := f.allowed[name]; !ok {
			log.Warnf("[agent] tool %q blocked by scenario allowlist", name)
			return &llm.ToolResult{
				Content: fmt.Sprintf("tool %q is not available in this scenario", name),
			}, nil
		}
	}
	return f.executor.CallTool(ctx, name, arguments)
}
