package mcp

import (
	"os"
	"path/filepath"
	"strings"
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

func TestLoadServersFile_Headers(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".mcp.json")

	content := `{
		"mcpServers": {
			"grafana": {
				"type": "sse",
				"url": "https://mcp.example.com/sse",
				"headers": {
					"Authorization": "Bearer secret-token",
					"X-Custom": "value"
				}
			},
			"plain": {
				"type": "http",
				"url": "https://mcp.example.com/mcp"
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

	grafana := sf.MCPServers["grafana"]
	if got, want := len(grafana.Headers), 2; got != want {
		t.Fatalf("grafana headers count = %d, want %d", got, want)
	}
	if got, want := grafana.Headers["Authorization"], "Bearer secret-token"; got != want {
		t.Errorf("Authorization header = %q, want %q", got, want)
	}
	if got, want := grafana.Headers["X-Custom"], "value"; got != want {
		t.Errorf("X-Custom header = %q, want %q", got, want)
	}

	if plain := sf.MCPServers["plain"]; plain.Headers != nil {
		t.Errorf("plain headers = %v, want nil", plain.Headers)
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

func TestLoadServersFile_HeadersExpandEnv(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".mcp.json")

	content := `{
		"mcpServers": {
			"grafana": {
				"type": "sse",
				"url": "https://mcp.example.com/sse",
				"headers": {
					"Authorization": "Bearer ${GRAFANA_TOKEN}",
					"X-Plain": "no-refs-here"
				}
			}
		}
	}`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("GRAFANA_TOKEN", "glsa_secret")

	sf, err := LoadServersFile(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	headers := sf.MCPServers["grafana"].Headers
	if got, want := headers["Authorization"], "Bearer glsa_secret"; got != want {
		t.Errorf("Authorization header = %q, want %q", got, want)
	}
	if got, want := headers["X-Plain"], "no-refs-here"; got != want {
		t.Errorf("X-Plain header = %q, want %q", got, want)
	}
}

func TestLoadServersFile_HeadersMissingEnv(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".mcp.json")

	content := `{
		"mcpServers": {
			"grafana": {
				"type": "sse",
				"url": "https://mcp.example.com/sse",
				"headers": {"Authorization": "Bearer ${MCP_TOKEN_NOT_SET}"}
			}
		}
	}`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadServersFile(cfgPath)
	if err == nil {
		t.Fatal("expected error for unset environment variable")
	}
	if !strings.Contains(err.Error(), "MCP_TOKEN_NOT_SET") {
		t.Errorf("error %q should name the missing variable", err)
	}
}
