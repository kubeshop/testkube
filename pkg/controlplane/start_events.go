package controlplane

import (
	"context"
	"sync"

	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
)

const (
	// startEventWorkers publish concurrently so one slow subscriber does not hold
	// up the rest of a batch: Notify retries up to three times with a sleep
	// between attempts.
	startEventWorkers = 4
	// startEventBufferBatches is how many full dispatch batches may queue before
	// events start being dropped.
	startEventBufferBatches = 8
)

// startEventDispatcher hydrates executions and publishes their start events off
// the dispatch request.
//
// Listeners want the whole execution; the runner does not. It reads a dozen
// scalars and fetches the workflow itself, so the poll returns a lean projection
// and the event is built here, where a slow publish delays a webhook instead of
// the runner's next batch of work.
type startEventDispatcher struct {
	ids     chan string
	repo    testworkflow.Repository
	emitter *event.Emitter
	envID   string
	log     *zap.SugaredLogger

	start sync.Once
}

func newStartEventDispatcher(repo testworkflow.Repository, emitter *event.Emitter, envID string, log *zap.SugaredLogger, bufferBatches int) *startEventDispatcher {
	return &startEventDispatcher{
		ids:     make(chan string, bufferBatches),
		repo:    repo,
		emitter: emitter,
		envID:   envID,
		log:     log,
	}
}

// run starts the workers once. Events queued at shutdown are lost; they already
// were, because the request they belonged to could time out mid-loop.
func (d *startEventDispatcher) run(ctx context.Context) {
	d.start.Do(func() {
		for i := 0; i < startEventWorkers; i++ {
			go d.work(ctx)
		}
	})
}

func (d *startEventDispatcher) work(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case id := <-d.ids:
			d.publish(ctx, id)
		}
	}
}

func (d *startEventDispatcher) publish(ctx context.Context, id string) {
	execution, err := d.repo.Get(ctx, id)
	if err != nil {
		d.log.Errorw("failed to load execution for its start event", "id", id, "err", err)
		return
	}
	d.emitter.Notify(testkube.NewEventStartTestWorkflow(&execution, d.envID))
}

// enqueue hands ids to the workers without ever blocking.
//
// Dropping an event when the queue is full is deliberate. This runs inside the
// dispatch RPC, and blocking it is the exact failure this whole path exists to
// prevent: a lost webhook is recoverable, a stalled runner is not.
func (d *startEventDispatcher) enqueue(ids []string) {
	for _, id := range ids {
		select {
		case d.ids <- id:
		default:
			d.log.Errorw("dropping workflow start event, dispatcher is saturated", "id", id)
		}
	}
}

// enqueueStartEvents is nil-safe so a Server built as a struct literal, as the
// tests do, still dispatches.
func (s *Server) enqueueStartEvents(ids []string) {
	if s.startEvents == nil || len(ids) == 0 {
		return
	}
	s.startEvents.enqueue(ids)
}
