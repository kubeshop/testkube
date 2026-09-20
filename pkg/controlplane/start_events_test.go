package controlplane

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
)

// recordingRepo counts hydrations and records which ids were asked for.
type recordingRepo struct {
	testworkflow.Repository

	mu      sync.Mutex
	fetched []string
	block   chan struct{}
}

func (r *recordingRepo) Get(_ context.Context, id string) (testkube.TestWorkflowExecution, error) {
	if r.block != nil {
		<-r.block
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fetched = append(r.fetched, id)
	return testkube.TestWorkflowExecution{Id: id}, nil
}

func (r *recordingRepo) ids() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.fetched...)
}

func newTestDispatcher(repo testworkflow.Repository, buffer int) *startEventDispatcher {
	return &startEventDispatcher{
		ids:    make(chan string, buffer),
		repo:   repo,
		log:    zap.NewNop().Sugar(),
		notify: func(testkube.TestWorkflowExecution) {},
		envID:  "env-1",
	}
}

// A full queue must not silently discard start events.
//
// Publishing is the only consumer action and nothing records that an event was
// emitted, so a dropped one is gone for good: webhook and event-stream consumers
// simply never learn that a running workflow started. Falling back to an inline
// publish costs the caller a hydration, which is exactly what the dispatch loop
// used to do for every row anyway.
func TestStartEventDispatcher_PublishesInlineWhenTheQueueIsFull(t *testing.T) {
	repo := &recordingRepo{}
	// No workers are running, so nothing drains the queue.
	d := newTestDispatcher(repo, 2)

	d.enqueue(context.Background(), []string{"a", "b", "c", "d"})

	// Two fit in the buffer; the rest had to be published inline rather than
	// dropped.
	assert.ElementsMatch(t, []string{"c", "d"}, repo.ids(),
		"ids that did not fit must be published, never discarded")
	assert.Len(t, d.ids, 2, "the buffered ids are still queued for the workers")
}

// Nothing may be lost on shutdown either.
func TestStartEventDispatcher_DrainsQueuedEventsOnShutdown(t *testing.T) {
	repo := &recordingRepo{}
	d := newTestDispatcher(repo, 8)

	ctx, cancel := context.WithCancel(context.Background())
	d.run(ctx)

	// Stop the workers first, then queue: this is the shutdown ordering that
	// used to strand ids in the channel.
	cancel()
	time.Sleep(50 * time.Millisecond)
	d.ids <- "queued-1"
	d.ids <- "queued-2"

	d.drain()

	assert.ElementsMatch(t, []string{"queued-1", "queued-2"}, repo.ids(),
		"queued events must be published at shutdown, not abandoned")
}

func TestStartEventDispatcher_WorkersPublishQueuedEvents(t *testing.T) {
	repo := &recordingRepo{}
	d := newTestDispatcher(repo, 8)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	d.run(ctx)

	d.enqueue(ctx, []string{"a", "b", "c"})

	require.Eventually(t, func() bool { return len(repo.ids()) == 3 }, 5*time.Second, 5*time.Millisecond)
	assert.ElementsMatch(t, []string{"a", "b", "c"}, repo.ids())
}

// A Server built as a struct literal has no dispatcher, and must not panic.
func TestEnqueueStartEvents_IsNilSafe(t *testing.T) {
	server := &Server{}
	assert.NotPanics(t, func() {
		server.enqueueStartEvents(context.Background(), []string{"exec-1"})
	})
}
