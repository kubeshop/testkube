package controlplane

import (
	"context"
	"sync"
	"time"

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
	// enqueue starts publishing inline instead.
	startEventBufferBatches = 8
	// startEventDrainTimeout bounds how long shutdown waits for the queue.
	startEventDrainTimeout = 10 * time.Second
)

// startEventDispatcher hydrates executions and publishes their start events off
// the dispatch request.
//
// Listeners want the whole execution; the runner does not. It reads a dozen
// scalars and fetches the workflow itself, so the poll returns a lean projection
// and the event is built here, where a slow publish delays a webhook instead of
// the runner's next batch of work.
//
// An event is never dropped. There is no replay path - publishing is the only
// consumer action and nothing records that a start event was emitted - so
// discarding one loses it permanently for webhook and event-stream consumers of
// a workflow that is in fact running. When the queue is full the caller
// publishes inline instead, which is no worse than the synchronous behaviour
// this replaced, and shutdown drains what is left.
type startEventDispatcher struct {
	ids     chan string
	repo    testworkflow.Repository
	emitter *event.Emitter
	envID   string
	log     *zap.SugaredLogger

	start   sync.Once
	workers sync.WaitGroup

	// notify is the publish step, injectable so tests can observe it without
	// standing up an event bus.
	notify func(testkube.TestWorkflowExecution)
}

func newStartEventDispatcher(repo testworkflow.Repository, emitter *event.Emitter, envID string, log *zap.SugaredLogger, buffer int) *startEventDispatcher {
	if buffer <= 0 {
		buffer = DefaultDispatchBatchSize * startEventBufferBatches
	}
	d := &startEventDispatcher{
		ids:     make(chan string, buffer),
		repo:    repo,
		emitter: emitter,
		envID:   envID,
		log:     log,
	}
	d.notify = func(execution testkube.TestWorkflowExecution) {
		d.emitter.Notify(testkube.NewEventStartTestWorkflow(&execution, d.envID))
	}
	return d
}

// run starts the workers once.
func (d *startEventDispatcher) run(ctx context.Context) {
	d.start.Do(func() {
		for i := 0; i < startEventWorkers; i++ {
			d.workers.Add(1)
			go d.work(ctx)
		}
	})
}

func (d *startEventDispatcher) work(ctx context.Context) {
	defer d.workers.Done()
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
	d.notify(execution)
}

// enqueue hands ids to the workers, falling back to publishing inline when the
// queue is full rather than discarding the event.
//
// The inline path costs the caller a hydration and a publish, which is exactly
// what the whole dispatch loop used to do for every row. It is the degraded
// case, not the normal one, so it is logged.
func (d *startEventDispatcher) enqueue(ctx context.Context, ids []string) {
	for _, id := range ids {
		select {
		case d.ids <- id:
		default:
			d.log.Warnw("start event queue is full, publishing inline", "id", id)
			d.publish(ctx, id)
		}
	}
}

// drain publishes whatever is still queued at shutdown.
//
// Workers stop on ctx.Done, so without this the queued ids are lost. Bounded,
// because shutdown cannot block indefinitely on a broken event bus.
func (d *startEventDispatcher) drain() {
	ctx, cancel := context.WithTimeout(context.Background(), startEventDrainTimeout)
	defer cancel()

	// Let the workers finish what they already picked up.
	done := make(chan struct{})
	go func() {
		d.workers.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		d.log.Warnw("timed out waiting for start event workers to finish")
		return
	}

	for {
		select {
		case id := <-d.ids:
			d.publish(ctx, id)
		default:
			return
		}
		if ctx.Err() != nil {
			d.log.Warnw("timed out draining start events", "remaining", len(d.ids))
			return
		}
	}
}

// enqueueStartEvents is nil-safe so a Server built as a struct literal, as the
// tests do, still dispatches.
func (s *Server) enqueueStartEvents(ctx context.Context, ids []string) {
	if s.startEvents == nil || len(ids) == 0 {
		return
	}
	s.startEvents.enqueue(ctx, ids)
}
