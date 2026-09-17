package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/tsisar/extended-log-go/log"
)

type probeRequest struct {
	Labels string `json:"labels"`
}

type probeResponse struct {
	Matched   bool   `json:"matched"`
	ID        uint   `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Reason    string `json:"reason,omitempty"`
	MatchedOn string `json:"matchedOn,omitempty"`
	Error     string `json:"error,omitempty"`
}

// matchProbe answers "which scenario would catch an alert with these labels?".
// It calls the same repository method the webhook path uses, so the answer
// cannot drift from the agent's real behaviour.
func (h *Handler) matchProbe(w http.ResponseWriter, r *http.Request) {
	var req probeRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&req); err != nil {
		writeProbe(w, probeResponse{Error: "Could not read the labels to test."})
		return
	}

	labels, err := parseProbeLabels(req.Labels)
	if err != nil {
		writeProbe(w, probeResponse{Error: err.Error()})
		return
	}
	if len(labels) == 0 {
		writeProbe(w, probeResponse{Error: "Enter at least one key=value label."})
		return
	}

	match := h.scenarios.Match(labels)
	if match == nil {
		writeProbe(w, probeResponse{Matched: false})
		return
	}

	resp := probeResponse{Matched: true, Name: match.Name, Reason: "labels"}
	if len(match.Match) == 0 {
		resp.Reason = "catch-all"
	} else {
		keys := make([]string, 0, len(match.Match))
		for k, v := range match.Match {
			keys = append(keys, k+"="+v)
		}
		sort.Strings(keys)
		resp.MatchedOn = strings.Join(keys, ", ")
	}

	// The domain scenario carries no ID; resolve it by name so the UI can
	// highlight the row the user is looking at.
	if scenarios, err := h.scenarios.List(); err == nil {
		for _, s := range scenarios {
			if s.Name == match.Name {
				resp.ID = s.ID
				break
			}
		}
	} else {
		log.Errorf("match probe: list scenarios: %v", err)
	}

	writeProbe(w, resp)
}

// parseProbeLabels accepts the label syntax people actually paste: one
// key=value per line, or several separated by commas.
func parseProbeLabels(raw string) (map[string]string, error) {
	labels := make(map[string]string)
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ','
	})
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		k, v, ok := strings.Cut(field, "=")
		if !ok {
			return nil, fmt.Errorf("%q is not a label — write it as key=value", field)
		}
		labels[strings.TrimSpace(k)] = strings.TrimSpace(strings.Trim(v, `"`))
	}
	return labels, nil
}

func writeProbe(w http.ResponseWriter, resp probeResponse) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
