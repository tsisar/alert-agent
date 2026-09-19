package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type ServersFile struct {
	MCPServers map[string]ServerConfig `json:"mcpServers"`
}

type ServerConfig struct {
	Type string `json:"type"` // "sse" or "http" (Streamable HTTP)
	URL  string `json:"url"`
	// Headers are sent with every HTTP request to the server, e.g.
	// {"Authorization": "Bearer <token>"} for servers that require auth.
	// Values may reference environment variables as ${VAR}, so a token can be
	// injected from a secret instead of being stored in the config file.
	Headers map[string]string `json:"headers,omitempty"`
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

	for name, srv := range sf.MCPServers {
		for header, value := range srv.Headers {
			expanded, err := expandEnv(value)
			if err != nil {
				return nil, fmt.Errorf("MCP server %q header %q: %w", name, header, err)
			}
			srv.Headers[header] = expanded
		}
	}

	return &sf, nil
}

var envRefRe = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// expandEnv replaces every ${VAR} reference with the environment variable's
// value. An unset variable is an error rather than an empty string — a silently
// empty Authorization header would fail later with a confusing 401.
func expandEnv(value string) (string, error) {
	var missing []string

	expanded := envRefRe.ReplaceAllStringFunc(value, func(ref string) string {
		name := envRefRe.FindStringSubmatch(ref)[1]
		v, ok := os.LookupEnv(name)
		if !ok {
			missing = append(missing, name)
			return ""
		}
		return v
	})

	if len(missing) > 0 {
		return "", fmt.Errorf("environment variable %s is not set", strings.Join(missing, ", "))
	}

	return expanded, nil
}
