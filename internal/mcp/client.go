package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/tsisar/alert-agent/internal/llm"
	"github.com/tsisar/extended-log-go/log"
)

const toolNameSep = "__"

func namespacedToolName(serverName, toolName string) string {
	return serverName + toolNameSep + toolName
}

func parseToolName(namespaced string) (serverName, toolName string) {
	if i := strings.Index(namespaced, toolNameSep); i >= 0 {
		return namespaced[:i], namespaced[i+len(toolNameSep):]
	}
	return "", namespaced
}

// Client wraps a single MCP server connection with automatic reconnect.
type Client struct {
	mu     sync.Mutex
	name   string
	cfg    ServerConfig
	client *mcpclient.Client
}

func newClient(ctx context.Context, name string, cfg ServerConfig) (*Client, error) {
	if cfg.Type != "sse" {
		return nil, fmt.Errorf("unsupported MCP transport %q for server %q (only sse is supported)", cfg.Type, name)
	}

	cl := &Client{name: name, cfg: cfg}
	if err := cl.connect(ctx); err != nil {
		return nil, err
	}
	return cl, nil
}

func (c *Client) connect(ctx context.Context) error {
	sseClient, err := mcpclient.NewSSEMCPClient(c.cfg.URL)
	if err != nil {
		return fmt.Errorf("create MCP SSE client for %q: %w", c.name, err)
	}

	if err := sseClient.Start(ctx); err != nil {
		return fmt.Errorf("start MCP client %q: %w", c.name, err)
	}

	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    "alert-agent",
		Version: "0.1.0",
	}

	if _, err := sseClient.Initialize(ctx, initReq); err != nil {
		return fmt.Errorf("initialize MCP session %q: %w", c.name, err)
	}

	c.client = sseClient
	return nil
}

func (c *Client) reconnect(ctx context.Context) error {
	log.Warnf("[mcp] reconnecting to server %q", c.name)
	if c.client != nil {
		_ = c.client.Close()
	}
	if err := c.connect(ctx); err != nil {
		return fmt.Errorf("reconnect to MCP server %q: %w", c.name, err)
	}
	log.Infof("[mcp] reconnected to server %q", c.name)
	return nil
}

// isSessionError checks if the error indicates a stale/expired MCP session.
func isSessionError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Could not find session") ||
		strings.Contains(msg, "status 404") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "EOF")
}

func (c *Client) listTools(ctx context.Context) ([]llm.Tool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	tools, err := c.doListTools(ctx)
	if err != nil && isSessionError(err) {
		if reconnErr := c.reconnect(ctx); reconnErr != nil {
			return nil, fmt.Errorf("list tools from MCP server %q: %w (reconnect also failed: %v)", c.name, err, reconnErr)
		}
		tools, err = c.doListTools(ctx)
	}
	return tools, err
}

func (c *Client) doListTools(ctx context.Context) ([]llm.Tool, error) {
	result, err := c.client.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, fmt.Errorf("list tools from MCP server %q: %w", c.name, err)
	}

	tools := make([]llm.Tool, 0, len(result.Tools))
	for _, t := range result.Tools {
		tools = append(tools, convertTool(t))
	}

	return tools, nil
}

func (c *Client) callTool(ctx context.Context, name string, args map[string]any) (*llm.ToolResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	result, err := c.doCallTool(ctx, name, args)
	if err != nil && isSessionError(err) {
		if reconnErr := c.reconnect(ctx); reconnErr != nil {
			return nil, fmt.Errorf("call MCP tool %q on server %q: %w (reconnect also failed: %v)", name, c.name, err, reconnErr)
		}
		result, err = c.doCallTool(ctx, name, args)
	}
	return result, err
}

func (c *Client) doCallTool(ctx context.Context, name string, args map[string]any) (*llm.ToolResult, error) {
	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args

	result, err := c.client.CallTool(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("call MCP tool %q on server %q: %w", name, c.name, err)
	}

	if result.IsError {
		return nil, fmt.Errorf("MCP tool %q returned error: %s", name, extractText(result))
	}

	return extractResult(result)
}

func (c *Client) close() error {
	return c.client.Close()
}

// Manager holds connections to all configured MCP servers and dispatches tool calls.
type Manager struct {
	clients  []*Client
	toolMap  map[string]*Client // tool name -> owning client
	toolList []llm.Tool
}

// NewManager creates MCP clients for all servers defined in the config file
// and pre-fetches their tool lists. Tool names are prefixed with the server
// name (e.g. "grafana__search_dashboards") to prevent collisions.
func NewManager(ctx context.Context, configPath string) (*Manager, error) {
	sf, err := LoadServersFile(configPath)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(sf.MCPServers))
	for name := range sf.MCPServers {
		names = append(names, name)
	}
	sort.Strings(names)

	m := &Manager{
		toolMap: make(map[string]*Client),
	}

	for _, name := range names {
		cfg := sf.MCPServers[name]
		c, err := newClient(ctx, name, cfg)
		if err != nil {
			m.Close()
			return nil, err
		}
		m.clients = append(m.clients, c)

		tools, err := c.listTools(ctx)
		if err != nil {
			m.Close()
			return nil, err
		}

		for _, t := range tools {
			nsName := namespacedToolName(name, t.Name)
			log.Debugf("[mcp] tool registered: %s -> %s (server %q)", t.Name, nsName, name)
			t.Name = nsName
			m.toolMap[nsName] = c
			m.toolList = append(m.toolList, t)
		}

		log.Infof("MCP server %q: %d tools available", name, len(tools))
	}

	return m, nil
}

// Tools returns the combined list of tools from all MCP servers.
func (m *Manager) Tools() []llm.Tool {
	return m.toolList
}

// ToolNames returns a set of all registered (namespaced) tool names.
func (m *Manager) ToolNames() map[string]struct{} {
	names := make(map[string]struct{}, len(m.toolList))
	for _, t := range m.toolList {
		names[t.Name] = struct{}{}
	}
	return names
}

// CallTool dispatches a tool call to the MCP server that owns the tool.
// The namespaced name is used for routing; the original tool name is
// forwarded to the MCP server.
func (m *Manager) CallTool(ctx context.Context, name string, arguments json.RawMessage) (*llm.ToolResult, error) {
	c, ok := m.toolMap[name]
	if !ok {
		return nil, fmt.Errorf("unknown MCP tool %q", name)
	}

	var args map[string]any
	if len(arguments) > 0 {
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("unmarshal tool arguments: %w", err)
		}
	}

	_, originalName := parseToolName(name)
	return c.callTool(ctx, originalName, args)
}

func (m *Manager) Close() {
	for _, c := range m.clients {
		if err := c.close(); err != nil {
			log.Errorf("failed to close MCP client %q: %v", c.name, err)
		}
	}
}

func convertTool(t mcp.Tool) llm.Tool {
	params := make(map[string]any)
	if t.InputSchema.Properties != nil {
		params["type"] = "object"
		params["properties"] = t.InputSchema.Properties
		if len(t.InputSchema.Required) > 0 {
			params["required"] = t.InputSchema.Required
		}
	}

	return llm.Tool{
		Name:        t.Name,
		Description: t.Description,
		Parameters:  params,
	}
}

func extractText(result *mcp.CallToolResult) string {
	var text string
	for _, c := range result.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			if text != "" {
				text += "\n"
			}
			text += tc.Text
		}
	}
	return text
}

func extractResult(result *mcp.CallToolResult) (*llm.ToolResult, error) {
	tr := &llm.ToolResult{}
	for _, c := range result.Content {
		switch v := c.(type) {
		case mcp.TextContent:
			if tr.Content != "" {
				tr.Content += "\n"
			}
			tr.Content += v.Text
		case mcp.ImageContent:
			decoded, err := base64.StdEncoding.DecodeString(v.Data)
			if err != nil {
				return nil, fmt.Errorf("decode image from MCP result: %w", err)
			}
			tr.Images = append(tr.Images, decoded)
		}
	}
	return tr, nil
}
