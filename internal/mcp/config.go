package mcp

import (
	"encoding/json"
	"fmt"
	"os"
)

type ServersFile struct {
	MCPServers map[string]ServerConfig `json:"mcpServers"`
}

type ServerConfig struct {
	Type string `json:"type"` // "sse"
	URL  string `json:"url"`
}

func LoadServersFile(path string) (*ServersFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read MCP config %q: %w", path, err)
	}

	var sf ServersFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil, fmt.Errorf("parse MCP config %q: %w", path, err)
	}

	return &sf, nil
}
