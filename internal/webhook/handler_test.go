package webhook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tsisar/alert-agent/internal/scenario"
	"github.com/tsisar/alert-agent/internal/storage"
)

type mockScenarioRepo struct {
	scenarios []scenario.Scenario
}

func (m *mockScenarioRepo) Match(alertLabels map[string]string) *scenario.Scenario {
	var catchAll *scenario.Scenario
	for i := range m.scenarios {
		s := &m.scenarios[i]
		if len(s.Match) == 0 {
			catchAll = s
			continue
		}
		if scenario.LabelsMatch(s.Match, alertLabels) {
			return s
		}
	}
	return catchAll
}

func (m *mockScenarioRepo) FindByName(name string) *scenario.Scenario {
	for i := range m.scenarios {
		if m.scenarios[i].Name == name {
			return &m.scenarios[i]
		}
	}
	return nil
}

func (m *mockScenarioRepo) List() ([]storage.Scenario, error) { return nil, nil }
func (m *mockScenarioRepo) Create(s *storage.Scenario) error  { return nil }
func (m *mockScenarioRepo) Update(s *storage.Scenario) error  { return nil }
func (m *mockScenarioRepo) Delete(id uint) error              { return nil }
func (m *mockScenarioRepo) Invalidate()                       {}

func testScenarios() storage.ScenarioRepository {
	return &mockScenarioRepo{
		scenarios: []scenario.Scenario{
			{
				Name:     "catch-all",
				Match:    map[string]string{},
				Prompt:   "Investigate the alert.",
				Timeout:  scenario.Duration{Duration: 1 * time.Minute},
				Priority: "medium",
				Channels: scenario.Channels{Telegram: "-100123"},
			},
		},
	}
}

func newTestHandler() *Handler {
	return NewHandler(testScenarios(), nil, nil, scenario.Channels{}, NewDeduplicator(48*time.Hour))
}

func TestHandleWebhook_ValidPayload(t *testing.T) {
	handler := newTestHandler()

	body := `{
		"receiver": "test",
		"status": "firing",
		"alerts": [
			{
				"status": "firing",
				"labels": {"alertname": "HighCPU", "instance": "web-1"},
				"annotations": {"summary": "CPU is above 90%"},
				"startsAt": "2026-03-30T04:44:18Z",
				"endsAt": "0001-01-01T00:00:00Z",
				"fingerprint": "abc123",
				"values": {"B": 95.2},
				"valueString": "B=95.2"
			}
		],
		"groupLabels": {"alertname": "HighCPU"},
		"commonLabels": {"alertname": "HighCPU"},
		"commonAnnotations": {"summary": "CPU is above 90%"},
		"externalURL": "http://grafana:3000/",
		"version": "1",
		"groupKey": "test-abc123-123456",
		"truncatedAlerts": 0,
		"orgId": 1,
		"title": "[FIRING:1] HighCPU",
		"state": "alerting"
	}`

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.HandleWebhook(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", rec.Code)
	}

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", ct)
	}

	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["status"] != "accepted" {
		t.Fatalf("expected status=accepted, got %q", resp["status"])
	}
}

func TestHandleWebhook_MultipleAlerts(t *testing.T) {
	handler := newTestHandler()

	body := `{
		"receiver": "test",
		"status": "firing",
		"alerts": [
			{
				"status": "firing",
				"labels": {"alertname": "HighCPU"},
				"annotations": {},
				"startsAt": "2026-03-30T04:00:00Z",
				"endsAt": "0001-01-01T00:00:00Z",
				"fingerprint": "aaa"
			},
			{
				"status": "resolved",
				"labels": {"alertname": "HighMemory"},
				"annotations": {},
				"startsAt": "2026-03-30T03:00:00Z",
				"endsAt": "2026-03-30T04:00:00Z",
				"fingerprint": "bbb"
			}
		],
		"groupLabels": {},
		"commonLabels": {},
		"commonAnnotations": {},
		"externalURL": "http://grafana:3000/",
		"version": "1",
		"groupKey": "test-multi"
	}`

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleWebhook(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", rec.Code)
	}
}

func TestHandleWebhook_InvalidJSON(t *testing.T) {
	handler := newTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(`{not valid json`))
	rec := httptest.NewRecorder()

	handler.HandleWebhook(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestHandleWebhook_EmptyBody(t *testing.T) {
	handler := newTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(""))
	rec := httptest.NewRecorder()

	handler.HandleWebhook(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestHandleWebhook_OversizedBody(t *testing.T) {
	handler := newTestHandler()

	bigPayload := `{"receiver":"test","status":"firing","alerts":[{"labels":{"data":"` +
		strings.Repeat("x", maxBodySize) + `"}}]}`

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(bigPayload))
	rec := httptest.NewRecorder()

	handler.HandleWebhook(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for oversized body, got %d", rec.Code)
	}
}

func TestHandleWebhook_NoMatchingScenario(t *testing.T) {
	repo := &mockScenarioRepo{
		scenarios: []scenario.Scenario{
			{
				Name:     "specific",
				Match:    map[string]string{"alertname": "SpecificAlert"},
				Prompt:   "Handle specific alert.",
				Timeout:  scenario.Duration{Duration: 1 * time.Minute},
				Priority: "high",
			},
		},
	}
	handler := NewHandler(repo, nil, nil, scenario.Channels{}, NewDeduplicator(48*time.Hour))

	body := `{
		"receiver": "test",
		"status": "firing",
		"alerts": [{"status": "firing", "labels": {"alertname": "OtherAlert"}, "fingerprint": "x"}],
		"commonLabels": {"alertname": "OtherAlert"},
		"groupKey": "test"
	}`

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleWebhook(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", rec.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["status"] != "no matching scenario" {
		t.Fatalf("expected 'no matching scenario', got %q", resp["status"])
	}
}

func TestChannelTargets_DefaultFallback(t *testing.T) {
	defaults := scenario.Channels{Telegram: "-999", Slack: "#ops"}

	tests := []struct {
		name   string
		ch     scenario.Channels
		wantTG string
		wantSL string
	}{
		{
			name:   "both empty, both inherited",
			ch:     scenario.Channels{},
			wantTG: "-999",
			wantSL: "#ops",
		},
		{
			name:   "telegram set, slack inherited",
			ch:     scenario.Channels{Telegram: "-111"},
			wantTG: "-111",
			wantSL: "#ops",
		},
		{
			name:   "both set, no inheritance",
			ch:     scenario.Channels{Telegram: "-222", Slack: "#custom"},
			wantTG: "-222",
			wantSL: "#custom",
		},
		{
			name:   "slack set, telegram inherited",
			ch:     scenario.Channels{Slack: "#team"},
			wantTG: "-999",
			wantSL: "#team",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targets := channelTargets(tt.ch, defaults)
			for _, tgt := range targets {
				switch tgt.name {
				case "telegram":
					if tgt.dest != tt.wantTG {
						t.Errorf("telegram: got %q, want %q", tgt.dest, tt.wantTG)
					}
				case "slack":
					if tgt.dest != tt.wantSL {
						t.Errorf("slack: got %q, want %q", tgt.dest, tt.wantSL)
					}
				}
			}
		})
	}
}

func TestChannelTargets_NoDefaults(t *testing.T) {
	targets := channelTargets(
		scenario.Channels{Telegram: "-111"},
		scenario.Channels{},
	)
	for _, tgt := range targets {
		switch tgt.name {
		case "telegram":
			if tgt.dest != "-111" {
				t.Errorf("telegram: got %q, want %q", tgt.dest, "-111")
			}
		case "slack":
			if tgt.dest != "" {
				t.Errorf("slack: got %q, want empty", tgt.dest)
			}
		}
	}
}

func TestHandleWebhook_GrafanaTestdataFixture(t *testing.T) {
	handler := newTestHandler()

	body := `{
		"receiver": "test",
		"status": "firing",
		"alerts": [{
			"status": "firing",
			"labels": {"alertname": "TestAlert", "grafana_folder": "Test Folder", "instance": "Grafana"},
			"annotations": {"summary": "Notification test"},
			"startsAt": "2026-03-30T04:44:18.571363065Z",
			"endsAt": "0001-01-01T00:00:00Z",
			"generatorURL": "?orgId=1",
			"fingerprint": "326ea703b01f6100",
			"silenceURL": "http://grafana:3000/alerting/silence/new",
			"dashboardURL": "http://grafana:3000/d/dashboard_uid",
			"panelURL": "http://grafana:3000/d/dashboard_uid?viewPanel=1",
			"values": {"B": 22, "C": 1},
			"valueString": "B=22, C=1",
			"orgId": 1
		}],
		"groupLabels": {"alertname": "TestAlert"},
		"commonLabels": {"alertname": "TestAlert"},
		"commonAnnotations": {"summary": "Notification test"},
		"externalURL": "http://grafana:3000/",
		"version": "1",
		"groupKey": "test-326ea703b01f6100",
		"truncatedAlerts": 0,
		"orgId": 1,
		"title": "[FIRING:1] TestAlert Test Folder Grafana",
		"state": "alerting",
		"message": "**Firing**"
	}`

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleWebhook(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d; body: %s", rec.Code, rec.Body.String())
	}
}
