package web

import (
	"encoding/json"
	"net/http"

	"github.com/tsisar/extended-log-go/log"
)

func (h *Handler) listTools(w http.ResponseWriter, r *http.Request) {
	if h.tools == nil {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
		return
	}

	type toolInfo struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	tools := h.tools.Tools()
	result := make([]toolInfo, len(tools))
	for i, t := range tools {
		result[i] = toolInfo{Name: t.Name, Description: t.Description}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (h *Handler) grafanaAlerts(w http.ResponseWriter, r *http.Request) {
	if h.tools == nil {
		http.Error(w, "MCP tools not configured", http.StatusServiceUnavailable)
		return
	}

	args, _ := json.Marshal(map[string]any{
		"operation":    "list",
		"limit_alerts": 0,
	})

	result, err := h.tools.CallTool(r.Context(), "grafana__alerting_manage_rules", args)
	if err != nil {
		log.Errorf("grafana alerts: %v", err)
		http.Error(w, "failed to fetch alerts from Grafana", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	// The MCP tool returns JSON as text content — pass it through directly.
	_, _ = w.Write([]byte(result.Content))
}
