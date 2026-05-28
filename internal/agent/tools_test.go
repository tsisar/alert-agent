package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/tsisar/alert-agent/internal/llm"
)

type mockExecutor struct {
	tools      []llm.Tool
	failTool   string            // if set, CallTool returns an error for this tool name
	toolImages map[string][]byte // if set, CallTool returns an image for this tool name
}

func (m *mockExecutor) Tools() []llm.Tool {
	return m.tools
}

func (m *mockExecutor) CallTool(_ context.Context, name string, _ json.RawMessage) (*llm.ToolResult, error) {
	if m.failTool != "" && name == m.failTool {
		return nil, fmt.Errorf("tool %q not found", name)
	}
	result := &llm.ToolResult{Content: fmt.Sprintf("called:%s", name)}
	if m.toolImages != nil {
		if img, ok := m.toolImages[name]; ok {
			result.Images = [][]byte{img}
		}
	}
	return result, nil
}

var testTools = []llm.Tool{
	{Name: "query_prometheus", Description: "Run PromQL"},
	{Name: "query_loki_logs", Description: "Search logs"},
	{Name: "get_panel_image", Description: "Screenshot panel"},
	{Name: "search_dashboards", Description: "Find dashboards"},
	{Name: "get_annotations", Description: "Get annotations"},
}

func TestFilteredExecutor_NoFilter(t *testing.T) {
	fe := NewFilteredExecutor(&mockExecutor{tools: testTools}, nil)

	tools := fe.Tools()
	if len(tools) != len(testTools) {
		t.Fatalf("expected %d tools, got %d", len(testTools), len(tools))
	}
}

func TestFilteredExecutor_EmptyFilter(t *testing.T) {
	fe := NewFilteredExecutor(&mockExecutor{tools: testTools}, []string{})

	tools := fe.Tools()
	if len(tools) != len(testTools) {
		t.Fatalf("expected %d tools, got %d", len(testTools), len(tools))
	}
}

func TestFilteredExecutor_WithFilter(t *testing.T) {
	allowed := []string{"query_prometheus", "get_panel_image"}
	fe := NewFilteredExecutor(&mockExecutor{tools: testTools}, allowed)

	tools := fe.Tools()
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}

	names := map[string]bool{}
	for _, tool := range tools {
		names[tool.Name] = true
	}
	if !names["query_prometheus"] {
		t.Fatal("expected query_prometheus")
	}
	if !names["get_panel_image"] {
		t.Fatal("expected get_panel_image")
	}
}

func TestFilteredExecutor_CallToolPassesThrough(t *testing.T) {
	fe := NewFilteredExecutor(&mockExecutor{tools: testTools}, []string{"query_prometheus"})

	result, err := fe.CallTool(context.Background(), "query_prometheus", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Content != "called:query_prometheus" {
		t.Fatalf("expected 'called:query_prometheus', got %q", result.Content)
	}
}

func TestFilteredExecutor_CallToolBlocked(t *testing.T) {
	fe := NewFilteredExecutor(&mockExecutor{tools: testTools}, []string{"query_prometheus"})

	result, err := fe.CallTool(context.Background(), "get_annotations", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `tool "get_annotations" is not available in this scenario`
	if result.Content != want {
		t.Fatalf("expected %q, got %q", want, result.Content)
	}
}

func TestFilteredExecutor_CallToolNoFilter(t *testing.T) {
	fe := NewFilteredExecutor(&mockExecutor{tools: testTools}, nil)

	result, err := fe.CallTool(context.Background(), "get_annotations", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Content != "called:get_annotations" {
		t.Fatalf("expected 'called:get_annotations', got %q", result.Content)
	}
}
