package webhook

import (
	"context"

	"github.com/tsisar/alert-agent/internal/model"
	"github.com/tsisar/alert-agent/internal/scenario"
	"github.com/tsisar/extended-log-go/log"
)

// job represents a single unit of work to be processed by the queue worker.
type job struct {
	payload  *model.WebhookPayload
	scenario *scenario.Scenario
	prompt   string // non-empty for status notifications (resolved/paused)
}

// JobQueue is the interface for enqueuing alert processing jobs.
type JobQueue interface {
	Enqueue(j job)
	Shutdown(ctx context.Context)
}

// Queue processes alert jobs sequentially in a single worker goroutine,
// preventing interleaved messages in notification channels.
// Used only for local development; alerts are dropped when the buffer is full.
type Queue struct {
	jobs chan job
}

// NewQueue creates a queue with the given buffer size and starts the worker.
func NewQueue(handler *Handler, bufferSize int) *Queue {
	q := &Queue{
		jobs: make(chan job, bufferSize),
	}
	go q.worker(handler)
	return q
}

func (q *Queue) Enqueue(j job) {
	select {
	case q.jobs <- j:
		log.Debugf("[queue] job enqueued: groupKey=%s", j.payload.GroupKey)
	default:
		log.Warnf("[queue] queue full, dropping alert: groupKey=%s", j.payload.GroupKey)
	}
}

// Shutdown is a no-op for the in-memory dev queue; jobs in the
// channel buffer are discarded on process exit.
func (q *Queue) Shutdown(_ context.Context) {
	// Dev-only queue, no persistence to drain.
}

func (q *Queue) worker(h *Handler) {
	for j := range q.jobs {
		log.Debugf("[queue] processing job: groupKey=%s", j.payload.GroupKey)
		var err error
		if j.prompt != "" {
			err = h.notifyStatus(j.payload, j.scenario, j.prompt)
		} else {
			err = h.investigate(j.payload, j.scenario)
		}
		if err != nil {
			log.Errorf("[queue] job failed: groupKey=%s err=%v", j.payload.GroupKey, err)
		}
	}
}
