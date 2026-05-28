package model

import (
	"encoding/json"
	"testing"
	"time"
)

func TestWebhookPayload_FullParse(t *testing.T) {
	// Payload based on the official Grafana webhook documentation:
	// https://grafana.com/docs/grafana/latest/alerting/configure-notifications/manage-contact-points/integrations/webhook-notifier/
	raw := `{
		"receiver": "My Super Webhook",
		"status": "firing",
		"orgId": 1,
		"alerts": [
			{
				"status": "firing",
				"labels": {
					"alertname": "High memory usage",
					"team": "blue",
					"zone": "us-1"
				},
				"annotations": {
					"description": "The system has high memory usage",
					"runbook_url": "https://myrunbook.com/runbook/1234",
					"summary": "This alert was triggered for zone us-1"
				},
				"startsAt": "2021-10-12T09:51:03.157076+02:00",
				"endsAt": "0001-01-01T00:00:00Z",
				"generatorURL": "https://play.grafana.org/alerting/1afz29v7z/edit",
				"fingerprint": "c6eadffa33fcdf37",
				"silenceURL": "https://play.grafana.org/alerting/silence/new?alertmanager=grafana&matchers=alertname%3DT2%2Cteam%3Dblue%2Czone%3Dus-1",
				"dashboardURL": "https://play.grafana.org/d/abc123",
				"panelURL": "https://play.grafana.org/d/abc123?viewPanel=1",
				"imageURL": "https://play.grafana.org/render/d-solo/abc123?panelId=1",
				"values": {
					"B": 44.23943737541908,
					"C": 1
				}
			},
			{
				"status": "firing",
				"labels": {
					"alertname": "High CPU usage",
					"team": "blue",
					"zone": "eu-1"
				},
				"annotations": {
					"description": "The system has high CPU usage",
					"runbook_url": "https://myrunbook.com/runbook/1234",
					"summary": "This alert was triggered for zone eu-1"
				},
				"startsAt": "2021-10-12T09:56:03.157076+02:00",
				"endsAt": "0001-01-01T00:00:00Z",
				"generatorURL": "https://play.grafana.org/alerting/d1rdpdv7k/edit",
				"fingerprint": "bc97ff14869b13e3",
				"silenceURL": "https://play.grafana.org/alerting/silence/new?alertmanager=grafana&matchers=alertname%3DT1%2Cteam%3Dblue%2Czone%3Deu-1",
				"dashboardURL": "",
				"panelURL": "",
				"values": {
					"B": 44.23943737541908,
					"C": 1
				}
			}
		],
		"groupLabels": {},
		"commonLabels": {
			"team": "blue"
		},
		"commonAnnotations": {},
		"externalURL": "https://play.grafana.org/",
		"version": "1",
		"groupKey": "{}:{}",
		"truncatedAlerts": 0,
		"title": "[FIRING:2]  (blue)",
		"state": "alerting",
		"message": "**Firing**\n\nLabels:\n - alertname = T2\n"
	}`

	var payload WebhookPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}

	// Top-level fields
	assertEqual(t, "receiver", "My Super Webhook", payload.Receiver)
	assertEqual(t, "status", "firing", payload.Status)
	assertEqual(t, "orgId", 1, payload.OrgID)
	assertEqual(t, "externalURL", "https://play.grafana.org/", payload.ExternalURL)
	assertEqual(t, "version", "1", payload.Version)
	assertEqual(t, "groupKey", "{}:{}", payload.GroupKey)
	assertEqual(t, "truncatedAlerts", 0, payload.TruncatedAlerts)
	assertEqual(t, "title", "[FIRING:2]  (blue)", payload.Title)
	assertEqual(t, "state", "alerting", payload.State)

	if len(payload.Alerts) != 2 {
		t.Fatalf("expected 2 alerts, got %d", len(payload.Alerts))
	}

	if payload.CommonLabels["team"] != "blue" {
		t.Fatalf("expected commonLabels.team=blue, got %q", payload.CommonLabels["team"])
	}

	// First alert
	a := payload.Alerts[0]
	assertEqual(t, "alert[0].status", "firing", a.Status)
	assertEqual(t, "alert[0].labels.alertname", "High memory usage", a.Labels["alertname"])
	assertEqual(t, "alert[0].labels.team", "blue", a.Labels["team"])
	assertEqual(t, "alert[0].labels.zone", "us-1", a.Labels["zone"])
	assertEqual(t, "alert[0].annotations.summary", "This alert was triggered for zone us-1", a.Annotations["summary"])
	assertEqual(t, "alert[0].generatorURL", "https://play.grafana.org/alerting/1afz29v7z/edit", a.GeneratorURL)
	assertEqual(t, "alert[0].fingerprint", "c6eadffa33fcdf37", a.Fingerprint)
	assertEqual(t, "alert[0].dashboardURL", "https://play.grafana.org/d/abc123", a.DashboardURL)
	assertEqual(t, "alert[0].panelURL", "https://play.grafana.org/d/abc123?viewPanel=1", a.PanelURL)
	assertEqual(t, "alert[0].imageURL", "https://play.grafana.org/render/d-solo/abc123?panelId=1", a.ImageURL)

	if a.StartsAt.IsZero() {
		t.Fatal("expected alert[0].startsAt to be non-zero")
	}

	if v, ok := a.Values["B"]; !ok || v != 44.23943737541908 {
		t.Fatalf("expected alert[0].values.B=44.239..., got %v", v)
	}
	if v, ok := a.Values["C"]; !ok || v != 1.0 {
		t.Fatalf("expected alert[0].values.C=1, got %v", v)
	}

	// Second alert — empty dashboardURL/panelURL
	b := payload.Alerts[1]
	assertEqual(t, "alert[1].dashboardURL", "", b.DashboardURL)
	assertEqual(t, "alert[1].panelURL", "", b.PanelURL)
	assertEqual(t, "alert[1].imageURL", "", b.ImageURL)
}

func TestWebhookPayload_RealGrafanaPayload(t *testing.T) {
	// Real payload from a Grafana instance with extra fields (valueString, orgId on alert).
	raw := `{
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

	var payload WebhookPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(payload.Alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(payload.Alerts))
	}

	a := payload.Alerts[0]
	assertEqual(t, "valueString", "B=22, C=1", a.ValueString)
	assertEqual(t, "alert.orgId", 1, a.OrgID)

	if v := a.Values["B"]; v != 22.0 {
		t.Fatalf("expected values.B=22, got %v", v)
	}

	expectedTime := time.Date(2026, 3, 30, 4, 44, 18, 571363065, time.UTC)
	if !a.StartsAt.Equal(expectedTime) {
		t.Fatalf("expected startsAt=%v, got %v", expectedTime, a.StartsAt)
	}
}

func TestWebhookPayload_ResolvedAlert(t *testing.T) {
	raw := `{
		"receiver": "webhook",
		"status": "resolved",
		"alerts": [{
			"status": "resolved",
			"labels": {"alertname": "DiskFull"},
			"annotations": {},
			"startsAt": "2026-03-30T01:00:00Z",
			"endsAt": "2026-03-30T02:00:00Z",
			"fingerprint": "def456",
			"values": {}
		}],
		"groupLabels": {},
		"commonLabels": {"alertname": "DiskFull"},
		"commonAnnotations": {},
		"externalURL": "http://grafana:3000/",
		"version": "1",
		"groupKey": "resolved-test",
		"state": "ok"
	}`

	var payload WebhookPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	assertEqual(t, "status", "resolved", payload.Status)
	assertEqual(t, "state", "ok", payload.State)

	a := payload.Alerts[0]
	assertEqual(t, "alert.status", "resolved", a.Status)

	if a.EndsAt.IsZero() {
		t.Fatal("expected endsAt to be non-zero for resolved alert")
	}
}

func TestAlert_EmptyValues(t *testing.T) {
	raw := `{
		"status": "firing",
		"labels": {"alertname": "NoValues"},
		"annotations": {},
		"startsAt": "2026-03-30T00:00:00Z",
		"endsAt": "0001-01-01T00:00:00Z",
		"fingerprint": "000",
		"values": null
	}`

	var a Alert
	if err := json.Unmarshal([]byte(raw), &a); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if a.Values != nil {
		t.Fatalf("expected nil values, got %v", a.Values)
	}
}

func assertEqual[T comparable](t *testing.T, field string, expected, actual T) {
	t.Helper()
	if expected != actual {
		t.Fatalf("%s: expected %v, got %v", field, expected, actual)
	}
}
