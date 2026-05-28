package mcp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadServersFile_Valid(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".mcp.json")

	content := `{
		"mcpServers": {
			"grafana": {
				"type": "sse",
				"url": "https://mcp.example.com/sse"
			},
			"another": {
				"type": "sse",
				"url": "http://localhost:9090/sse"
			}
		}
	}`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	sf, err := LoadServersFile(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sf.MCPServers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(sf.MCPServers))
	}

	grafana, ok := sf.MCPServers["grafana"]
	if !ok {
		t.Fatal("expected 'grafana' server")
	}
	if grafana.Type != "sse" {
		t.Fatalf("expected type=sse, got %q", grafana.Type)
	}
	if grafana.URL != "https://mcp.example.com/sse" {
		t.Fatalf("expected URL=https://mcp.example.com/sse, got %q", grafana.URL)
	}
}

func TestLoadServersFile_FileNotFound(t *testing.T) {
	_, err := LoadServersFile("/nonexistent/.mcp.json")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestLoadServersFile_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".mcp.json")

	if err := os.WriteFile(cfgPath, []byte(`{invalid`), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadServersFile(cfgPath)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoadServersFile_EmptyServers(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".mcp.json")

	if err := os.WriteFile(cfgPath, []byte(`{"mcpServers": {}}`), 0644); err != nil {
		t.Fatal(err)
	}

	sf, err := LoadServersFile(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sf.MCPServers) != 0 {
		t.Fatalf("expected 0 servers, got %d", len(sf.MCPServers))
	}
}
