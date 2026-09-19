package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tsisar/alert-agent/internal/scenario"
	"github.com/tsisar/alert-agent/internal/storage"
	"gorm.io/gorm"
)

func gormModel(id uint) gorm.Model { return gorm.Model{ID: id} }

func TestParseProbeLabels(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    map[string]string
		wantErr bool
	}{
		{
			name: "one per line",
			raw:  "alertname=HighCPU\nseverity=critical",
			want: map[string]string{"alertname": "HighCPU", "severity": "critical"},
		},
		{
			name: "comma separated",
			raw:  "alertname=HighCPU, severity=critical",
			want: map[string]string{"alertname": "HighCPU", "severity": "critical"},
		},
		{
			name: "quoted values are unwrapped",
			raw:  `alertname="High CPU"`,
			want: map[string]string{"alertname": "High CPU"},
		},
		{
			name: "value may contain equals",
			raw:  "query=up==0",
			want: map[string]string{"query": "up==0"},
		},
		{
			name: "blank input yields no labels",
			raw:  "  \n , \n",
			want: map[string]string{},
		},
		{
			name:    "a bare word is rejected",
			raw:     "garbage",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseProbeLabels(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("label %q = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

// stubRepo is the smallest ScenarioRepository that exercises the probe.
type stubRepo struct {
	models []storage.Scenario
}

func (r *stubRepo) List() ([]storage.Scenario, error) { return r.models, nil }

// Match mirrors the production rule: ordered scan, first label match wins,
// otherwise the catch-all.
func (r *stubRepo) Match(labels map[string]string) *scenario.Scenario {
	var catchAll *scenario.Scenario
	for i := range r.models {
		d := r.models[i].ToDomain()
		if len(d.Match) == 0 {
			catchAll = &d
			continue
		}
		if scenario.LabelsMatch(d.Match, labels) {
			return &d
		}
	}
	return catchAll
}

func (r *stubRepo) FindByName(string) *scenario.Scenario { return nil }
func (r *stubRepo) Create(*storage.Scenario) error       { return nil }
func (r *stubRepo) Update(*storage.Scenario) error       { return nil }
func (r *stubRepo) Delete(uint) error                    { return nil }
func (r *stubRepo) Invalidate()                          {}

func probeHandler() *Handler {
	return NewHandler(&stubRepo{models: []storage.Scenario{
		{
			Model:   gormModel(7),
			Name:    "cpu",
			Match:   storage.JSONMap{"alertname": "HighCPU"},
			Timeout: "2m",
		},
		{
			Model:   gormModel(9),
			Name:    "everything-else",
			Match:   storage.JSONMap{},
			Timeout: "2m",
		},
	}}, nil, nil)
}

func postProbe(t *testing.T, h *Handler, body string) probeResponse {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/match-probe", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.matchProbe(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp probeResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}

func TestMatchProbe_LabelMatchWins(t *testing.T) {
	resp := postProbe(t, probeHandler(), `{"labels":"alertname=HighCPU, severity=critical"}`)

	if !resp.Matched {
		t.Fatal("expected a match")
	}
	if resp.Name != "cpu" {
		t.Errorf("name = %q, want \"cpu\"", resp.Name)
	}
	if resp.ID != 7 {
		t.Errorf("id = %d, want 7 (so the UI can highlight the row)", resp.ID)
	}
	if resp.Reason != "labels" {
		t.Errorf("reason = %q, want \"labels\"", resp.Reason)
	}
	if resp.MatchedOn != "alertname=HighCPU" {
		t.Errorf("matchedOn = %q, want \"alertname=HighCPU\"", resp.MatchedOn)
	}
}

func TestMatchProbe_FallsBackToCatchAll(t *testing.T) {
	resp := postProbe(t, probeHandler(), `{"labels":"alertname=Unknown"}`)

	if !resp.Matched || resp.Name != "everything-else" {
		t.Fatalf("expected the catch-all, got %+v", resp)
	}
	if resp.Reason != "catch-all" {
		t.Errorf("reason = %q, want \"catch-all\"", resp.Reason)
	}
}

func TestMatchProbe_NothingCatchesIt(t *testing.T) {
	h := NewHandler(&stubRepo{models: []storage.Scenario{
		{Model: gormModel(1), Name: "cpu", Match: storage.JSONMap{"alertname": "HighCPU"}, Timeout: "2m"},
	}}, nil, nil)

	resp := postProbe(t, h, `{"labels":"alertname=Unknown"}`)
	if resp.Matched {
		t.Fatalf("expected no match, got %+v", resp)
	}
}

func TestMatchProbe_RejectsUnparsableLabels(t *testing.T) {
	resp := postProbe(t, probeHandler(), `{"labels":"garbage"}`)

	if resp.Matched {
		t.Fatal("expected no match")
	}
	if resp.Error == "" {
		t.Fatal("expected an error the user can act on")
	}
}

func TestMatchProbe_RejectsEmptyLabels(t *testing.T) {
	resp := postProbe(t, probeHandler(), `{"labels":"   "}`)

	if resp.Error == "" {
		t.Fatal("expected an error naming the empty input")
	}
}
