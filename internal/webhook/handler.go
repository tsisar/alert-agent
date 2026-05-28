package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/tsisar/alert-agent/internal/agent"
	"github.com/tsisar/alert-agent/internal/model"
	"github.com/tsisar/alert-agent/internal/notify"
	"github.com/tsisar/alert-agent/internal/scenario"
	"github.com/tsisar/alert-agent/internal/storage"
	"github.com/tsisar/extended-log-go/log"
)

const (
	maxBodySize = 1 << 20 // 1 MB
	contentType = "Content-Type"
)

type Handler struct {
	scenarios       storage.ScenarioRepository
	agent           *agent.Agent
	notifiers       map[string]notify.Notifier
	defaultChannels scenario.Channels
	dedup           DedupStore
	queue           JobQueue
}

func NewHandler(scenarios storage.ScenarioRepository, ag *agent.Agent, notifiers map[string]notify.Notifier, defaultChannels scenario.Channels, dedup DedupStore) *Handler {
	return &Handler{
		scenarios:       scenarios,
		agent:           ag,
		notifiers:       notifiers,
		defaultChannels: defaultChannels,
		dedup:           dedup,
	}
}

// SetQueue sets the job queue for the handler. Must be called before handling requests.
func (h *Handler) SetQueue(q JobQueue) {
	h.queue = q
}

func (h *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	var payload model.WebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Errorf("failed to decode webhook payload: %v", err)
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	log.Infof("webhook received: status=%s alerts=%d groupKey=%s",
		payload.Status, len(payload.Alerts), payload.GroupKey)

	if payloadJSON, err := json.MarshalIndent(payload, "", "  "); err == nil {
		log.Tracef("[webhook] payload:\n%s", payloadJSON)
	}

	sc := h.scenarios.Match(payload.CommonLabels)
	if sc == nil {
		log.Warnf("no matching scenario for labels: %v", payload.CommonLabels)
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "no matching scenario"})
		return
	}

	log.Infof("matched scenario: %s (priority=%s, timeout=%s)", sc.Name, sc.Priority, sc.Timeout.Duration)

	// On resolved status, clear the dedup entry and send a short notification.
	if payload.Status == "resolved" {
		h.dedup.Clear(payload.GroupKey)
		log.Infof("alert resolved, dedup cleared: groupKey=%s", payload.GroupKey)
		if h.queue != nil {
			h.queue.Enqueue(job{payload: &payload, scenario: sc, prompt: h.agent.ResolvedPrompt()})
		}
		w.Header().Set(contentType, "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "resolved", "scenario": sc.Name})
		return
	}

	// On paused state, send a short notification without investigation.
	if payload.State == "paused" {
		log.Infof("alert paused: groupKey=%s", payload.GroupKey)
		if h.queue != nil {
			h.queue.Enqueue(job{payload: &payload, scenario: sc, prompt: h.agent.PausedPrompt()})
		}
		w.Header().Set(contentType, "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "paused", "scenario": sc.Name})
		return
	}

	// Deduplicate firing alerts within the cooldown period.
	if !h.dedup.ShouldProcess(payload.GroupKey) {
		log.Infof("alert suppressed (cooldown): groupKey=%s", payload.GroupKey)
		w.Header().Set(contentType, "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "suppressed", "scenario": sc.Name})
		return
	}

	if h.queue != nil {
		h.queue.Enqueue(job{payload: &payload, scenario: sc})
	}

	w.Header().Set(contentType, "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "accepted", "scenario": sc.Name})
}

// channelTarget maps a notifier key to the destination extracted from the scenario.
type channelTarget struct {
	name string
	dest string
}

// resolveChannel returns the effective destination for a channel.
// Empty string falls back to the default; "-" explicitly disables the channel.
func resolveChannel(val, fallback string) string {
	if val == "-" {
		return ""
	}
	if val == "" {
		return fallback
	}
	return val
}

func channelTargets(ch, defaults scenario.Channels) []channelTarget {
	return []channelTarget{
		{"telegram", resolveChannel(ch.Telegram, defaults.Telegram)},
		{"slack", resolveChannel(ch.Slack, defaults.Slack)},
	}
}

func (h *Handler) notifyStatus(payload *model.WebhookPayload, sc *scenario.Scenario, prompt string) error {
	if h.agent == nil {
		return fmt.Errorf("agent not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	msg, err := h.agent.Notify(ctx, payload, prompt)
	if err != nil {
		return fmt.Errorf("notify status (scenario %s): %w", sc.Name, err)
	}

	log.Infof("status notification ready: scenario=%s msg=%s", sc.Name, msg)

	for _, t := range channelTargets(sc.Channels, h.defaultChannels) {
		if t.dest == "" {
			continue
		}
		n, ok := h.notifiers[t.name]
		if !ok {
			continue
		}
		if err := n.SendMessage(ctx, t.dest, msg); err != nil {
			log.Errorf("failed to send %s status notification: %v", t.name, err)
		}
	}
	return nil
}

func (h *Handler) investigate(payload *model.WebhookPayload, sc *scenario.Scenario) error {
	if h.agent == nil {
		return fmt.Errorf("agent not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), sc.Timeout.Duration)
	defer cancel()

	result, err := h.agent.Run(ctx, payload, sc)
	if err != nil {
		return fmt.Errorf("investigation (scenario %s): %w", sc.Name, err)
	}

	log.Infof("agent report ready: scenario=%s summary_len=%d report_len=%d images=%d",
		sc.Name, len(result.Summary), len(result.Report), len(result.Images))
	log.Debugf("[notify] report:\n%s", result.Report)

	for _, t := range channelTargets(sc.Channels, h.defaultChannels) {
		if t.dest == "" {
			continue
		}
		n, ok := h.notifiers[t.name]
		if !ok {
			log.Warnf("[notify] %s notifier not configured, skipping", t.name)
			continue
		}
		h.sendReport(ctx, n, t.name, t.dest, sc, result)
	}
	return nil
}

// notifyFailure sends a fallback message when a job permanently fails
// after all retry attempts have been exhausted.
func (h *Handler) notifyFailure(payload *model.WebhookPayload, sc *scenario.Scenario) {
	alertName := payload.CommonLabels["alertname"]
	summary := payload.CommonAnnotations["summary"]
	if summary == "" {
		summary = payload.CommonAnnotations["description"]
	}

	msg := fmt.Sprintf("Investigation failed for alert %q: %s\nThe alert was received but could not be investigated after multiple attempts.",
		alertName, summary)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	for _, t := range channelTargets(sc.Channels, h.defaultChannels) {
		if t.dest == "" {
			continue
		}
		n, ok := h.notifiers[t.name]
		if !ok {
			continue
		}
		if err := n.SendMessage(ctx, t.dest, msg); err != nil {
			log.Errorf("failed to send %s failure notification: %v", t.name, err)
		}
	}
}

func (h *Handler) sendReport(ctx context.Context, n notify.Notifier, name, dest string, sc *scenario.Scenario, result *agent.Result) {
	log.Debugf("[notify] sending to %s dest=%s", name, dest)

	summary := result.Summary
	if summary == "" {
		summary = result.Report
	}
	if err := n.SendMessage(ctx, dest, summary); err != nil {
		log.Errorf("failed to send %s summary: %v", name, err)
	}

	filename := fmt.Sprintf("report_%s_%s.md", sc.Name, time.Now().Format("20060102_150405"))
	if err := n.SendDocument(ctx, dest, []byte(result.Report), filename, ""); err != nil {
		log.Errorf("failed to send %s report document: %v", name, err)
	}

	if sc.SendImages {
		for i, img := range result.Images {
			log.Debugf("[notify] sending image %d/%d to %s (%d bytes)", i+1, len(result.Images), name, len(img))
			if err := n.SendPhoto(ctx, dest, img, ""); err != nil {
				log.Errorf("failed to send %s image %d: %v", name, i+1, err)
			}
		}
	}
}
