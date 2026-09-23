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
	if err := templates.ScenarioFormPage(s, false, "").Render(r.Context(), w); err != nil {
		log.Errorf("render new scenario page: %v", err)
	}
}

func (h *Handler) createScenario(w http.ResponseWriter, r *http.Request) {
	s, err := parseScenarioForm(r)
	if err != nil {
		h.renderFormError(w, r, submittedScenario(r), false, err.Error())
		return
	}
	if err := h.scenarios.Create(s); err != nil {
		log.Errorf("create scenario: %v", err)
		h.renderFormError(w, r, s, false, saveErrorMessage(err, s.Name))
		return
	}
	h.listScenarios(w, r)
}

// renderFormError re-renders the editor with the submitted values and a stated
// reason. htmx does not swap a plain error response, so returning one here
// would look to the user like the save silently did nothing.
func (h *Handler) renderFormError(w http.ResponseWriter, r *http.Request, s *storage.Scenario, isEdit bool, msg string) {
	w.Header().Set("HX-Retarget", "body")
	w.Header().Set("HX-Reswap", "innerHTML")
	// The form's hx-push-url points at the list; a rejected save must not
	// leave the editor sitting under the /scenarios URL.
	w.Header().Set("HX-Push-Url", "false")
	w.WriteHeader(http.StatusOK)
	if err := templates.ScenarioFormPage(s, isEdit, msg).Render(r.Context(), w); err != nil {
		log.Errorf("render scenario form error: %v", err)
	}
}

// submittedScenario rebuilds a scenario from the raw form so a rejected save
// keeps every field the user typed, including the invalid one.
func submittedScenario(r *http.Request) *storage.Scenario {
	match, _ := parseMatchLabels(r.FormValue("match"))
	orderIndex, _ := strconv.Atoi(r.FormValue("order_index"))
	return &storage.Scenario{
		Name:            strings.TrimSpace(r.FormValue("name")),
		OrderIndex:      orderIndex,
		Match:           match,
		Prompt:          r.FormValue("prompt"),
		Tools:           parseToolsList(r.FormValue("tools")),
		ChannelTelegram: strings.TrimSpace(r.FormValue("channel_telegram")),
		ChannelSlack:    strings.TrimSpace(r.FormValue("channel_slack")),
		Timeout:         strings.TrimSpace(r.FormValue("timeout")),
		Priority:        r.FormValue("priority"),
		SendImages:      r.FormValue("send_images") == "on",
	}
}

// saveErrorMessage names the problem and the way out, rather than echoing a
// driver error the user cannot act on.
func saveErrorMessage(err error, name string) string {
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "unique") || strings.Contains(msg, "duplicate") {
		return fmt.Sprintf("A scenario named %q already exists. Pick another name, or edit the existing one.", name)
	}
	return "The scenario could not be saved. The server log has the details."
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
	if err := templates.ScenarioFormPage(target, true, "").Render(r.Context(), w); err != nil {
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

	if err := templates.ScenarioFormPage(&dup, false, "").Render(r.Context(), w); err != nil {
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
		bad := submittedScenario(r)
		bad.ID = uint(id)
		h.renderFormError(w, r, bad, true, err.Error())
		return
	}
	s.ID = uint(id)
	if err := h.scenarios.Update(s); err != nil {
		log.Errorf("update scenario: %v", err)
		h.renderFormError(w, r, s, true, saveErrorMessage(err, s.Name))
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
	// The list re-renders so positions renumber and the fallback group
	// reflects the deletion.
	h.listScenarios(w, r)
}

// moveScenario swaps a label-matching scenario with its neighbour in
// evaluation order, then rewrites order_index for every scenario so the new
// order is explicit (no ties left to the database).
func (h *Handler) moveScenario(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, errInvalidID, http.StatusBadRequest)
		return
	}
	dir := r.URL.Query().Get("dir")
	if dir != "up" && dir != "down" {
		http.Error(w, "dir must be up or down", http.StatusBadRequest)
		return
	}
	scenarios, err := h.scenarios.List()
	if err != nil {
		http.Error(w, "failed to load scenarios", http.StatusInternalServerError)
		log.Errorf("list scenarios for move: %v", err)
		return
	}
	for _, s := range reorderScenarios(scenarios, uint(id), dir == "up") {
		if err := h.scenarios.Update(&s); err != nil {
			http.Error(w, "failed to reorder scenarios", http.StatusInternalServerError)
			log.Errorf("reorder scenario %q: %v", s.Name, err)
			return
		}
	}
	h.listScenarios(w, r)
}

// reorderScenarios moves scenario id one step up or down among the
// label-matching scenarios and returns every scenario whose order_index must
// change. Catch-alls are only consulted when nothing else matches, so their
// position is irrelevant; they are kept after the ordered ones.
func reorderScenarios(scenarios []storage.Scenario, id uint, up bool) []storage.Scenario {
	var ordered, fallback []storage.Scenario
	for _, s := range scenarios {
		if len(s.Match) == 0 {
			fallback = append(fallback, s)
		} else {
			ordered = append(ordered, s)
		}
	}
	at := -1
	for i, s := range ordered {
		if s.ID == id {
			at = i
			break
		}
	}
	to := at + 1
	if up {
		to = at - 1
	}
	if at < 0 || to < 0 || to >= len(ordered) {
		return nil
	}
	ordered[at], ordered[to] = ordered[to], ordered[at]

	var changed []storage.Scenario
	for i, s := range append(ordered, fallback...) {
		want := (i + 1) * 10
		if s.OrderIndex != want {
			s.OrderIndex = want
			changed = append(changed, s)
		}
	}
	return changed
}

type exportScenario struct {
	Name string `yaml:"name"`
	// nolint:tagliatelle
	OrderIndex int               `yaml:"order_index"`
	Match      map[string]string `yaml:"match"`
	Prompt     string            `yaml:"prompt"`
	Tools      []string          `yaml:"tools,omitempty"`
	Channels   *exportChannels   `yaml:"channels,omitempty"`
	Timeout    string            `yaml:"timeout"`
	Priority   string            `yaml:"priority"`
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
			OrderIndex: s.OrderIndex,
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
			OrderIndex: es.OrderIndex,
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

// validationError is a message for the person filling in the form, rendered
// verbatim in the editor. It is written as a full sentence, so it is not a Go
// error string built with fmt.Errorf.
type validationError string

func (e validationError) Error() string { return string(e) }

func parseID(r *http.Request) (uint64, error) {
	return strconv.ParseUint(r.PathValue("id"), 10, 64)
}

func parseScenarioForm(r *http.Request) (*storage.Scenario, error) {
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("parse form: %w", err)
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		return nil, validationError("Give the scenario a name — it identifies the path in logs, exports and notifications.")
	}

	orderIndex, _ := strconv.Atoi(r.FormValue("order_index"))

	match, err := parseMatchLabels(r.FormValue("match"))
	if err != nil {
		return nil, err
	}

	prompt := strings.TrimSpace(r.FormValue("prompt"))
	if prompt == "" {
		return nil, validationError("The prompt is what the agent is told to do for these alerts; it cannot be empty.")
	}

	tools := parseToolsList(r.FormValue("tools"))

	timeout := strings.TrimSpace(r.FormValue("timeout"))
	if timeout == "" {
		timeout = "2m"
	}
	if _, err := time.ParseDuration(timeout); err != nil {
		return nil, validationError(fmt.Sprintf("%q is not a duration. Write it like 30s, 2m or 1h30m.", timeout))
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
			return nil, validationError(fmt.Sprintf("%q is not a label. Each match label is written as key=value.", line))
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
