package web

import (
	"net/http"
	"strings"

	"github.com/tsisar/alert-agent/internal/web/templates"
	"github.com/tsisar/extended-log-go/log"
)

var promptLabels = map[string]string{
	"system":   "System Prompt",
	"summary":  "Summary Prompt",
	"resolved": "Resolved Prompt",
	"paused":   "Paused Prompt",
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
	label, ok := promptLabels[key]
	if !ok {
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

	if err := templates.PromptCardSaved(key, label, value).Render(r.Context(), w); err != nil {
		log.Errorf("render prompt card: %v", err)
	}
}
