package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tsisar/alert-agent/internal/model"
	"github.com/tsisar/alert-agent/internal/storage"
	"github.com/tsisar/extended-log-go/log"
)

const (
	redisKeyDedup  = "alert-agent:dedup:"
	redisKeyQueue  = "alert-agent:queue" // legacy list key
	redisKeyStream = "alert-agent:stream"
	redisKeyDLQ    = "alert-agent:dlq"
	redisGroup     = "workers"

	streamBlockTimeout = 5 * time.Second
	claimMinIdle       = 10 * time.Minute
	maxDeliveries      = 3
)

// RedisDedup implements DedupStore using Redis SET NX with TTL.
type RedisDedup struct {
	client   *redis.Client
	cooldown time.Duration
}

func NewRedisDedup(client *redis.Client, cooldown time.Duration) *RedisDedup {
	return &RedisDedup{client: client, cooldown: cooldown}
}

func (d *RedisDedup) ShouldProcess(groupKey string) bool {
	ok, err := d.client.SetArgs(context.Background(), redisKeyDedup+dedupKey(groupKey), "1", redis.SetArgs{
		TTL:  d.cooldown,
		Mode: "NX",
	}).Result()
	if err != nil && err != redis.Nil {
		log.Errorf("[redis-dedup] SET NX error: %v", err)
		return true // fail-open
	}
	return ok == "OK"
}

func (d *RedisDedup) Clear(groupKey string) {
	if err := d.client.Del(context.Background(), redisKeyDedup+dedupKey(groupKey)).Err(); err != nil {
		log.Errorf("[redis-dedup] DEL error: %v", err)
	}
}

// redisJob is the serializable representation of a job for the Redis stream.
type redisJob struct {
	Payload      *model.WebhookPayload `json:"payload"`
	ScenarioName string                `json:"scenario_name"`
	Prompt       string                `json:"prompt,omitempty"`
}

// RedisQueue implements JobQueue using a Redis Stream with consumer groups
// for at-least-once processing, automatic retry via XAUTOCLAIM, and a
// dead-letter queue for permanently failed jobs.
type RedisQueue struct {
	client   *redis.Client
	consumer string
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

func consumerName() string {
	host, _ := os.Hostname()
	return fmt.Sprintf("%s-%d", host, os.Getpid())
}

// NewRedisQueue creates the consumer group (if needed), starts the worker
// goroutine, and returns the queue. The provided context controls the
// worker lifetime.
func NewRedisQueue(ctx context.Context, client *redis.Client, handler *Handler, scenarios storage.ScenarioRepository) (*RedisQueue, error) {
	if err := ensureGroup(ctx, client); err != nil {
		return nil, err
	}

	workerCtx, cancel := context.WithCancel(ctx)
	q := &RedisQueue{
		client:   client,
		consumer: consumerName(),
		cancel:   cancel,
	}
	q.wg.Add(1)
	go q.worker(workerCtx, handler, scenarios)
	return q, nil
}

func ensureGroup(ctx context.Context, client *redis.Client) error {
	err := client.XGroupCreateMkStream(ctx, redisKeyStream, redisGroup, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("create consumer group: %w", err)
	}
	return nil
}

func (q *RedisQueue) Enqueue(j job) {
	rj := redisJob{
		Payload:      j.payload,
		ScenarioName: j.scenario.Name,
		Prompt:       j.prompt,
	}
	data, err := json.Marshal(rj)
	if err != nil {
		log.Errorf("[stream] marshal error: %v", err)
		return
	}
	err = q.client.XAdd(context.Background(), &redis.XAddArgs{
		Stream: redisKeyStream,
		Values: map[string]interface{}{"data": string(data)},
	}).Err()
	if err != nil {
		log.Errorf("[stream] XADD error: %v", err)
	} else {
		log.Debugf("[stream] job enqueued: groupKey=%s", j.payload.GroupKey)
	}
}

// Shutdown signals the worker to stop and waits for the current job to
// finish within the given context deadline.
func (q *RedisQueue) Shutdown(ctx context.Context) {
	q.cancel()
	done := make(chan struct{})
	go func() {
		q.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		log.Info("[stream] queue shutdown complete")
	case <-ctx.Done():
		log.Warn("[stream] queue shutdown timed out, worker still running")
	}
}

func (q *RedisQueue) worker(ctx context.Context, h *Handler, scenarios storage.ScenarioRepository) {
	defer q.wg.Done()
	log.Infof("[stream] worker started: consumer=%s", q.consumer)

	for {
		if ctx.Err() != nil {
			log.Info("[stream] worker stopped")
			return
		}

		// Reclaim stuck messages from other (dead) consumers.
		q.reclaimStuck(ctx, h, scenarios)

		// Read new messages.
		streams, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    redisGroup,
			Consumer: q.consumer,
			Streams:  []string{redisKeyStream, ">"},
			Count:    1,
			Block:    streamBlockTimeout,
		}).Result()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			if err != redis.Nil {
				log.Errorf("[stream] XREADGROUP error: %v", err)
				time.Sleep(time.Second)
			}
			continue
		}

		for _, stream := range streams {
			for _, msg := range stream.Messages {
				q.processMessage(ctx, h, scenarios, msg, 1)
			}
		}
	}
}

// reclaimStuck uses XAUTOCLAIM to take ownership of messages that have
// been pending longer than claimMinIdle (consumer crashed / timed out).
func (q *RedisQueue) reclaimStuck(ctx context.Context, h *Handler, scenarios storage.ScenarioRepository) {
	msgs, _, err := q.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   redisKeyStream,
		Group:    redisGroup,
		Consumer: q.consumer,
		MinIdle:  claimMinIdle,
		Start:    "0-0",
		Count:    10,
	}).Result()
	if err != nil {
		if ctx.Err() == nil {
			log.Errorf("[stream] XAUTOCLAIM error: %v", err)
		}
		return
	}

	for _, msg := range msgs {
		deliveries := q.deliveryCount(ctx, msg.ID)
		log.Warnf("[stream] reclaimed stuck message: id=%s deliveries=%d", msg.ID, deliveries)
		q.processMessage(ctx, h, scenarios, msg, deliveries)
	}
}

// deliveryCount returns how many times a message has been delivered,
// using XPENDING for the specific message ID.
func (q *RedisQueue) deliveryCount(ctx context.Context, msgID string) int64 {
	pending, err := q.client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: redisKeyStream,
		Group:  redisGroup,
		Start:  msgID,
		End:    msgID,
		Count:  1,
	}).Result()
	if err != nil || len(pending) == 0 {
		return 1
	}
	return pending[0].RetryCount
}

func (q *RedisQueue) processMessage(ctx context.Context, h *Handler, scenarios storage.ScenarioRepository, msg redis.XMessage, deliveries int64) {
	dataStr, ok := msg.Values["data"].(string)
	if !ok {
		log.Errorf("[stream] invalid message format: id=%s", msg.ID)
		q.ack(ctx, msg.ID)
		return
	}

	var rj redisJob
	if err := json.Unmarshal([]byte(dataStr), &rj); err != nil {
		log.Errorf("[stream] unmarshal error: id=%s err=%v", msg.ID, err)
		q.ack(ctx, msg.ID)
		return
	}
	if rj.Payload == nil {
		log.Errorf("[stream] invalid payload: id=%s", msg.ID)
		q.ack(ctx, msg.ID)
		return
	}

	sc := scenarios.FindByName(rj.ScenarioName)
	if sc == nil {
		log.Errorf("[stream] unknown scenario %q: id=%s", rj.ScenarioName, msg.ID)
		q.ack(ctx, msg.ID)
		return
	}

	log.Infof("[stream] processing job: id=%s groupKey=%s scenario=%s delivery=%d",
		msg.ID, rj.Payload.GroupKey, rj.ScenarioName, deliveries)

	var err error
	if rj.Prompt != "" {
		err = h.notifyStatus(rj.Payload, sc, rj.Prompt)
	} else {
		err = h.investigate(rj.Payload, sc)
	}

	if err == nil {
		q.ack(ctx, msg.ID)
		return
	}

	log.Errorf("[stream] job failed: id=%s groupKey=%s err=%v", msg.ID, rj.Payload.GroupKey, err)

	if deliveries >= maxDeliveries {
		log.Errorf("[stream] max retries exhausted, moving to DLQ: id=%s groupKey=%s", msg.ID, rj.Payload.GroupKey)
		if err := q.moveToDLQ(ctx, msg, rj.Payload.GroupKey); err != nil {
			log.Errorf("[stream] failed to move message to DLQ: id=%s groupKey=%s err=%v", msg.ID, rj.Payload.GroupKey, err)
			return
		}
		q.ack(ctx, msg.ID)
		h.notifyFailure(rj.Payload, sc)
	}
	// Otherwise leave in PEL for XAUTOCLAIM to retry later.
}

func (q *RedisQueue) ack(ctx context.Context, msgID string) {
	pipe := q.client.Pipeline()
	pipe.XAck(ctx, redisKeyStream, redisGroup, msgID)
	pipe.XDel(ctx, redisKeyStream, msgID)
	if _, err := pipe.Exec(ctx); err != nil {
		log.Errorf("[stream] ACK/DEL error: id=%s err=%v", msgID, err)
	}
}

func (q *RedisQueue) moveToDLQ(ctx context.Context, msg redis.XMessage, groupKey string) error {
	values := make(map[string]interface{}, len(msg.Values)+3)
	for k, v := range msg.Values {
		values[k] = v
	}
	values["original_id"] = msg.ID
	values["group_key"] = groupKey
	values["failed_at"] = time.Now().UTC().Format(time.RFC3339)

	err := q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: redisKeyDLQ,
		Values: values,
	}).Err()
	if err != nil {
		return fmt.Errorf("dlq xadd: %w", err)
	}
	return nil
}

// WarnLegacyQueue logs a warning if the old list-based queue key still
// exists in Redis. Operators should drain or delete it manually.
func WarnLegacyQueue(ctx context.Context, client *redis.Client) {
	n, err := client.Exists(ctx, redisKeyQueue).Result()
	if err != nil {
		log.Errorf("[stream] failed to check legacy queue key: %v", err)
		return
	}
	if n > 0 {
		log.Warnf("legacy Redis queue key %q still exists — drain or delete it manually", redisKeyQueue)
	}
}
