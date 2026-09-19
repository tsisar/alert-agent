package web

import (
	"net/http"
	"strings"

	"github.com/tsisar/alert-agent/internal/web/templates"
	"github.com/tsisar/extended-log-go/log"
)

var promptKeys = map[string]bool{
	"system":   true,
	"summary":  true,
	"resolved": true,
	"paused":   true,
}

func (h *Handler) listPrompts(w http.ResponseWriter, r *http.Request) {
	ps, err := h.prompts.GetAll()
	if err != nil {
		http.Error(w, "failed to load prompts", http.StatusInternalServerError)
		log.Errorf("list prompts: %v", err)
		return
	}
	if err := templates.PromptsPage(ps).Render(r.Context(), w); err != nil {
		log.Errorf("render prompts page: %v", err)
	}
}

func (h *Handler) updatePrompt(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if !promptKeys[key] {
		http.Error(w, "unknown prompt key", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "failed to parse form", http.StatusBadRequest)
		return
	}

	value := strings.TrimSpace(r.FormValue("value"))
	if err := h.prompts.Set(key, value); err != nil {
		http.Error(w, "failed to save prompt", http.StatusInternalServerError)
		log.Errorf("update prompt %s: %v", key, err)
		return
	}

	if err := templates.PromptStageSaved(key, value).Render(r.Context(), w); err != nil {
		log.Errorf("render prompt stage: %v", err)
	}
}
