package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/tsisar/alert-agent/internal/model"
	"github.com/tsisar/alert-agent/internal/scenario"
	"github.com/tsisar/alert-agent/internal/storage"
)

func startMiniredis(t *testing.T) *miniredis.Miniredis {
	t.Helper()
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	t.Cleanup(s.Close)
	return s
}

func redisClient(t *testing.T, s *miniredis.Miniredis) *redis.Client {
	t.Helper()
	return redis.NewClient(&redis.Options{Addr: s.Addr()})
}

func testPayload(groupKey string) *model.WebhookPayload {
	return &model.WebhookPayload{
		Receiver:     "test",
		Status:       "firing",
		GroupKey:     groupKey,
		CommonLabels: map[string]string{"alertname": "TestAlert"},
		CommonAnnotations: map[string]string{
			"summary": "test alert fired",
		},
		Alerts: []model.Alert{
			{
				Status:      "firing",
				Labels:      map[string]string{"alertname": "TestAlert"},
				Fingerprint: "abc",
			},
		},
	}
}

func testScenario() *scenario.Scenario {
	return &scenario.Scenario{
		Name:     "catch-all",
		Match:    map[string]string{},
		Prompt:   "Investigate.",
		Timeout:  scenario.Duration{Duration: 1 * time.Minute},
		Priority: "medium",
	}
}

type testScenarioRepo struct {
	sc *scenario.Scenario
}

func (r *testScenarioRepo) Match(map[string]string) *scenario.Scenario { return r.sc }
func (r *testScenarioRepo) FindByName(name string) *scenario.Scenario {
	if r.sc != nil && r.sc.Name == name {
		return r.sc
	}
	return nil
}
func (r *testScenarioRepo) List() ([]storage.Scenario, error) { return nil, nil }
func (r *testScenarioRepo) Create(*storage.Scenario) error    { return nil }
func (r *testScenarioRepo) Update(*storage.Scenario) error    { return nil }
func (r *testScenarioRepo) Delete(uint) error                 { return nil }
func (r *testScenarioRepo) Invalidate()                       {}

func TestRedisQueue_EnqueueAndProcess(t *testing.T) {
	s := startMiniredis(t)
	rdb := redisClient(t, s)

	sc := testScenario()
	repo := &testScenarioRepo{sc: sc}

	processed := make(chan string, 1)
	handler := NewHandler(repo, nil, nil, scenario.Channels{}, NewDeduplicator(48*time.Hour))
	handler.agent = nil

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	q, err := NewRedisQueue(ctx, rdb, handler, repo)
	if err != nil {
		t.Fatalf("NewRedisQueue: %v", err)
	}

	// Verify stream and group were created.
	groups, err := rdb.XInfoGroups(ctx, redisKeyStream).Result()
	if err != nil {
		t.Fatalf("XInfoGroups: %v", err)
	}
	if len(groups) != 1 || groups[0].Name != redisGroup {
		t.Fatalf("expected group %q, got %v", redisGroup, groups)
	}

	// Enqueue a job with a prompt (notifyStatus path) — will fail because
	// agent is nil, but we can verify the message lands in the stream.
	payload := testPayload("test-key-1")
	q.Enqueue(job{payload: payload, scenario: sc, prompt: "test prompt"})

	// Verify message is in the stream.
	msgs, err := rdb.XRange(ctx, redisKeyStream, "-", "+").Result()
	if err != nil {
		t.Fatalf("XRange: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message in stream, got %d", len(msgs))
	}

	dataStr, ok := msgs[0].Values["data"].(string)
	if !ok {
		t.Fatal("expected 'data' field in stream message")
	}
	var rj redisJob
	if err := json.Unmarshal([]byte(dataStr), &rj); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if rj.Payload.GroupKey != "test-key-1" {
		t.Fatalf("expected groupKey 'test-key-1', got %q", rj.Payload.GroupKey)
	}
	if rj.ScenarioName != "catch-all" {
		t.Fatalf("expected scenario 'catch-all', got %q", rj.ScenarioName)
	}

	_ = processed
	q.Shutdown(context.Background())
}

func TestRedisQueue_AckRemovesMessage(t *testing.T) {
	s := startMiniredis(t)
	rdb := redisClient(t, s)
	ctx := context.Background()

	if err := ensureGroup(ctx, rdb); err != nil {
		t.Fatalf("ensureGroup: %v", err)
	}

	// Add a message directly.
	msgID, err := rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: redisKeyStream,
		Values: map[string]interface{}{"data": `{"payload":{},"scenario_name":"x"}`},
	}).Result()
	if err != nil {
		t.Fatalf("XADD: %v", err)
	}

	// Read it to put in PEL.
	_, err = rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    redisGroup,
		Consumer: "test-consumer",
		Streams:  []string{redisKeyStream, ">"},
		Count:    1,
	}).Result()
	if err != nil {
		t.Fatalf("XREADGROUP: %v", err)
	}

	// Verify pending.
	pending, err := rdb.XPending(ctx, redisKeyStream, redisGroup).Result()
	if err != nil {
		t.Fatalf("XPENDING: %v", err)
	}
	if pending.Count != 1 {
		t.Fatalf("expected 1 pending, got %d", pending.Count)
	}

	// ACK + DEL.
	q := &RedisQueue{client: rdb, consumer: "test-consumer"}
	q.ack(ctx, msgID)

	// Verify no pending and no messages.
	pending, err = rdb.XPending(ctx, redisKeyStream, redisGroup).Result()
	if err != nil {
		t.Fatalf("XPENDING after ack: %v", err)
	}
	if pending.Count != 0 {
		t.Fatalf("expected 0 pending after ack, got %d", pending.Count)
	}

	msgs, err := rdb.XRange(ctx, redisKeyStream, "-", "+").Result()
	if err != nil {
		t.Fatalf("XRange after ack: %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("expected 0 messages after ack+del, got %d", len(msgs))
	}
}

func TestRedisQueue_MoveToDLQ(t *testing.T) {
	s := startMiniredis(t)
	rdb := redisClient(t, s)
	ctx := context.Background()

	q := &RedisQueue{client: rdb, consumer: "test-consumer"}

	msg := redis.XMessage{
		ID:     "123-0",
		Values: map[string]interface{}{"data": `{"payload":{},"scenario_name":"x"}`},
	}

	if err := q.moveToDLQ(ctx, msg, "test-group-key"); err != nil {
		t.Fatalf("moveToDLQ: %v", err)
	}

	// Verify DLQ has the message.
	dlqMsgs, err := rdb.XRange(ctx, redisKeyDLQ, "-", "+").Result()
	if err != nil {
		t.Fatalf("XRange DLQ: %v", err)
	}
	if len(dlqMsgs) != 1 {
		t.Fatalf("expected 1 DLQ message, got %d", len(dlqMsgs))
	}

	if dlqMsgs[0].Values["original_id"] != "123-0" {
		t.Fatalf("expected original_id '123-0', got %v", dlqMsgs[0].Values["original_id"])
	}
	if dlqMsgs[0].Values["group_key"] != "test-group-key" {
		t.Fatalf("expected group_key 'test-group-key', got %v", dlqMsgs[0].Values["group_key"])
	}
	if _, ok := dlqMsgs[0].Values["failed_at"]; !ok {
		t.Fatal("expected 'failed_at' field in DLQ message")
	}
}

func TestRedisQueue_MoveToDLQError(t *testing.T) {
	s := startMiniredis(t)
	rdb := redisClient(t, s)
	ctx := context.Background()

	if err := rdb.Set(ctx, redisKeyDLQ, "wrong-type", 0).Err(); err != nil {
		t.Fatalf("SET wrong type key: %v", err)
	}

	q := &RedisQueue{client: rdb, consumer: "test-consumer"}
	msg := redis.XMessage{
		ID:     "123-0",
		Values: map[string]interface{}{"data": `{"payload":{},"scenario_name":"x"}`},
	}

	if err := q.moveToDLQ(ctx, msg, "test-group-key"); err == nil {
		t.Fatal("expected moveToDLQ to fail on WRONGTYPE key")
	}
}

func TestRedisQueue_Shutdown(t *testing.T) {
	s := startMiniredis(t)
	rdb := redisClient(t, s)

	sc := testScenario()
	repo := &testScenarioRepo{sc: sc}
	handler := NewHandler(repo, nil, nil, scenario.Channels{}, NewDeduplicator(48*time.Hour))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	q, err := NewRedisQueue(ctx, rdb, handler, repo)
	if err != nil {
		t.Fatalf("NewRedisQueue: %v", err)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	q.Shutdown(shutdownCtx)
}

func TestRedisQueue_EnsureGroupIdempotent(t *testing.T) {
	s := startMiniredis(t)
	rdb := redisClient(t, s)
	ctx := context.Background()

	if err := ensureGroup(ctx, rdb); err != nil {
		t.Fatalf("first ensureGroup: %v", err)
	}
	if err := ensureGroup(ctx, rdb); err != nil {
		t.Fatalf("second ensureGroup should be idempotent: %v", err)
	}
}

func TestWarnLegacyQueue(t *testing.T) {
	s := startMiniredis(t)
	rdb := redisClient(t, s)
	ctx := context.Background()

	// No legacy key — should not panic.
	WarnLegacyQueue(ctx, rdb)

	// Set legacy key.
	rdb.LPush(ctx, redisKeyQueue, "old-job")

	// Should log warning but not panic.
	WarnLegacyQueue(ctx, rdb)
}

func TestRedisDedup_ShouldProcess(t *testing.T) {
	s := startMiniredis(t)
	rdb := redisClient(t, s)

	dedup := NewRedisDedup(rdb, 1*time.Hour)

	if !dedup.ShouldProcess("key-1") {
		t.Fatal("first call should return true")
	}
	if dedup.ShouldProcess("key-1") {
		t.Fatal("second call should return false (within cooldown)")
	}

	dedup.Clear("key-1")
	if !dedup.ShouldProcess("key-1") {
		t.Fatal("after clear, should return true again")
	}
}

func TestRedisQueue_ProcessMessageUnknownScenario(t *testing.T) {
	s := startMiniredis(t)
	rdb := redisClient(t, s)
	ctx := context.Background()

	if err := ensureGroup(ctx, rdb); err != nil {
		t.Fatalf("ensureGroup: %v", err)
	}

	repo := &testScenarioRepo{sc: nil}
	handler := NewHandler(repo, nil, nil, scenario.Channels{}, NewDeduplicator(48*time.Hour))

	q := &RedisQueue{client: rdb, consumer: "test-consumer"}

	rj := redisJob{
		Payload:      testPayload("unknown-scenario-key"),
		ScenarioName: "nonexistent",
	}
	data, _ := json.Marshal(rj)

	msgID, err := rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: redisKeyStream,
		Values: map[string]interface{}{"data": string(data)},
	}).Result()
	if err != nil {
		t.Fatalf("XADD: %v", err)
	}

	// Read to get it in PEL.
	streams, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    redisGroup,
		Consumer: "test-consumer",
		Streams:  []string{redisKeyStream, ">"},
		Count:    1,
	}).Result()
	if err != nil {
		t.Fatalf("XREADGROUP: %v", err)
	}

	msg := streams[0].Messages[0]
	q.processMessage(ctx, handler, repo, msg, 1)

	// Unknown scenario should be acked (no retry).
	pending, _ := rdb.XPending(ctx, redisKeyStream, redisGroup).Result()
	if pending.Count != 0 {
		t.Fatalf("expected 0 pending after unknown scenario, got %d", pending.Count)
	}
	_ = msgID
}

func TestRedisQueue_ProcessMessageNilPayload(t *testing.T) {
	s := startMiniredis(t)
	rdb := redisClient(t, s)
	ctx := context.Background()

	if err := ensureGroup(ctx, rdb); err != nil {
		t.Fatalf("ensureGroup: %v", err)
	}

	msgID, err := rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: redisKeyStream,
		Values: map[string]interface{}{"data": `{"payload":null,"scenario_name":"catch-all"}`},
	}).Result()
	if err != nil {
		t.Fatalf("XADD: %v", err)
	}

	streams, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    redisGroup,
		Consumer: "test-consumer",
		Streams:  []string{redisKeyStream, ">"},
		Count:    1,
	}).Result()
	if err != nil {
		t.Fatalf("XREADGROUP: %v", err)
	}

	q := &RedisQueue{client: rdb, consumer: "test-consumer"}
	msg := streams[0].Messages[0]
	q.processMessage(ctx, nil, nil, msg, 1)

	pending, err := rdb.XPending(ctx, redisKeyStream, redisGroup).Result()
	if err != nil {
		t.Fatalf("XPENDING: %v", err)
	}
	if pending.Count != 0 {
		t.Fatalf("expected 0 pending after nil payload, got %d", pending.Count)
	}

	msgs, err := rdb.XRange(ctx, redisKeyStream, "-", "+").Result()
	if err != nil {
		t.Fatalf("XRange after nil payload: %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("expected 0 stream messages after nil payload, got %d", len(msgs))
	}

	_ = msgID
}

func TestConsumerName(t *testing.T) {
	name := consumerName()
	if name == "" {
		t.Fatal("consumer name should not be empty")
	}
	if len(name) < 3 {
		t.Fatalf("consumer name too short: %q", name)
	}
	_ = fmt.Sprintf("consumer: %s", name)
}
