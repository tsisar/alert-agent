package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// startTestMCPServer creates a real MCP server with tools and returns its SSE URL.
func startTestMCPServer(t *testing.T) string {
	t.Helper()

	mcpServer := server.NewMCPServer(
		"test-server",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	echoTool := mcp.NewTool("echo",
		mcp.WithDescription("Echoes the input back"),
		mcp.WithString("message",
			mcp.Required(),
			mcp.Description("Message to echo"),
		),
	)
	mcpServer.AddTool(echoTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		msg, _ := req.RequireString("message")
		return mcp.NewToolResultText(fmt.Sprintf("echo: %s", msg)), nil
	})

	addTool := mcp.NewTool("add",
		mcp.WithDescription("Adds two numbers"),
		mcp.WithNumber("a", mcp.Required(), mcp.Description("First number")),
		mcp.WithNumber("b", mcp.Required(), mcp.Description("Second number")),
	)
	mcpServer.AddTool(addTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		a, _ := req.RequireFloat("a")
		b, _ := req.RequireFloat("b")
		return mcp.NewToolResultText(fmt.Sprintf("%.2f", a+b)), nil
	})

	imageTool := mcp.NewTool("get_image",
		mcp.WithDescription("Returns a test image"),
	)
	mcpServer.AddTool(imageTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		imgData := base64.StdEncoding.EncodeToString([]byte("fake-png-data"))
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.NewTextContent("Here is the panel screenshot"),
				mcp.NewImageContent(imgData, "image/png"),
			},
		}, nil
	})

	ts := server.NewTestServer(mcpServer)
	t.Cleanup(ts.Close)

	return ts.URL
}

func writeTempMCPConfig(t *testing.T, servers map[string]ServerConfig) string {
	t.Helper()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".mcp.json")

	sf := ServersFile{MCPServers: servers}
	data, err := json.Marshal(sf)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(cfgPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	return cfgPath
}

func TestNewClient_ConnectAndListTools(t *testing.T) {
	url := startTestMCPServer(t)

	ctx := context.Background()
	c, err := newClient(ctx, "test", ServerConfig{Type: "sse", URL: url + "/sse"})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer func() { _ = c.close() }()

	tools, err := c.listTools(ctx)
	if err != nil {
		t.Fatalf("failed to list tools: %v", err)
	}

	if len(tools) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(tools))
	}

	names := map[string]bool{}
	for _, tool := range tools {
		names[tool.Name] = true
	}

	if !names["echo"] {
		t.Fatal("expected 'echo' tool")
	}
	if !names["add"] {
		t.Fatal("expected 'add' tool")
	}
	if !names["get_image"] {
		t.Fatal("expected 'get_image' tool")
	}
}

func TestNewClient_CallTool(t *testing.T) {
	url := startTestMCPServer(t)

	ctx := context.Background()
	c, err := newClient(ctx, "test", ServerConfig{Type: "sse", URL: url + "/sse"})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer func() { _ = c.close() }()

	result, err := c.callTool(ctx, "echo", map[string]any{"message": "hello"})
	if err != nil {
		t.Fatalf("failed to call echo tool: %v", err)
	}

	expected := "echo: hello"
	if result.Content != expected {
		t.Fatalf("expected %q, got %q", expected, result.Content)
	}
}

func TestNewClient_CallToolAdd(t *testing.T) {
	url := startTestMCPServer(t)

	ctx := context.Background()
	c, err := newClient(ctx, "test", ServerConfig{Type: "sse", URL: url + "/sse"})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer func() { _ = c.close() }()

	result, err := c.callTool(ctx, "add", map[string]any{"a": 3.0, "b": 7.0})
	if err != nil {
		t.Fatalf("failed to call add tool: %v", err)
	}

	if result.Content != "10.00" {
		t.Fatalf("expected '10.00', got %q", result.Content)
	}
}

func TestNewClient_CallToolImage(t *testing.T) {
	url := startTestMCPServer(t)

	ctx := context.Background()
	c, err := newClient(ctx, "test", ServerConfig{Type: "sse", URL: url + "/sse"})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer func() { _ = c.close() }()

	result, err := c.callTool(ctx, "get_image", map[string]any{})
	if err != nil {
		t.Fatalf("failed to call get_image tool: %v", err)
	}

	if result.Content != "Here is the panel screenshot" {
		t.Fatalf("expected text content, got %q", result.Content)
	}
	if len(result.Images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(result.Images))
	}
	if string(result.Images[0]) != "fake-png-data" {
		t.Fatalf("unexpected image data: %q", result.Images[0])
	}
}

func TestNewClient_UnsupportedTransport(t *testing.T) {
	ctx := context.Background()
	_, err := newClient(ctx, "test", ServerConfig{Type: "stdio", URL: ""})
	if err == nil {
		t.Fatal("expected error for unsupported transport")
	}
}

func TestManager_FullLifecycle(t *testing.T) {
	url := startTestMCPServer(t)

	cfgPath := writeTempMCPConfig(t, map[string]ServerConfig{
		"test-server": {Type: "sse", URL: url + "/sse"},
	})

	ctx := context.Background()
	mgr, err := NewManager(ctx, cfgPath)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	defer mgr.Close()

	tools := mgr.Tools()
	if len(tools) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(tools))
	}

	names := map[string]bool{}
	for _, tool := range tools {
		names[tool.Name] = true
	}
	if !names["test-server__echo"] {
		t.Fatal("expected 'test-server__echo' tool")
	}
	if !names["test-server__add"] {
		t.Fatal("expected 'test-server__add' tool")
	}
	if !names["test-server__get_image"] {
		t.Fatal("expected 'test-server__get_image' tool")
	}

	result, err := mgr.CallTool(ctx, "test-server__echo", json.RawMessage(`{"message": "from manager"}`))
	if err != nil {
		t.Fatalf("failed to call tool via manager: %v", err)
	}

	if result.Content != "echo: from manager" {
		t.Fatalf("expected 'echo: from manager', got %q", result.Content)
	}
}

func TestManager_UnknownTool(t *testing.T) {
	url := startTestMCPServer(t)

	cfgPath := writeTempMCPConfig(t, map[string]ServerConfig{
		"test-server": {Type: "sse", URL: url + "/sse"},
	})

	ctx := context.Background()
	mgr, err := NewManager(ctx, cfgPath)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	defer mgr.Close()

	_, err = mgr.CallTool(ctx, "nonexistent", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}

	// Non-namespaced original name must also fail.
	_, err = mgr.CallTool(ctx, "echo", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error for non-namespaced tool name")
	}
}

func TestManager_InvalidConfig(t *testing.T) {
	ctx := context.Background()
	_, err := NewManager(ctx, "/nonexistent/.mcp.json")
	if err == nil {
		t.Fatal("expected error for missing config")
	}
}

func TestManager_TwoServersNoCollision(t *testing.T) {
	url := startTestMCPServer(t)

	cfgPath := writeTempMCPConfig(t, map[string]ServerConfig{
		"alpha": {Type: "sse", URL: url + "/sse"},
		"beta":  {Type: "sse", URL: url + "/sse"},
	})

	ctx := context.Background()
	mgr, err := NewManager(ctx, cfgPath)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	defer mgr.Close()

	tools := mgr.Tools()
	if len(tools) != 6 {
		t.Fatalf("expected 6 tools (3 per server), got %d", len(tools))
	}

	names := mgr.ToolNames()
	for _, expected := range []string{
		"alpha__echo", "alpha__add", "alpha__get_image",
		"beta__echo", "beta__add", "beta__get_image",
	} {
		if _, ok := names[expected]; !ok {
			t.Errorf("expected tool %q to be registered", expected)
		}
	}

	// Both servers expose "echo", but namespacing keeps them distinct.
	resultA, err := mgr.CallTool(ctx, "alpha__echo", json.RawMessage(`{"message": "A"}`))
	if err != nil {
		t.Fatalf("alpha__echo failed: %v", err)
	}
	if resultA.Content != "echo: A" {
		t.Fatalf("expected 'echo: A', got %q", resultA.Content)
	}

	resultB, err := mgr.CallTool(ctx, "beta__echo", json.RawMessage(`{"message": "B"}`))
	if err != nil {
		t.Fatalf("beta__echo failed: %v", err)
	}
	if resultB.Content != "echo: B" {
		t.Fatalf("expected 'echo: B', got %q", resultB.Content)
	}
}

func TestManager_DeterministicOrder(t *testing.T) {
	url := startTestMCPServer(t)

	cfgPath := writeTempMCPConfig(t, map[string]ServerConfig{
		"zulu":  {Type: "sse", URL: url + "/sse"},
		"alpha": {Type: "sse", URL: url + "/sse"},
	})

	ctx := context.Background()
	mgr, err := NewManager(ctx, cfgPath)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	defer mgr.Close()

	tools := mgr.Tools()
	// Alpha tools should come before Zulu tools due to sorted iteration.
	if len(tools) < 4 {
		t.Fatalf("expected at least 4 tools, got %d", len(tools))
	}

	firstServer, _ := parseToolName(tools[0].Name)
	if firstServer != "alpha" {
		t.Fatalf("expected first tool to belong to 'alpha', got server %q (tool %q)", firstServer, tools[0].Name)
	}
}

func TestNamespacedToolName(t *testing.T) {
	got := namespacedToolName("grafana", "search_dashboards")
	if got != "grafana__search_dashboards" {
		t.Fatalf("expected 'grafana__search_dashboards', got %q", got)
	}
}

func TestParseToolName(t *testing.T) {
	tests := []struct {
		input      string
		wantServer string
		wantTool   string
	}{
		{"grafana__search_dashboards", "grafana", "search_dashboards"},
		{"alpha__beta__gamma", "alpha", "beta__gamma"},
		{"notool", "", "notool"},
	}
	for _, tt := range tests {
		server, tool := parseToolName(tt.input)
		if server != tt.wantServer || tool != tt.wantTool {
			t.Errorf("parseToolName(%q) = (%q, %q), want (%q, %q)",
				tt.input, server, tool, tt.wantServer, tt.wantTool)
		}
	}
}

func TestConvertTool(t *testing.T) {
	mcpTool := mcp.NewTool("test_tool",
		mcp.WithDescription("A test tool"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name param")),
		mcp.WithNumber("count", mcp.Description("Count param")),
	)

	tool := convertTool(mcpTool)

	if tool.Name != "test_tool" {
		t.Fatalf("expected name 'test_tool', got %q", tool.Name)
	}
	if tool.Description != "A test tool" {
		t.Fatalf("expected description 'A test tool', got %q", tool.Description)
	}

	params := tool.Parameters
	if params["type"] != "object" {
		t.Fatalf("expected type=object, got %v", params["type"])
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties to be map[string]any")
	}
	if _, ok := props["name"]; !ok {
		t.Fatal("expected 'name' in properties")
	}

	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("expected required to be []string")
	}
	if len(required) != 1 || required[0] != "name" {
		t.Fatalf("expected required=[name], got %v", required)
	}
}
