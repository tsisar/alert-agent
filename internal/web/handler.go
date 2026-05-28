package web

import (
	"context"
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"

	"github.com/tsisar/alert-agent/internal/llm"
	"github.com/tsisar/alert-agent/internal/storage"
)

//go:embed static
var staticFS embed.FS

// ToolCaller can invoke MCP tools by name and list available tools.
type ToolCaller interface {
	Tools() []llm.Tool
	CallTool(ctx context.Context, name string, arguments json.RawMessage) (*llm.ToolResult, error)
}

// Handler serves the web UI for managing scenarios and prompts.
type Handler struct {
	scenarios storage.ScenarioRepository
	prompts   storage.PromptRepository
	tools     ToolCaller
}

// NewHandler creates a web UI handler.
func NewHandler(scenarios storage.ScenarioRepository, prompts storage.PromptRepository, tools ToolCaller) *Handler {
	return &Handler{
		scenarios: scenarios,
		prompts:   prompts,
		tools:     tools,
	}
}

// RegisterRoutes mounts all web UI routes on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	staticSub, _ := fs.Sub(staticFS, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticSub))))

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/scenarios", http.StatusFound)
	})

	mux.HandleFunc("GET /scenarios", h.listScenarios)
	mux.HandleFunc("GET /scenarios/export.yaml", h.exportScenarios)
	mux.HandleFunc("POST /scenarios/import", h.importScenarios)
	mux.HandleFunc("GET /scenarios/new", h.newScenario)
	mux.HandleFunc("POST /scenarios", h.createScenario)
	mux.HandleFunc("GET /scenarios/{id}/edit", h.editScenario)
	mux.HandleFunc("PUT /scenarios/{id}", h.updateScenario)
	mux.HandleFunc("DELETE /scenarios/{id}", h.deleteScenario)

	mux.HandleFunc("GET /prompts", h.listPrompts)
	mux.HandleFunc("PUT /prompts/{key}", h.updatePrompt)

	mux.HandleFunc("GET /api/grafana/alerts", h.grafanaAlerts)
	mux.HandleFunc("GET /api/tools", h.listTools)
}
