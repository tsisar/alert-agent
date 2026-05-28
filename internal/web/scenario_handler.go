package web

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tsisar/alert-agent/internal/storage"
	"github.com/tsisar/alert-agent/internal/web/templates"
	"github.com/tsisar/extended-log-go/log"
	"gopkg.in/yaml.v3"
)

const errInvalidID = "invalid id"

func (h *Handler) listScenarios(w http.ResponseWriter, r *http.Request) {
	scenarios, err := h.scenarios.List()
	if err != nil {
		http.Error(w, "failed to load scenarios", http.StatusInternalServerError)
		log.Errorf("list scenarios: %v", err)
		return
	}
	if err := templates.ScenariosPage(scenarios).Render(r.Context(), w); err != nil {
		log.Errorf("render scenarios page: %v", err)
	}
}

func (h *Handler) newScenario(w http.ResponseWriter, r *http.Request) {
	s := &storage.Scenario{
		Timeout:  "2m",
		Priority: "normal",
	}
	if err := templates.ScenarioFormPage(s, false).Render(r.Context(), w); err != nil {
		log.Errorf("render new scenario page: %v", err)
	}
}

func (h *Handler) createScenario(w http.ResponseWriter, r *http.Request) {
	s, err := parseScenarioForm(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.scenarios.Create(s); err != nil {
		http.Error(w, "failed to create scenario", http.StatusInternalServerError)
		log.Errorf("create scenario: %v", err)
		return
	}
	h.listScenarios(w, r)
}

func (h *Handler) editScenario(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, errInvalidID, http.StatusBadRequest)
		return
	}
	scenarios, err := h.scenarios.List()
	if err != nil {
		http.Error(w, "failed to load scenarios", http.StatusInternalServerError)
		log.Errorf("list scenarios for edit: %v", err)
		return
	}
	var target *storage.Scenario
	for i := range scenarios {
		if scenarios[i].ID == uint(id) {
			target = &scenarios[i]
			break
		}
	}
	if target == nil {
		http.Error(w, "scenario not found", http.StatusNotFound)
		return
	}
	if err := templates.ScenarioFormPage(target, true).Render(r.Context(), w); err != nil {
		log.Errorf("render edit scenario page: %v", err)
	}
}

func (h *Handler) copyScenario(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, errInvalidID, http.StatusBadRequest)
		return
	}
	scenarios, err := h.scenarios.List()
	if err != nil {
		http.Error(w, "failed to load scenarios", http.StatusInternalServerError)
		log.Errorf("list scenarios for copy: %v", err)
		return
	}
	var source *storage.Scenario
	for i := range scenarios {
		if scenarios[i].ID == uint(id) {
			source = &scenarios[i]
			break
		}
	}
	if source == nil {
		http.Error(w, "scenario not found", http.StatusNotFound)
		return
	}

	dup := *source
	dup.ID = 0
	dup.Name = copyName(source.Name, scenarios)
	dup.OrderIndex = 0

	if err := templates.ScenarioFormPage(&dup, false).Render(r.Context(), w); err != nil {
		log.Errorf("render copy scenario page: %v", err)
	}
}

func copyName(base string, existing []storage.Scenario) string {
	taken := make(map[string]bool, len(existing))
	for _, s := range existing {
		taken[s.Name] = true
	}
	name := base + " (copy)"
	for i := 2; taken[name]; i++ {
		name = fmt.Sprintf("%s (copy %d)", base, i)
	}
	return name
}

func (h *Handler) updateScenario(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, errInvalidID, http.StatusBadRequest)
		return
	}
	s, err := parseScenarioForm(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.ID = uint(id)
	if err := h.scenarios.Update(s); err != nil {
		http.Error(w, "failed to update scenario", http.StatusInternalServerError)
		log.Errorf("update scenario: %v", err)
		return
	}
	h.listScenarios(w, r)
}

func (h *Handler) deleteScenario(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, errInvalidID, http.StatusBadRequest)
		return
	}
	if err := h.scenarios.Delete(uint(id)); err != nil {
		http.Error(w, "failed to delete scenario", http.StatusInternalServerError)
		log.Errorf("delete scenario: %v", err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

type exportScenario struct {
	Name     string            `yaml:"name"`
	Match    map[string]string `yaml:"match"`
	Prompt   string            `yaml:"prompt"`
	Tools    []string          `yaml:"tools,omitempty"`
	Channels *exportChannels   `yaml:"channels,omitempty"`
	Timeout  string            `yaml:"timeout"`
	Priority string            `yaml:"priority"`
	// nolint:tagliatelle
	SendImages bool `yaml:"send_images,omitempty"`
}

type exportChannels struct {
	Telegram string `yaml:"telegram,omitempty"`
	Slack    string `yaml:"slack,omitempty"`
}

type exportFile struct {
	Scenarios []exportScenario `yaml:"scenarios"`
}

func (h *Handler) exportScenarios(w http.ResponseWriter, r *http.Request) {
	scenarios, err := h.scenarios.List()
	if err != nil {
		http.Error(w, "failed to load scenarios", http.StatusInternalServerError)
		log.Errorf("export scenarios: %v", err)
		return
	}

	out := exportFile{Scenarios: make([]exportScenario, len(scenarios))}
	for i, s := range scenarios {
		es := exportScenario{
			Name:       s.Name,
			Match:      map[string]string(s.Match),
			Prompt:     s.Prompt,
			Timeout:    s.Timeout,
			Priority:   s.Priority,
			SendImages: s.SendImages,
		}
		if len(s.Tools) > 0 {
			es.Tools = []string(s.Tools)
		}
		if s.ChannelTelegram != "" || s.ChannelSlack != "" {
			es.Channels = &exportChannels{
				Telegram: s.ChannelTelegram,
				Slack:    s.ChannelSlack,
			}
		}
		out.Scenarios[i] = es
	}

	data, err := yaml.Marshal(out)
	if err != nil {
		http.Error(w, "failed to marshal yaml", http.StatusInternalServerError)
		log.Errorf("export scenarios marshal: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/x-yaml")
	w.Header().Set("Content-Disposition", "attachment; filename=scenarios.yaml")
	_, _ = w.Write(data)
}

const maxImportSize = 1 << 20 // 1 MB

func (h *Handler) importScenarios(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxImportSize)

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing or invalid file", http.StatusBadRequest)
		return
	}
	defer func() { _ = file.Close() }()

	var ef exportFile
	if err := yaml.NewDecoder(file).Decode(&ef); err != nil {
		http.Error(w, "invalid YAML: "+err.Error(), http.StatusBadRequest)
		return
	}

	var created, updated int
	for _, es := range ef.Scenarios {
		if es.Name == "" {
			continue
		}

		s := &storage.Scenario{
			Name:       es.Name,
			Match:      storage.JSONMap(es.Match),
			Prompt:     es.Prompt,
			Tools:      storage.JSONStringSlice(es.Tools),
			Timeout:    es.Timeout,
			Priority:   es.Priority,
			SendImages: es.SendImages,
		}
		if s.Timeout == "" {
			s.Timeout = "2m"
		}
		if s.Priority == "" {
			s.Priority = "normal"
		}
		if es.Channels != nil {
			s.ChannelTelegram = es.Channels.Telegram
			s.ChannelSlack = es.Channels.Slack
		}

		existing := h.scenarios.FindByName(es.Name)
		if existing != nil {
			// Find the DB model to get the ID.
			scenarios, _ := h.scenarios.List()
			for _, dbS := range scenarios {
				if dbS.Name == es.Name {
					s.ID = dbS.ID
					break
				}
			}
			if err := h.scenarios.Update(s); err != nil {
				log.Errorf("import update scenario %q: %v", es.Name, err)
				continue
			}
			updated++
		} else {
			if err := h.scenarios.Create(s); err != nil {
				log.Errorf("import create scenario %q: %v", es.Name, err)
				continue
			}
			created++
		}
	}

	log.Infof("scenarios imported: created=%d updated=%d", created, updated)
	h.listScenarios(w, r)
}

func parseID(r *http.Request) (uint64, error) {
	return strconv.ParseUint(r.PathValue("id"), 10, 64)
}

func parseScenarioForm(r *http.Request) (*storage.Scenario, error) {
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("parse form: %w", err)
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	orderIndex, _ := strconv.Atoi(r.FormValue("order_index"))

	match, err := parseMatchLabels(r.FormValue("match"))
	if err != nil {
		return nil, err
	}

	prompt := strings.TrimSpace(r.FormValue("prompt"))
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	tools := parseToolsList(r.FormValue("tools"))

	timeout := strings.TrimSpace(r.FormValue("timeout"))
	if timeout == "" {
		timeout = "2m"
	}
	if _, err := time.ParseDuration(timeout); err != nil {
		return nil, fmt.Errorf("invalid timeout %q: %w", timeout, err)
	}

	priority := r.FormValue("priority")
	if priority == "" {
		priority = "normal"
	}

	return &storage.Scenario{
		Name:            name,
		OrderIndex:      orderIndex,
		Match:           match,
		Prompt:          prompt,
		Tools:           tools,
		ChannelTelegram: strings.TrimSpace(r.FormValue("channel_telegram")),
		ChannelSlack:    strings.TrimSpace(r.FormValue("channel_slack")),
		Timeout:         timeout,
		Priority:        priority,
		SendImages:      r.FormValue("send_images") == "on",
	}, nil
}

func parseMatchLabels(raw string) (storage.JSONMap, error) {
	raw = strings.TrimSpace(raw)
	m := make(storage.JSONMap)
	if raw == "" {
		return m, nil
	}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("invalid match line %q: expected key=value", line)
		}
		m[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return m, nil
}

func parseToolsList(raw string) storage.JSONStringSlice {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var tools storage.JSONStringSlice
	for _, t := range strings.Split(raw, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			tools = append(tools, t)
		}
	}
	return tools
}
